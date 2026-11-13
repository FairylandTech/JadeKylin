/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 23:30:00 America/Los_Angeles
 *****************************************************************************/

package model

// Example 示例实体。
type Example struct {
	SupportModelBase
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CreateExampleRequest 创建请求。
type CreateExampleRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=64"`
	Description string `json:"description" binding:"max=255"`
}

// UpdateExampleRequest 更新请求，字段为 nil 表示不修改。
type UpdateExampleRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=64"`
	Description *string `json:"description" binding:"omitempty,max=255"`
}

// ListExampleQuery 列表查询参数。
type ListExampleQuery struct {
	PageQuery
	Keyword string `form:"keyword" binding:"max=64"`
}

// ExampleURI 路径参数。
type ExampleURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}
