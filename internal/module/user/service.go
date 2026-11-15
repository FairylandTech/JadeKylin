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

package user

import (
	"context"
	"errors"
	"strings"

	"jkylin.com/kylin/v2/internal/module/role"
	"jkylin.com/kylin/v2/internal/pkg/authorization"
	"jkylin.com/kylin/v2/internal/pkg/credential"
	"jkylin.com/kylin/v2/internal/pkg/identity"
	"jkylin.com/kylin/v2/internal/pkg/pagination"
	"jkylin.com/kylin/v2/internal/pkg/persistence"
	"jkylin.com/kylin/v2/internal/pkg/validation"
	"jkylin.com/kylin/v2/internal/pkg/valueobject"
	"jkylin.com/kylin/v2/pkg/errcode"
)

const (
	// administratorRoleCode 是初始管理员拥有的角色 code。
	administratorRoleCode = "admin"
	// administratorNickname 是初始管理员的昵称。
	administratorNickname = "Administrator"
	// resetPasswordLength 是重置密码时生成的随机密码长度。
	resetPasswordLength = 16
	// managementPermission 是用户管理接口需要的权限 code。
	managementPermission = "system.users"
)

// SessionRevoker 让用户所有登录会话下线。
type SessionRevoker interface {
	RevokeUser(ctx context.Context, userID int64) error
}

type Service struct {
	UserRepository *Repository
	RoleRepository *role.Repository
	// SessionRevoker 为 nil 时（未启用登录）跳过会话下线。
	SessionRevoker SessionRevoker
	// Authorizer 判断当前用户能否调用管理方法；为 nil 时（未启用登录）管理方法一律拒绝，缓存失效直接跳过。
	Authorizer *authorization.Authorizer
}

func NewService(userRepository *Repository, roleRepository *role.Repository, authorizer *authorization.Authorizer) *Service {
	return &Service{UserRepository: userRepository, RoleRepository: roleRepository, Authorizer: authorizer}
}

// revokeSessions 在密码变更、禁用、删除后让用户所有设备下线。
func (userService *Service) revokeSessions(ctx context.Context, userID int64) error {
	if userService.SessionRevoker == nil {
		return nil
	}
	if err := userService.SessionRevoker.RevokeUser(ctx, userID); err != nil {
		return errcode.Internal.WithMessage("user updated but failed to revoke login sessions").Wrap(err)
	}
	return nil
}

func (userService *Service) Create(ctx context.Context, requestBody *CreateRequest) (*User, error) {
	if err := userService.Authorizer.Require(ctx, managementPermission); err != nil {
		return nil, err
	}
	return userService.create(ctx, requestBody)
}

// create 创建用户，不做权限判断（EnsureAdmin 在启动时复用）。
func (userService *Service) create(ctx context.Context, requestBody *CreateRequest) (*User, error) {
	username, err := validation.RequireText("username", requestBody.Username)
	if err != nil {
		return nil, err
	}
	roles, err := userService.normalizeRoles(ctx, requestBody.Roles)
	if err != nil {
		return nil, err
	}
	hash, err := credential.HashPassword(requestBody.Password)
	if err != nil {
		return nil, err
	}
	record := &User{
		Username: username,
		Password: hash,
		Nickname: requestBody.Nickname,
		Remark:   requestBody.Remark,
		Roles:    roles,
	}
	if err := userService.UserRepository.Create(ctx, record); err != nil {
		return nil, TranslateError(err)
	}
	record.Password = ""
	return record, nil
}

func (userService *Service) Get(ctx context.Context, id int64) (*User, error) {
	if err := userService.Authorizer.Require(ctx, managementPermission); err != nil {
		return nil, err
	}
	record, err := userService.UserRepository.Get(ctx, id)
	return record, TranslateError(err)
}

func (userService *Service) List(ctx context.Context, query *pagination.KeywordQuery) ([]User, int64, error) {
	if err := userService.Authorizer.Require(ctx, managementPermission); err != nil {
		return nil, 0, err
	}
	if err := query.Validate(); err != nil {
		return nil, 0, err
	}
	records, total, err := userService.UserRepository.List(ctx, strings.TrimSpace(query.Keyword), query.Offset(), query.Size)
	return records, total, TranslateError(err)
}

func (userService *Service) Update(ctx context.Context, id int64, requestBody *UpdateRequest) (*User, error) {
	if err := userService.Authorizer.Require(ctx, managementPermission); err != nil {
		return nil, err
	}
	changes := make(map[string]any)
	for column, value := range map[string]*string{
		"nickname": requestBody.Nickname,
		"remark":   requestBody.Remark,
		"status":   requestBody.Status,
	} {
		if value != nil {
			changes[column] = *value
		}
	}
	if requestBody.Username != nil {
		username, err := validation.RequireText("username", *requestBody.Username)
		if err != nil {
			return nil, err
		}
		changes["username"] = username
	}
	if requestBody.Roles != nil {
		roles, err := userService.normalizeRoles(ctx, requestBody.Roles)
		if err != nil {
			return nil, err
		}
		changes["roles"] = roles
	}
	if len(changes) == 0 {
		return nil, errcode.InvalidParameters.WithMessage("at least one users field is required")
	}
	record, err := userService.UserRepository.Update(ctx, id, requestBody.Version, changes)
	if err != nil {
		return nil, TranslateError(err)
	}
	if requestBody.Roles != nil || requestBody.Status != nil {
		if err := userService.Authorizer.InvalidateUser(ctx, id); err != nil {
			return nil, err
		}
	}
	if requestBody.Status != nil && *requestBody.Status != "Y" {
		if err := userService.revokeSessions(ctx, id); err != nil {
			return nil, err
		}
	}
	return record, nil
}

// ChangePassword 管理员修改指定用户的密码（需要 system.users）。
func (userService *Service) ChangePassword(ctx context.Context, id int64, requestBody *ChangePasswordRequest) error {
	if err := userService.Authorizer.Require(ctx, managementPermission); err != nil {
		return err
	}
	return userService.changePassword(ctx, id, requestBody)
}

// ChangeOwnPassword 当前登录用户修改自己的密码，只要求登录，不需要权限。
func (userService *Service) ChangeOwnPassword(ctx context.Context, requestBody *ChangePasswordRequest) error {
	principal, err := identity.Require(ctx)
	if err != nil {
		return err
	}
	return userService.changePassword(ctx, principal.UserID, requestBody)
}

// changePassword 校验旧密码后写入新密码。
func (userService *Service) changePassword(ctx context.Context, id int64, requestBody *ChangePasswordRequest) error {
	current, err := userService.UserRepository.GetPasswordHash(ctx, id)
	if err != nil {
		return TranslateError(err)
	}
	if !credential.MatchesPassword(current, requestBody.OldPassword) {
		return errcode.InvalidParameters.WithMessage("old password is incorrect")
	}
	if requestBody.OldPassword == requestBody.NewPassword {
		return errcode.InvalidParameters.WithMessage("new password must differ from the old password")
	}
	hash, err := credential.HashPassword(requestBody.NewPassword)
	if err != nil {
		return err
	}
	// 以旧哈希为条件更新，避免并发修改时覆盖他人刚设置的密码。
	if err := userService.UserRepository.UpdatePassword(ctx, id, &current, hash); errors.Is(err, persistence.ErrNotFound) {
		return errcode.Conflict.WithMessage("password was changed concurrently, please retry")
	} else if err != nil {
		return TranslateError(err)
	}
	return userService.revokeSessions(ctx, id)
}

// EnsureAdmin 在启用的同名用户不存在时创建初始管理员（角色 admin），已存在则不做任何修改。返回是否新建。
func (userService *Service) EnsureAdmin(ctx context.Context, username, password string) (bool, error) {
	username = strings.TrimSpace(username)
	exists, err := userService.UserRepository.ExistsByUsername(ctx, username)
	if err != nil || exists {
		return false, err
	}
	roles := valueobject.RoleList{administratorRoleCode}
	_, err = userService.create(ctx, &CreateRequest{
		Username: username,
		Password: password,
		Nickname: administratorNickname,
		Roles:    &roles,
	})
	if err != nil && errcode.From(err).Code == errcode.Conflict.Code {
		return false, nil // 并发启动的另一个实例已经创建
	}
	return err == nil, err
}

// ResetPassword 生成随机新密码并返回明文，数据库只保存哈希。
func (userService *Service) ResetPassword(ctx context.Context, id int64) (*ResetPasswordResponse, error) {
	if err := userService.Authorizer.Require(ctx, managementPermission); err != nil {
		return nil, err
	}
	password, err := credential.GeneratePassword(resetPasswordLength)
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	hash, err := credential.HashPassword(password)
	if err != nil {
		return nil, err
	}
	if err := userService.UserRepository.UpdatePassword(ctx, id, nil, hash); err != nil {
		return nil, TranslateError(err)
	}
	if err := userService.revokeSessions(ctx, id); err != nil {
		return nil, err
	}
	return &ResetPasswordResponse{ID: id, NewPassword: password}, nil
}

func (userService *Service) Delete(ctx context.Context, id int64) error {
	if err := userService.Authorizer.Require(ctx, managementPermission); err != nil {
		return err
	}
	if err := userService.UserRepository.Delete(ctx, id); err != nil {
		return TranslateError(err)
	}
	if err := userService.Authorizer.InvalidateUser(ctx, id); err != nil {
		return err
	}
	return userService.revokeSessions(ctx, id)
}

// normalizeRoles 去除首尾空白、去重，并确认每个角色都已存在且启用。
func (userService *Service) normalizeRoles(ctx context.Context, roles *valueobject.RoleList) (*valueobject.RoleList, error) {
	if roles == nil {
		return nil, nil
	}
	normalized, err := validation.NormalizeCodes(ctx, "roles", *roles, userService.RoleRepository.ExistingCodes)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

// TranslateError 把仓储层错误转换为对外的业务错误（auth 模块读取当前用户时复用）。
func TranslateError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, persistence.ErrNotFound):
		return errcode.NotFound.WithMessage("users not found")
	case errors.Is(err, persistence.ErrDuplicate):
		return errcode.Conflict.WithMessage("username already exists")
	case errors.Is(err, persistence.ErrVersionConflict):
		return errcode.Conflict.WithMessage("users version is stale")
	default:
		return errcode.Internal.Wrap(err)
	}
}
