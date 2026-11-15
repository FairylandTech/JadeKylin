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

type RoleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

func (repo *RoleRepository) Create(ctx context.Context, record *model.Role) error {
	return support.DBError(repo.db.WithContext(ctx).Create(record).Error)
}

func (repo *RoleRepository) Get(ctx context.Context, id int64) (*model.Role, error) {
	return support.FirstRecord[model.Role](support.EnabledRecords(ctx, repo.db).Where("id = ?", id))
}

func (repo *RoleRepository) GetByCode(ctx context.Context, code string) (*model.Role, error) {
	return support.FirstRecord[model.Role](support.EnabledRecords(ctx, repo.db).Where("code = ?", code))
}

func (repo *RoleRepository) List(ctx context.Context, keyword string, offset, limit int) ([]model.Role, int64, error) {
	query := support.EnabledRecords(ctx, repo.db)
	if keyword != "" {
		query = query.Where("(strpos(lower(code), lower(?)) > 0 OR strpos(lower(name), lower(?)) > 0)", keyword, keyword)
	}
	query = query.Order("id")
	return support.ListRecords[model.Role](query, offset, limit)
}

func (repo *RoleRepository) Update(ctx context.Context, id int64, changes map[string]any) (*model.Role, error) {
	return support.UpdateRecord[model.Role](support.EnabledRecords(ctx, repo.db), id, changes)
}

func (repo *RoleRepository) Delete(ctx context.Context, id int64) error {
	result := support.EnabledRecords(ctx, repo.db).Model(&model.Role{}).Where("id = ?", id).Update("enabled", "N")
	return support.WriteError(result)
}

// ExistingCodes 返回 codes 中已启用角色的 code。
func (repo *RoleRepository) ExistingCodes(ctx context.Context, codes []string) ([]string, error) {
	return support.ExistingCodes[model.Role](ctx, repo.db, codes)
}
