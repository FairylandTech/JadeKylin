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

package support

import "time"

type SupportModelBase struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PageQuery struct {
	Page int `form:"page" binding:"omitempty,min=1"`
	Size int `form:"size" binding:"omitempty,min=1,max=100"`
}

func (pageQuery *PageQuery) Normalize() {
	if pageQuery.Page <= 0 {
		pageQuery.Page = 1
	}
	if pageQuery.Size <= 0 {
		pageQuery.Size = 10
	}
}

func (pageQuery *PageQuery) Offset() int {
	return (pageQuery.Page - 1) * pageQuery.Size
}
