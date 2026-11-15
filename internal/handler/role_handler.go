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

package handler

import (
	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/contracts/requests"
	"jkylin.com/kylin/v2/internal/pkg/support"
	"jkylin.com/kylin/v2/internal/service"
)

type RoleHandler struct {
	support.HandlerSupportBase
	roleService *service.RoleService
}

func NewRoleHandler(roleService *service.RoleService) *RoleHandler {
	return &RoleHandler{roleService: roleService}
}

func (handler *RoleHandler) Create(ctx *gin.Context) {
	var requestBody requests.CreateRoleRequest
	if err := handler.BindJSON(ctx, &requestBody); err != nil {
		handler.Fail(ctx, err)
		return
	}
	record, err := handler.roleService.Create(ctx.Request.Context(), &requestBody)
	if err != nil {
		handler.Fail(ctx, err)
		return
	}
	handler.Success(ctx, record)
}

func (handler *RoleHandler) Get(ctx *gin.Context) {
	var uri requests.URI
	if err := handler.BindURI(ctx, &uri); err != nil {
		handler.Fail(ctx, err)
		return
	}
	record, err := handler.roleService.Get(ctx.Request.Context(), uri.ID)
	if err != nil {
		handler.Fail(ctx, err)
		return
	}
	handler.Success(ctx, record)
}

func (handler *RoleHandler) List(ctx *gin.Context) {
	var q requests.ListQuery
	if err := handler.BindQuery(ctx, &q); err != nil {
		handler.Fail(ctx, err)
		return
	}
	list, total, err := handler.roleService.List(ctx.Request.Context(), &q)
	if err != nil {
		handler.Fail(ctx, err)
		return
	}
	handler.Page(ctx, list, q.Page, q.Size, total)
}

func (handler *RoleHandler) Update(ctx *gin.Context) {
	var uri requests.URI
	if err := handler.BindURI(ctx, &uri); err != nil {
		handler.Fail(ctx, err)
		return
	}
	var req requests.UpdateRoleRequest
	if err := handler.BindJSON(ctx, &req); err != nil {
		handler.Fail(ctx, err)
		return
	}
	record, err := handler.roleService.Update(ctx.Request.Context(), uri.ID, &req)
	if err != nil {
		handler.Fail(ctx, err)
		return
	}
	handler.Success(ctx, record)
}

func (handler *RoleHandler) Delete(ctx *gin.Context) {
	var uri requests.URI
	if err := handler.BindURI(ctx, &uri); err != nil {
		handler.Fail(ctx, err)
		return
	}
	if err := handler.roleService.Delete(ctx.Request.Context(), uri.ID); err != nil {
		handler.Fail(ctx, err)
		return
	}
	handler.Success(ctx, nil)
}
