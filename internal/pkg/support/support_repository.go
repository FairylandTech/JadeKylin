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

package support

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound        = errors.New("record not found")
	ErrDuplicate       = errors.New("duplicate record")
	ErrVersionConflict = errors.New("record version conflict")
)

func DBError(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return ErrNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey), errors.As(err, &pgErr) && pgErr.Code == "23505":
		return ErrDuplicate
	default:
		return err
	}
}

func WriteError(result *gorm.DB) error {
	if result.Error != nil {
		return DBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func EnabledRecords(ctx context.Context, db *gorm.DB) *gorm.DB {
	return db.WithContext(ctx).Where("enabled = ?", "Y")
}

func FirstRecord[T any](query *gorm.DB) (*T, error) {
	var record T
	if err := query.First(&record).Error; err != nil {
		return nil, DBError(err)
	}
	return &record, nil
}

func ListRecords[T any](query *gorm.DB, offset, limit int) ([]T, int64, error) {
	var record T
	query = query.Model(&record).Session(&gorm.Session{})
	records := make([]T, 0, limit)
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&records).Error; err != nil {
		return nil, 0, DBError(err)
	}
	// 最后一页（未取满且确实落在数据范围内）时总数可以直接推出来，省一次 COUNT。
	if n := len(records); n < limit && (n > 0 || offset == 0) {
		return records, int64(offset + n), nil
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, DBError(err)
	}
	return records, total, nil
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// ContainsPattern 把关键字转成 LIKE/ILIKE 的 "包含" 模式，并转义 \ % _（配合 ESCAPE '\'）。
func ContainsPattern(keyword string) string {
	return "%" + likeEscaper.Replace(keyword) + "%"
}

func UpdateRecord[T any](query *gorm.DB, id int64, changes map[string]any) (*T, error) {
	var record T
	result := query.Model(&record).Where("id = ?", id).Clauses(clause.Returning{}).Updates(changes)
	if err := WriteError(result); err != nil {
		return nil, err
	}
	return &record, nil
}

// ExistingCodes 返回 codes 中已启用记录的 code。
func ExistingCodes[T any](ctx context.Context, db *gorm.DB, codes []string) ([]string, error) {
	found := make([]string, 0, len(codes))
	if len(codes) == 0 {
		return found, nil
	}
	var record T
	err := EnabledRecords(ctx, db).Model(&record).Where("code IN ?", codes).Pluck("code", &found).Error
	return found, DBError(err)
}
