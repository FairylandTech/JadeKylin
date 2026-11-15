/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 07:54:30 America/Los_Angeles
 *****************************************************************************/

package service

import (
	"context"
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"jkylin.com/kylin/v2/internal/contracts/requests"
	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/internal/model/valueobject"
	"jkylin.com/kylin/v2/internal/pkg/support"
	"jkylin.com/kylin/v2/internal/repository"
	"jkylin.com/kylin/v2/pkg/errcode"
)

// SessionRevoker 让用户所有登录会话下线。
type SessionRevoker interface {
	RevokeUser(ctx context.Context, userID int64) error
}

type UserService struct {
	UserRepository *repository.UserRepository
	RoleRepository *repository.RoleRepository
	// SessionRevoker 为 nil 时（未启用登录）跳过会话下线。
	SessionRevoker SessionRevoker
	// AuthzCache 为 nil 时（未启用登录）跳过鉴权缓存失效。
	AuthzCache AuthzInvalidator
}

// invalidateAuthz 用户角色、状态变更或删除后删除其权限缓存，下次请求从数据库重新加载。
func (service *UserService) invalidateAuthz(ctx context.Context, userID int64) error {
	if service.AuthzCache == nil {
		return nil
	}
	if err := service.AuthzCache.InvalidateUser(ctx, userID); err != nil {
		return authzSyncError(err)
	}
	return nil
}

func NewUserService(userRepository *repository.UserRepository, roleRepository *repository.RoleRepository) *UserService {
	return &UserService{UserRepository: userRepository, RoleRepository: roleRepository}
}

// revokeSessions 在密码变更、禁用、删除后让用户所有设备下线。
func (service *UserService) revokeSessions(ctx context.Context, userID int64) error {
	if service.SessionRevoker == nil {
		return nil
	}
	if err := service.SessionRevoker.RevokeUser(ctx, userID); err != nil {
		return errcode.Internal.WithMessage("user updated but failed to revoke login sessions").Wrap(err)
	}
	return nil
}

func (service *UserService) Create(ctx context.Context, requestBody *requests.CreateRequest) (*model.User, error) {
	username := strings.TrimSpace(requestBody.Username)
	if username == "" {
		return nil, errcode.InvalidParams.WithMessage("username must not be blank")
	}
	roles, err := service.normalizeRoles(ctx, requestBody.Roles)
	if err != nil {
		return nil, err
	}
	hash, err := hashUserPassword(requestBody.Password)
	if err != nil {
		return nil, err
	}
	record := &model.User{
		Username: username,
		Password: hash,
		Nickname: requestBody.Nickname,
		Remark:   requestBody.Remark,
		Roles:    roles,
	}
	if err := service.UserRepository.Create(ctx, record); err != nil {
		return nil, translateUserError(err)
	}
	record.Password = ""
	return record, nil
}

func (service *UserService) Get(ctx context.Context, id int64) (*model.User, error) {
	record, err := service.UserRepository.Get(ctx, id)
	return record, translateUserError(err)
}

func (service *UserService) List(ctx context.Context, query *requests.ListQuery) ([]model.User, int64, error) {
	query.Normalize()
	if query.Page-1 > math.MaxInt/query.Size {
		return nil, 0, errcode.InvalidParams.WithMessage("page is too large")
	}
	records, total, err := service.UserRepository.List(ctx, strings.TrimSpace(query.Keyword), query.Offset(), query.Size)
	return records, total, translateUserError(err)
}

func (service *UserService) Update(ctx context.Context, id int64, requestBody *requests.UpdateRequest) (*model.User, error) {
	changes := make(map[string]any)
	for column, value := range map[string]*string{
		"nickname": requestBody.Nickname, "remark": requestBody.Remark, "status": requestBody.Status,
	} {
		if value != nil {
			changes[column] = *value
		}
	}
	if requestBody.Username != nil {
		username := strings.TrimSpace(*requestBody.Username)
		if username == "" {
			return nil, errcode.InvalidParams.WithMessage("username must not be blank")
		}
		changes["username"] = username
	}
	if requestBody.Roles != nil {
		roles, err := service.normalizeRoles(ctx, requestBody.Roles)
		if err != nil {
			return nil, err
		}
		changes["roles"] = roles
	}
	if len(changes) == 0 {
		return nil, errcode.InvalidParams.WithMessage("at least one users field is required")
	}
	record, err := service.UserRepository.Update(ctx, id, requestBody.Version, changes)
	if err != nil {
		return nil, translateUserError(err)
	}
	if requestBody.Roles != nil || requestBody.Status != nil {
		if err := service.invalidateAuthz(ctx, id); err != nil {
			return nil, err
		}
	}
	if requestBody.Status != nil && *requestBody.Status != "Y" {
		if err := service.revokeSessions(ctx, id); err != nil {
			return nil, err
		}
	}
	return record, nil
}

// ChangePassword 校验旧密码后写入新密码。
func (service *UserService) ChangePassword(ctx context.Context, id int64, requestBody *requests.ChangePasswordRequest) error {
	current, err := service.UserRepository.GetPasswordHash(ctx, id)
	if err != nil {
		return translateUserError(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(current), []byte(requestBody.OldPassword)) != nil {
		return errcode.InvalidParams.WithMessage("old password is incorrect")
	}
	if requestBody.OldPassword == requestBody.NewPassword {
		return errcode.InvalidParams.WithMessage("new password must differ from the old password")
	}
	hash, err := hashUserPassword(requestBody.NewPassword)
	if err != nil {
		return err
	}
	// 以旧哈希为条件更新，避免并发修改时覆盖他人刚设置的密码。
	if err := service.UserRepository.UpdatePassword(ctx, id, &current, hash); errors.Is(err, support.ErrNotFound) {
		return errcode.Conflict.WithMessage("password was changed concurrently, please retry")
	} else if err != nil {
		return translateUserError(err)
	}
	return service.revokeSessions(ctx, id)
}

// EnsureAdmin 在启用的同名用户不存在时创建初始管理员（角色 admin），已存在则不做任何修改。返回是否新建。
func (service *UserService) EnsureAdmin(ctx context.Context, username, password string) (bool, error) {
	username = strings.TrimSpace(username)
	exists, err := service.UserRepository.ExistsByUsername(ctx, username)
	if err != nil || exists {
		return false, err
	}
	roles := valueobject.RoleList{"admin"}
	_, err = service.Create(ctx, &requests.CreateRequest{Username: username, Password: password, Nickname: "Administrator", Roles: &roles})
	if err != nil && errcode.From(err).Code == errcode.Conflict.Code {
		return false, nil // 并发启动的另一个实例已经创建
	}
	return err == nil, err
}

// ResetPassword 生成随机新密码并返回明文，数据库只保存哈希。
func (service *UserService) ResetPassword(ctx context.Context, id int64) (*requests.ResetPasswordResponse, error) {
	password, err := generatePassword(resetPasswordLength)
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	hash, err := hashUserPassword(password)
	if err != nil {
		return nil, err
	}
	if err := service.UserRepository.UpdatePassword(ctx, id, nil, hash); err != nil {
		return nil, translateUserError(err)
	}
	if err := service.revokeSessions(ctx, id); err != nil {
		return nil, err
	}
	return &requests.ResetPasswordResponse{ID: id, NewPassword: password}, nil
}

func (service *UserService) Delete(ctx context.Context, id int64) error {
	if err := service.UserRepository.Delete(ctx, id); err != nil {
		return translateUserError(err)
	}
	if err := service.invalidateAuthz(ctx, id); err != nil {
		return err
	}
	return service.revokeSessions(ctx, id)
}

// normalizeRoles 去除首尾空白、去重，并确认每个角色都已存在且启用。
func (service *UserService) normalizeRoles(ctx context.Context, roles *valueobject.RoleList) (*valueobject.RoleList, error) {
	if roles == nil {
		return nil, nil
	}
	normalized, err := normalizeCodes(ctx, "roles", *roles, service.RoleRepository.ExistingCodes)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

const (
	resetPasswordLength   = 16
	resetPasswordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789" // 去掉易混淆的 0 O 1 l I
)

func generatePassword(length int) (string, error) {
	limit := big.NewInt(int64(len(resetPasswordAlphabet)))
	password := make([]byte, length)
	for i := range password {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		password[i] = resetPasswordAlphabet[n.Int64()]
	}
	return string(password), nil
}

func hashUserPassword(password string) (string, error) {
	if strings.TrimSpace(password) == "" || len(password) > 72 {
		return "", errcode.InvalidParams.WithMessage("password must be non-blank and at most 72 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", errcode.Internal.Wrap(err)
	}
	return string(hash), nil
}

func translateUserError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, support.ErrNotFound):
		return errcode.NotFound.WithMessage("users not found")
	case errors.Is(err, support.ErrDuplicate):
		return errcode.Conflict.WithMessage("username already exists")
	case errors.Is(err, support.ErrVersionConflict):
		return errcode.Conflict.WithMessage("users version is stale")
	default:
		return errcode.Internal.Wrap(err)
	}
}
