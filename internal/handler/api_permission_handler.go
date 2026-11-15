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

package handler

import (
	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/contracts/requests"
	"jkylin.com/kylin/v2/internal/pkg/support"
	"jkylin.com/kylin/v2/internal/service"
)

type ApiPermissionHandler struct {
	support.HandlerSupportBase
	apiPermissionService *service.ApiPermissionService
}

func NewApiPermissionHandler(apiPermissionService *service.ApiPermissionService) *ApiPermissionHandler {
	return &ApiPermissionHandler{apiPermissionService: apiPermissionService}
}

func (handler *ApiPermissionHandler) List(ctx *gin.Context) {
	list, err := handler.apiPermissionService.List(ctx.Request.Context())
	if err != nil {
		handler.Fail(ctx, err)
		return
	}
	handler.Success(ctx, list)
}

func (handler *ApiPermissionHandler) Update(ctx *gin.Context) {
	var uri requests.URI
	if err := handler.BindURI(ctx, &uri); err != nil {
		handler.Fail(ctx, err)
		return
	}
	var req requests.UpdateApiPermissionRequest
	if err := handler.BindJSON(ctx, &req); err != nil {
		handler.Fail(ctx, err)
		return
	}
	record, err := handler.apiPermissionService.Update(ctx.Request.Context(), uri.ID, &req)
	if err != nil {
		handler.Fail(ctx, err)
		return
	}
	handler.Success(ctx, record)
}
