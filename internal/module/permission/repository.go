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

package permission

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

func (permissionRepository *Repository) Create(ctx context.Context, record *Permission) error {
	return persistence.TranslateError(permissionRepository.database.WithContext(ctx).Create(record).Error)
}

func (permissionRepository *Repository) Get(ctx context.Context, id int64) (*Permission, error) {
	return persistence.FirstRecord[Permission](persistence.EnabledRecords(ctx, permissionRepository.database).Where("id = ?", id))
}

func (permissionRepository *Repository) GetByCode(ctx context.Context, code string) (*Permission, error) {
	return persistence.FirstRecord[Permission](persistence.EnabledRecords(ctx, permissionRepository.database).Where("code = ?", code))
}

// List 返回全部启用的权限（不分页），按分类、id 排序，方便前端分组展示。
func (permissionRepository *Repository) List(ctx context.Context) ([]Permission, error) {
	records := make([]Permission, 0)
	err := persistence.EnabledRecords(ctx, permissionRepository.database).Order("category, id").Find(&records).Error
	return records, persistence.TranslateError(err)
}

// ExistingCodes 返回 codes 中已启用权限的 code。
func (permissionRepository *Repository) ExistingCodes(ctx context.Context, codes []string) ([]string, error) {
	return persistence.ExistingCodes[Permission](ctx, permissionRepository.database, codes)
}

func (permissionRepository *Repository) Update(ctx context.Context, id int64, changes map[string]any) (*Permission, error) {
	return persistence.UpdateRecord[Permission](persistence.EnabledRecords(ctx, permissionRepository.database), id, changes)
}

func (permissionRepository *Repository) Delete(ctx context.Context, id int64) error {
	return persistence.DisableRecord[Permission](persistence.EnabledRecords(ctx, permissionRepository.database), id)
}
