/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-15 01:30:00 America/Los_Angeles
 *****************************************************************************/

package model

import "time"

// ApiPermission 是一个受保护接口（method + gin 路由模式）的放行权限。
type ApiPermission struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Method      string    `gorm:"not null" json:"method"`
	Path        string    `gorm:"not null" json:"path"`
	Permission  string    `gorm:"not null" json:"permission"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Enabled     string    `gorm:"default:Y" json:"enabled"`
}

func (ApiPermission) TableName() string {
	return "kylin_v2_api_permissions"
}
