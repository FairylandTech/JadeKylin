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

package apipermission

type UpdateRequest struct {
	Permission  *string `json:"permission" binding:"omitempty,required,max=128"`
	Description *string `json:"description" binding:"omitempty,max=512"`
}
