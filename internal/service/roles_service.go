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

package service

import (
	"context"
	"errors"
	"math"
	"strings"

	"jkylin.com/kylin/v2/internal/contracts/requests"
	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/internal/model/valueobject"
	"jkylin.com/kylin/v2/internal/pkg/support"
	"jkylin.com/kylin/v2/internal/repository"
	"jkylin.com/kylin/v2/pkg/errcode"
)

type RoleService struct {
	RoleRepository       *repository.RoleRepository
	PermissionRepository *repository.PermissionRepository
	// AuthzCache 为 nil 时（未启用登录）跳过鉴权缓存失效。
	AuthzCache AuthzInvalidator
}

// invalidateAuthz 角色变更可能影响任意用户，删除所有用户的权限缓存。
// 新建角色也需要失效：被删除的角色 code 可能仍留在用户的 roles 中，重建同 code 角色会让这些用户获得权限。
func (service *RoleService) invalidateAuthz(ctx context.Context) error {
	if service.AuthzCache == nil {
		return nil
	}
	if err := service.AuthzCache.InvalidateAllUsers(ctx); err != nil {
		return authzSyncError(err)
	}
	return nil
}

func NewRoleService(roleRepository *repository.RoleRepository, permissionRepository *repository.PermissionRepository) *RoleService {
	return &RoleService{RoleRepository: roleRepository, PermissionRepository: permissionRepository}
}

func (service *RoleService) Create(ctx context.Context, requestBody *requests.CreateRoleRequest) (*model.Role, error) {
	code, err := requireText("code", requestBody.Code)
	if err != nil {
		return nil, err
	}
	name, err := requireText("name", requestBody.Name)
	if err != nil {
		return nil, err
	}
	permissions, err := service.normalizePermissions(ctx, requestBody.Permissions)
	if err != nil {
		return nil, err
	}
	record := &model.Role{
		Code:        code,
		Name:        name,
		Description: requestBody.Description,
		Permissions: permissions,
	}
	if err := service.RoleRepository.Create(ctx, record); err != nil {
		return nil, translateRoleError(err)
	}
	if err := service.invalidateAuthz(ctx); err != nil {
		return nil, err
	}
	return record, nil
}

func (service *RoleService) Get(ctx context.Context, id int64) (*model.Role, error) {
	record, err := service.RoleRepository.Get(ctx, id)
	return record, translateRoleError(err)
}

func (service *RoleService) List(ctx context.Context, query *requests.ListQuery) ([]model.Role, int64, error) {
	query.Normalize()
	if query.Page-1 > math.MaxInt/query.Size {
		return nil, 0, errcode.InvalidParams.WithMessage("page is too large")
	}
	records, total, err := service.RoleRepository.List(ctx, strings.TrimSpace(query.Keyword), query.Offset(), query.Size)
	return records, total, translateRoleError(err)
}

func (service *RoleService) Update(ctx context.Context, id int64, requestBody *requests.UpdateRoleRequest) (*model.Role, error) {
	changes := make(map[string]any)
	if requestBody.Name != nil {
		name, err := requireText("name", *requestBody.Name)
		if err != nil {
			return nil, err
		}
		changes["name"] = name
	}
	if requestBody.Description != nil {
		changes["description"] = *requestBody.Description
	}
	if requestBody.Permissions != nil {
		permissions, err := service.normalizePermissions(ctx, requestBody.Permissions)
		if err != nil {
			return nil, err
		}
		changes["permissions"] = permissions
	}
	if len(changes) == 0 {
		return nil, errcode.InvalidParams.WithMessage("at least one roles field is required")
	}
	record, err := service.RoleRepository.Update(ctx, id, changes)
	if err != nil {
		return nil, translateRoleError(err)
	}
	if err := service.invalidateAuthz(ctx); err != nil {
		return nil, err
	}
	return record, nil
}

func (service *RoleService) Delete(ctx context.Context, id int64) error {
	if err := service.RoleRepository.Delete(ctx, id); err != nil {
		return translateRoleError(err)
	}
	return service.invalidateAuthz(ctx)
}

// normalizePermissions 去除首尾空白、去重，并确认每个权限都已存在且启用。
func (service *RoleService) normalizePermissions(ctx context.Context, permissions *valueobject.PermissionList) (*valueobject.PermissionList, error) {
	if permissions == nil {
		return nil, nil
	}
	normalized, err := normalizeCodes(ctx, "permissions", *permissions, service.PermissionRepository.ExistingCodes)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

func translateRoleError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, support.ErrNotFound):
		return errcode.NotFound.WithMessage("roles not found")
	case errors.Is(err, support.ErrDuplicate):
		return errcode.Conflict.WithMessage("role code or name already exists")
	default:
		return errcode.Internal.Wrap(err)
	}
}
