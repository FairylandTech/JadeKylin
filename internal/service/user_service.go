/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

package service

import (
	"context"
	"errors"
	"math"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"jkylin.com/kylin/v2/internal/contracts/requests"
	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/internal/repository"
	"jkylin.com/kylin/v2/pkg/errcode"
)

type UserService struct {
	UserRepository *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{UserRepository: repo}
}

func (service *UserService) Create(ctx context.Context, req *requests.CreateRequest) (*model.User, error) {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return nil, errcode.InvalidParams.WithMessage("username must not be blank")
	}
	hash, err := hashUserPassword(req.Password)
	if err != nil {
		return nil, err
	}
	record := &model.User{
		Username: username, Password: hash, Nickname: req.Nickname, Remark: req.Remark,
		Status: req.Status, Enabled: req.Enabled, Roles: req.Roles,
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

func (service *UserService) Update(ctx context.Context, id int64, req *requests.UpdateRequest) (*model.User, error) {
	changes := make(map[string]any)
	for column, value := range map[string]*string{
		"nickname": req.Nickname, "remark": req.Remark, "status": req.Status, "enabled": req.Enabled,
	} {
		if value != nil {
			changes[column] = *value
		}
	}
	if req.Username != nil {
		username := strings.TrimSpace(*req.Username)
		if username == "" {
			return nil, errcode.InvalidParams.WithMessage("username must not be blank")
		}
		changes["username"] = username
	}
	if req.Password != nil {
		hash, err := hashUserPassword(*req.Password)
		if err != nil {
			return nil, err
		}
		changes["password"] = hash
	}
	if req.Roles != nil {
		changes["roles"] = req.Roles
	}
	if len(changes) == 0 {
		return nil, errcode.InvalidParams.WithMessage("at least one user field is required")
	}
	record, err := service.UserRepository.Update(ctx, id, req.Version, changes)
	return record, translateUserError(err)
}

func (service *UserService) Delete(ctx context.Context, id int64) error {
	return translateUserError(service.UserRepository.Delete(ctx, id))
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
	case errors.Is(err, repository.ErrNotFound):
		return errcode.NotFound.WithMessage("user not found")
	case errors.Is(err, repository.ErrDuplicate):
		return errcode.Conflict.WithMessage("username already exists")
	case errors.Is(err, repository.ErrVersionConflict):
		return errcode.Conflict.WithMessage("user version is stale")
	default:
		return errcode.Internal.Wrap(err)
	}
}
