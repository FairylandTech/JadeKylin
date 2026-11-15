/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 21:30:00 America/Los_Angeles
 *****************************************************************************/

package role

import "jkylin.com/kylin/v2/internal/pkg/valueobject"

type CreateRequest struct {
	Code        string                      `json:"code" binding:"required,max=128"`
	Name        string                      `json:"name" binding:"required,max=128"`
	Description string                      `json:"description" binding:"max=512"`
	Permissions *valueobject.PermissionList `json:"permissions" binding:"omitempty,dive,required"`
}

// UpdateRequest 不允许修改 code：用户通过角色 code 关联角色，改 code 会让已有关联失效。
type UpdateRequest struct {
	Name        *string                     `json:"name" binding:"omitempty,required,max=128"`
	Description *string                     `json:"description" binding:"omitempty,max=512"`
	Permissions *valueobject.PermissionList `json:"permissions" binding:"omitempty,dive,required"`
}
