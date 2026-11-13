/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

package requests

import (
	"jkylin.com/kylin/v2/internal/model"
)

type CreateRequest struct {
	Username string          `json:"username" binding:"required,max=128"`
	Password string          `json:"password" binding:"required"`
	Nickname string          `json:"nickname" binding:"max=128"`
	Remark   string          `json:"remark" binding:"max=512"`
	Status   string          `json:"status" binding:"omitempty,len=1"`
	Enabled  string          `json:"enabled" binding:"omitempty,len=1"`
	Roles    *model.RoleList `json:"roles" binding:"omitempty,dive,required"`
}

type UpdateRequest struct {
	Username *string         `json:"username" binding:"omitempty,required,max=128"`
	Password *string         `json:"password" binding:"omitempty,required"`
	Nickname *string         `json:"nickname" binding:"omitempty,max=128"`
	Remark   *string         `json:"remark" binding:"omitempty,max=512"`
	Status   *string         `json:"status" binding:"omitempty,len=1"`
	Enabled  *string         `json:"enabled" binding:"omitempty,len=1"`
	Roles    *model.RoleList `json:"roles" binding:"omitempty,dive,required"`
	Version  *int            `json:"version" binding:"omitempty,min=1,max=2147483647"`
}

type ListQuery struct {
	model.PageQuery
	Keyword string `form:"keyword" binding:"max=128"`
}

type URI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}
