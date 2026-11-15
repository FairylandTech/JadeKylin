/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 20:01:06 America/Los_Angeles
 *****************************************************************************/

package valueobject

import (
	"database/sql/driver"
	"sync"

	"github.com/jackc/pgx/v5/pgtype"
)

var pgTypeMaps = sync.Pool{New: func() any { return pgtype.NewMap() }}

// StringArray 映射 PostgreSQL varchar[]。nil 和空切片都写为 '{}'。
type StringArray []string

func (array *StringArray) Scan(src any) error {
	typeMap := pgTypeMaps.Get().(*pgtype.Map)
	defer pgTypeMaps.Put(typeMap)
	return typeMap.SQLScanner((*[]string)(array)).Scan(src)
}

func (array *StringArray) Value() (driver.Value, error) {
	if array == nil || len(*array) == 0 {
		return "{}", nil
	}
	typeMap := pgTypeMaps.Get().(*pgtype.Map)
	defer pgTypeMaps.Put(typeMap)
	encoded, err := typeMap.Encode(pgtype.VarcharArrayOID, pgtype.TextFormatCode, []string(*array), nil)
	if err != nil {
		return nil, err
	}
	return string(encoded), nil
}

// RoleList 是用户的角色 code 列表。
type RoleList = StringArray

// PermissionList 是角色的权限 code 列表。
type PermissionList = StringArray
