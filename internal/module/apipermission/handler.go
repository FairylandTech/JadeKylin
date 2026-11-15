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

import (
	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/pkg/web"
)

type Handler struct {
	web.BaseHandler
	apiPermissionService *Service
}

func NewHandler(apiPermissionService *Service) *Handler {
	return &Handler{apiPermissionService: apiPermissionService}
}

func (apiPermissionHandler *Handler) List(ctx *gin.Context) {
	list, err := apiPermissionHandler.apiPermissionService.List(ctx.Request.Context())
	if err != nil {
		apiPermissionHandler.Fail(ctx, err)
		return
	}
	apiPermissionHandler.Success(ctx, list)
}

func (apiPermissionHandler *Handler) Update(ctx *gin.Context) {
	id, err := apiPermissionHandler.BindIdentifier(ctx)
	if err != nil {
		apiPermissionHandler.Fail(ctx, err)
		return
	}
	var requestBody UpdateRequest
	if err := apiPermissionHandler.BindJSON(ctx, &requestBody); err != nil {
		apiPermissionHandler.Fail(ctx, err)
		return
	}
	record, err := apiPermissionHandler.apiPermissionService.Update(ctx.Request.Context(), id, &requestBody)
	if err != nil {
		apiPermissionHandler.Fail(ctx, err)
		return
	}
	apiPermissionHandler.Success(ctx, record)
}
