/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-13 13:34:03 America/Los_Angeles
 *****************************************************************************/

package model

import (
	"database/sql/driver"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type RoleList []string

func (roles *RoleList) Scan(src any) error {
	return pgtype.NewMap().SQLScanner((*[]string)(roles)).Scan(src)
}

func (roles *RoleList) Value() (driver.Value, error) {
	if roles == nil || len(*roles) == 0 {
		return "{}", nil
	}
	encoded, err := pgtype.NewMap().Encode(pgtype.VarcharArrayOID, pgtype.TextFormatCode, []string(*roles), nil)
	if err != nil {
		return nil, err
	}
	return string(encoded), nil
}

type User struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Nickname  string    `json:"nickname"`
	Remark    string    `json:"remark"`
	Status    string    `gorm:"default:Y" json:"status"`
	Roles     *RoleList `gorm:"type:varchar[];default:(-)" json:"roles"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Enabled   string    `gorm:"default:Y" json:"enabled"`
	Version   int       `gorm:"default:1" json:"version"`
}

func (User) TableName() string {
	return "kylin_v2_users"
}
