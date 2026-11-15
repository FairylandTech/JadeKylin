/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 11:42:18 America/Los_Angeles
 *****************************************************************************/

package repository

import (
	"context"

	"gorm.io/gorm"
	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/internal/pkg/support"
)

type PermissionRepository struct {
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

func (repo *PermissionRepository) Create(ctx context.Context, record *model.Permission) error {
	return support.DBError(repo.db.WithContext(ctx).Create(record).Error)
}

func (repo *PermissionRepository) Get(ctx context.Context, id int64) (*model.Permission, error) {
	return support.FirstRecord[model.Permission](support.EnabledRecords(ctx, repo.db).Where("id = ?", id))
}

func (repo *PermissionRepository) GetByCode(ctx context.Context, code string) (*model.Permission, error) {
	return support.FirstRecord[model.Permission](support.EnabledRecords(ctx, repo.db).Where("code = ?", code))
}

// List 返回全部启用的权限（不分页），按分类、id 排序，方便前端分组展示。
func (repo *PermissionRepository) List(ctx context.Context) ([]model.Permission, error) {
	records := make([]model.Permission, 0)
	err := support.EnabledRecords(ctx, repo.db).Order("category, id").Find(&records).Error
	return records, support.DBError(err)
}

// ExistingCodes 返回 codes 中已启用权限的 code。
func (repo *PermissionRepository) ExistingCodes(ctx context.Context, codes []string) ([]string, error) {
	return support.ExistingCodes[model.Permission](ctx, repo.db, codes)
}

func (repo *PermissionRepository) Update(ctx context.Context, id int64, changes map[string]any) (*model.Permission, error) {
	return support.UpdateRecord[model.Permission](support.EnabledRecords(ctx, repo.db), id, changes)
}

func (repo *PermissionRepository) Delete(ctx context.Context, id int64) error {
	result := support.EnabledRecords(ctx, repo.db).Model(&model.Permission{}).
		Where("id = ?", id).
		Update("enabled", "N")
	return support.WriteError(result)
}
