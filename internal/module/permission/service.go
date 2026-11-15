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

	"jkylin.com/kylin/v2/pkg/errcode"
)

type Service struct {
	PermissionRepository *Repository
}

func NewService(permissionRepository *Repository) *Service {
	return &Service{PermissionRepository: permissionRepository}
}

// List 返回全部启用的权限，不分页。
func (permissionService *Service) List(ctx context.Context) ([]Permission, error) {
	records, err := permissionService.PermissionRepository.List(ctx)
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	return records, nil
}
