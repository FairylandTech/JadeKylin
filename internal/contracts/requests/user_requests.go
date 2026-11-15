/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-13 13:56:36 America/Los_Angeles
 *****************************************************************************/

package requests

import (
	"jkylin.com/kylin/v2/internal/model/valueobject"
	"jkylin.com/kylin/v2/internal/pkg/support"
)

type CreateRequest struct {
	Username string                `json:"username" binding:"required,max=128"`
	Password string                `json:"password" binding:"required"`
	Nickname string                `json:"nickname" binding:"max=128"`
	Remark   string                `json:"remark" binding:"max=512"`
	Roles    *valueobject.RoleList `json:"roles" binding:"omitempty,dive,required"`
}

type UpdateRequest struct {
	Username *string               `json:"username" binding:"omitempty,required,max=128"`
	Nickname *string               `json:"nickname" binding:"omitempty,max=128"`
	Remark   *string               `json:"remark" binding:"omitempty,max=512"`
	Status   *string               `json:"status" binding:"omitempty,oneof=Y N"`
	Roles    *valueobject.RoleList `json:"roles" binding:"omitempty,dive,required"`
	Version  *int                  `json:"version" binding:"omitempty,min=1,max=2147483647"`
}

// ChangePasswordRequest 校验旧密码后设置新密码。
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ResetPasswordResponse 携带重置后生成的明文密码；明文只在这次响应中返回，数据库只存 bcrypt 哈希。
type ResetPasswordResponse struct {
	ID          int64  `json:"id"`
	NewPassword string `json:"new_password"`
}

type ListQuery struct {
	support.PageQuery
	Keyword string `form:"keyword" binding:"max=128"`
}

type URI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}
