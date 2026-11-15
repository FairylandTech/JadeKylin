/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 09:56:13 America/Los_Angeles
 *****************************************************************************/

package model

import (
	"time"

	"jkylin.com/kylin/v2/internal/model/valueobject"
)

type Role struct {
	ID          int64                       `gorm:"primaryKey;autoIncrement" json:"id"`
	Code        string                      `gorm:"unique;not null" json:"code"`
	Name        string                      `gorm:"unique;not null" json:"name"`
	Description string                      `json:"description"`
	Permissions *valueobject.PermissionList `gorm:"type:varchar[];default:(-)" json:"permissions"`
	CreatedAt   time.Time                   `json:"created_at"`
	UpdatedAt   time.Time                   `json:"updated_at"`
	Enabled     string                      `gorm:"default:Y" json:"enabled"`
}

func (Role) TableName() string {
	return "kylin_v2_roles"
}
