/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 11:53:35 America/Los_Angeles
 *****************************************************************************/

package persistence

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EnabledRecords 返回只匹配启用记录（enabled = 'Y'）的查询。
func EnabledRecords(ctx context.Context, database *gorm.DB) *gorm.DB {
	return database.WithContext(ctx).Where("enabled = ?", "Y")
}

// FirstRecord 返回查询命中的第一条记录。
func FirstRecord[Model any](query *gorm.DB) (*Model, error) {
	var record Model
	if err := query.First(&record).Error; err != nil {
		return nil, TranslateError(err)
	}
	return &record, nil
}

// ListRecords 分页查询并返回总数。
func ListRecords[Model any](query *gorm.DB, offset, limit int) ([]Model, int64, error) {
	var record Model
	query = query.Model(&record).Session(&gorm.Session{})
	records := make([]Model, 0, limit)
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&records).Error; err != nil {
		return nil, 0, TranslateError(err)
	}
	// 最后一页（未取满且确实落在数据范围内）时总数可以直接推出来，省一次 COUNT。
	if count := len(records); count < limit && (count > 0 || offset == 0) {
		return records, int64(offset + count), nil
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}
	return records, total, nil
}

// UpdateRecord 按 id 更新并返回更新后的记录。
func UpdateRecord[Model any](query *gorm.DB, id int64, changes map[string]any) (*Model, error) {
	var record Model
	result := query.Model(&record).Where("id = ?", id).Clauses(clause.Returning{}).Updates(changes)
	if err := WriteError(result); err != nil {
		return nil, err
	}
	return &record, nil
}

// DisableRecord 软删除：把 id 对应的记录标记为停用（enabled = 'N'）。
func DisableRecord[Model any](query *gorm.DB, id int64) error {
	var record Model
	result := query.Model(&record).Where("id = ?", id).Update("enabled", "N")
	return WriteError(result)
}

// ExistingCodes 返回 codes 中已启用记录的 code。
func ExistingCodes[Model any](ctx context.Context, database *gorm.DB, codes []string) ([]string, error) {
	found := make([]string, 0, len(codes))
	if len(codes) == 0 {
		return found, nil
	}
	var record Model
	err := EnabledRecords(ctx, database).Model(&record).Where("code IN ?", codes).Pluck("code", &found).Error
	return found, TranslateError(err)
}
