/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-15 01:30:00 America/Los_Angeles
 *****************************************************************************/

package apipermission

import (
	"context"

	"gorm.io/gorm"
	"jkylin.com/kylin/v2/internal/pkg/persistence"
)

type Repository struct {
	database *gorm.DB
}

func NewRepository(database *gorm.DB) *Repository {
	return &Repository{database: database}
}

// List 返回全部启用的接口放行权限，按路径、方法排序。
func (apiPermissionRepository *Repository) List(ctx context.Context) ([]ApiPermission, error) {
	records := make([]ApiPermission, 0)
	err := persistence.EnabledRecords(ctx, apiPermissionRepository.database).Order("path, method").Find(&records).Error
	return records, persistence.TranslateError(err)
}

func (apiPermissionRepository *Repository) Update(ctx context.Context, id int64, changes map[string]any) (*ApiPermission, error) {
	return persistence.UpdateRecord[ApiPermission](persistence.EnabledRecords(ctx, apiPermissionRepository.database), id, changes)
}
