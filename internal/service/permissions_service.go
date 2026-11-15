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

	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/internal/repository"
	"jkylin.com/kylin/v2/pkg/errcode"
)

type PermissionService struct {
	PermissionRepository *repository.PermissionRepository
}

func NewPermissionService(permissionRepository *repository.PermissionRepository) *PermissionService {
	return &PermissionService{PermissionRepository: permissionRepository}
}

// List 返回全部启用的权限，不分页。
func (service *PermissionService) List(ctx context.Context) ([]model.Permission, error) {
	records, err := service.PermissionRepository.List(ctx)
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	return records, nil
}
