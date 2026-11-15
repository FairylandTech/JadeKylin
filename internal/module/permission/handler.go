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

package permission

import (
	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/pkg/web"
)

type Handler struct {
	web.BaseHandler
	permissionService *Service
}

func NewHandler(permissionService *Service) *Handler {
	return &Handler{permissionService: permissionService}
}

func (permissionHandler *Handler) List(ctx *gin.Context) {
	list, err := permissionHandler.permissionService.List(ctx.Request.Context())
	if err != nil {
		permissionHandler.Fail(ctx, err)
		return
	}
	permissionHandler.Success(ctx, list)
}
