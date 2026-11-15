/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 21:30:00 America/Los_Angeles
 *****************************************************************************/

package role

import (
	"context"
	"errors"
	"strings"

	"jkylin.com/kylin/v2/internal/module/permission"
	"jkylin.com/kylin/v2/internal/pkg/authorization"
	"jkylin.com/kylin/v2/internal/pkg/pagination"
	"jkylin.com/kylin/v2/internal/pkg/persistence"
	"jkylin.com/kylin/v2/internal/pkg/validation"
	"jkylin.com/kylin/v2/internal/pkg/valueobject"
	"jkylin.com/kylin/v2/pkg/errcode"
)

type Service struct {
	RoleRepository       *Repository
	PermissionRepository *permission.Repository
	// AuthorizationCache 为 nil 时（未启用登录）跳过鉴权缓存失效。
	AuthorizationCache authorization.Invalidator
}

func NewService(roleRepository *Repository, permissionRepository *permission.Repository) *Service {
	return &Service{RoleRepository: roleRepository, PermissionRepository: permissionRepository}
}

// invalidateAuthorization 角色变更可能影响任意用户，删除所有用户的权限缓存。
// 新建角色也需要失效：被删除的角色 code 可能仍留在用户的 roles 中，重建同 code 角色会让这些用户获得权限。
func (roleService *Service) invalidateAuthorization(ctx context.Context) error {
	if roleService.AuthorizationCache == nil {
		return nil
	}
	if err := roleService.AuthorizationCache.InvalidateAllUsers(ctx); err != nil {
		return authorization.CacheRefreshError(err)
	}
	return nil
}

func (roleService *Service) Create(ctx context.Context, requestBody *CreateRequest) (*Role, error) {
	code, err := validation.RequireText("code", requestBody.Code)
	if err != nil {
		return nil, err
	}
	name, err := validation.RequireText("name", requestBody.Name)
	if err != nil {
		return nil, err
	}
	permissions, err := roleService.normalizePermissions(ctx, requestBody.Permissions)
	if err != nil {
		return nil, err
	}
	record := &Role{
		Code:        code,
		Name:        name,
		Description: requestBody.Description,
		Permissions: permissions,
	}
	if err := roleService.RoleRepository.Create(ctx, record); err != nil {
		return nil, translateError(err)
	}
	if err := roleService.invalidateAuthorization(ctx); err != nil {
		return nil, err
	}
	return record, nil
}

func (roleService *Service) Get(ctx context.Context, id int64) (*Role, error) {
	record, err := roleService.RoleRepository.Get(ctx, id)
	return record, translateError(err)
}

func (roleService *Service) List(ctx context.Context, query *pagination.KeywordQuery) ([]Role, int64, error) {
	if err := query.Validate(); err != nil {
		return nil, 0, err
	}
	records, total, err := roleService.RoleRepository.List(ctx, strings.TrimSpace(query.Keyword), query.Offset(), query.Size)
	return records, total, translateError(err)
}

func (roleService *Service) Update(ctx context.Context, id int64, requestBody *UpdateRequest) (*Role, error) {
	changes := make(map[string]any)
	if requestBody.Name != nil {
		name, err := validation.RequireText("name", *requestBody.Name)
		if err != nil {
			return nil, err
		}
		changes["name"] = name
	}
	if requestBody.Description != nil {
		changes["description"] = *requestBody.Description
	}
	if requestBody.Permissions != nil {
		permissions, err := roleService.normalizePermissions(ctx, requestBody.Permissions)
		if err != nil {
			return nil, err
		}
		changes["permissions"] = permissions
	}
	if len(changes) == 0 {
		return nil, errcode.InvalidParameters.WithMessage("at least one roles field is required")
	}
	record, err := roleService.RoleRepository.Update(ctx, id, changes)
	if err != nil {
		return nil, translateError(err)
	}
	if err := roleService.invalidateAuthorization(ctx); err != nil {
		return nil, err
	}
	return record, nil
}

func (roleService *Service) Delete(ctx context.Context, id int64) error {
	if err := roleService.RoleRepository.Delete(ctx, id); err != nil {
		return translateError(err)
	}
	return roleService.invalidateAuthorization(ctx)
}

// normalizePermissions 去除首尾空白、去重，并确认每个权限都已存在且启用。
func (roleService *Service) normalizePermissions(ctx context.Context, permissions *valueobject.PermissionList) (*valueobject.PermissionList, error) {
	if permissions == nil {
		return nil, nil
	}
	normalized, err := validation.NormalizeCodes(ctx, "permissions", *permissions, roleService.PermissionRepository.ExistingCodes)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

// translateError 把仓储层错误转换为对外的业务错误。
func translateError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, persistence.ErrNotFound):
		return errcode.NotFound.WithMessage("roles not found")
	case errors.Is(err, persistence.ErrDuplicate):
		return errcode.Conflict.WithMessage("role code or name already exists")
	default:
		return errcode.Internal.Wrap(err)
	}
}
