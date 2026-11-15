/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 11:22:25 America/Los_Angeles
 *****************************************************************************/

// Package permission 是权限模块：权限数据的模型、仓储、服务与接口。
package permission

import "time"

type Permission struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Code        string    `gorm:"unique;not null" json:"code"`
	Name        string    `gorm:"unique;not null" json:"name"`
	Category    string    `gorm:"not null" json:"category"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Enabled     string    `gorm:"default:Y" json:"enabled"`
}

func (Permission) TableName() string {
	return "kylin_v2_permissions"
}
