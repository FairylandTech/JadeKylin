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

package permission

import (
	"context"

	"jkylin.com/kylin/v2/internal/pkg/authorization"
	"jkylin.com/kylin/v2/pkg/errcode"
)

// listPermission 是查询全部权限需要的权限 code（为角色分配权限时使用）。
const listPermission = "system.roles"

type Service struct {
	PermissionRepository *Repository
	// Authorizer 判断当前用户能否调用方法；为 nil 时一律拒绝。
	Authorizer *authorization.Authorizer
}

func NewService(permissionRepository *Repository, authorizer *authorization.Authorizer) *Service {
	return &Service{PermissionRepository: permissionRepository, Authorizer: authorizer}
}

// List 返回全部启用的权限，不分页。
func (permissionService *Service) List(ctx context.Context) ([]Permission, error) {
	if err := permissionService.Authorizer.Require(ctx, listPermission); err != nil {
		return nil, err
	}
	records, err := permissionService.PermissionRepository.List(ctx)
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	return records, nil
}
