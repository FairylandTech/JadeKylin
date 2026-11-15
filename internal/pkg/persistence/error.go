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

// Package persistence 提供仓储层公共能力：统一错误、常用查询与写入封装。
package persistence

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var (
	ErrNotFound        = errors.New("record not found")
	ErrDuplicate       = errors.New("duplicate record")
	ErrVersionConflict = errors.New("record version conflict")
)

// uniqueViolationCode 是 PostgreSQL 唯一约束冲突的 SQLSTATE。
const uniqueViolationCode = "23505"

// TranslateError 把 GORM / PostgreSQL 错误转换为仓储层统一错误，其余错误原样返回。
func TranslateError(err error) error {
	var postgresError *pgconn.PgError
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return ErrNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey), errors.As(err, &postgresError) && postgresError.Code == uniqueViolationCode:
		return ErrDuplicate
	default:
		return err
	}
}

// WriteError 返回写操作的错误；没有任何记录被影响时返回 ErrNotFound。
func WriteError(result *gorm.DB) error {
	if result.Error != nil {
		return TranslateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
