/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 09:57:11 America/Los_Angeles
 *****************************************************************************/

// Package pagination 提供分页查询参数。
package pagination

import (
	"math"

	"jkylin.com/kylin/v2/pkg/errcode"
)

const (
	defaultPage = 1
	defaultSize = 10
)

// Query 是分页查询参数。
type Query struct {
	Page int `form:"page" binding:"omitempty,min=1"`
	Size int `form:"size" binding:"omitempty,min=1,max=100"`
}

// Normalize 为缺省的页码、页大小补上默认值。
func (query *Query) Normalize() {
	if query.Page <= 0 {
		query.Page = defaultPage
	}
	if query.Size <= 0 {
		query.Size = defaultSize
	}
}

// Offset 返回当前页的起始偏移量。
func (query *Query) Offset() int {
	return (query.Page - 1) * query.Size
}

// Validate 补全默认值，并拒绝会让偏移量溢出的页码。
func (query *Query) Validate() error {
	query.Normalize()
	if query.Page-1 > math.MaxInt/query.Size {
		return errcode.InvalidParameters.WithMessage("page is too large")
	}
	return nil
}

// KeywordQuery 是带关键字的分页查询参数。
type KeywordQuery struct {
	Query
	Keyword string `form:"keyword" binding:"max=128"`
}
