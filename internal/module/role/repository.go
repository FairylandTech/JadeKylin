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

package role

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

func (roleRepository *Repository) Create(ctx context.Context, record *Role) error {
	return persistence.TranslateError(roleRepository.database.WithContext(ctx).Create(record).Error)
}

func (roleRepository *Repository) Get(ctx context.Context, id int64) (*Role, error) {
	return persistence.FirstRecord[Role](persistence.EnabledRecords(ctx, roleRepository.database).Where("id = ?", id))
}

func (roleRepository *Repository) GetByCode(ctx context.Context, code string) (*Role, error) {
	return persistence.FirstRecord[Role](persistence.EnabledRecords(ctx, roleRepository.database).Where("code = ?", code))
}

func (roleRepository *Repository) List(ctx context.Context, keyword string, offset, limit int) ([]Role, int64, error) {
	query := persistence.EnabledRecords(ctx, roleRepository.database)
	if keyword != "" {
		query = query.Where("(strpos(lower(code), lower(?)) > 0 OR strpos(lower(name), lower(?)) > 0)", keyword, keyword)
	}
	query = query.Order("id")
	return persistence.ListRecords[Role](query, offset, limit)
}

func (roleRepository *Repository) Update(ctx context.Context, id int64, changes map[string]any) (*Role, error) {
	return persistence.UpdateRecord[Role](persistence.EnabledRecords(ctx, roleRepository.database), id, changes)
}

func (roleRepository *Repository) Delete(ctx context.Context, id int64) error {
	return persistence.DisableRecord[Role](persistence.EnabledRecords(ctx, roleRepository.database), id)
}

// ExistingCodes 返回 codes 中已启用角色的 code。
func (roleRepository *Repository) ExistingCodes(ctx context.Context, codes []string) ([]string, error) {
	return persistence.ExistingCodes[Role](ctx, roleRepository.database, codes)
}
