/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 07:53:51 America/Los_Angeles
 *****************************************************************************/

package model

import (
	"time"

	"jkylin.com/kylin/v2/internal/model/valueobject"
)

type User struct {
	ID        int64                 `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string                `json:"username"`
	Password  string                `json:"-"`
	Nickname  string                `json:"nickname"`
	Remark    string                `json:"remark"`
	Status    string                `gorm:"default:Y" json:"status"`
	Roles     *valueobject.RoleList `gorm:"type:varchar[];default:(-)" json:"roles"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
	Enabled   string                `gorm:"default:Y" json:"enabled"`
	Version   int                   `gorm:"default:1" json:"version"`
}

func (User) TableName() string {
	return "kylin_v2_users"
}
