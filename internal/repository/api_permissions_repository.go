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

package repository

import (
	"context"

	"gorm.io/gorm"
	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/internal/pkg/support"
)

type ApiPermissionRepository struct {
	db *gorm.DB
}

func NewApiPermissionRepository(db *gorm.DB) *ApiPermissionRepository {
	return &ApiPermissionRepository{db: db}
}

// List 返回全部启用的接口放行权限，按路径、方法排序。
func (repo *ApiPermissionRepository) List(ctx context.Context) ([]model.ApiPermission, error) {
	records := make([]model.ApiPermission, 0)
	err := support.EnabledRecords(ctx, repo.db).Order("path, method").Find(&records).Error
	return records, support.DBError(err)
}

func (repo *ApiPermissionRepository) Update(ctx context.Context, id int64, changes map[string]any) (*model.ApiPermission, error) {
	return support.UpdateRecord[model.ApiPermission](support.EnabledRecords(ctx, repo.db), id, changes)
}
