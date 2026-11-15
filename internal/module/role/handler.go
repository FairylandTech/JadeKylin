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

import (
	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/pkg/pagination"
	"jkylin.com/kylin/v2/internal/pkg/web"
)

type Handler struct {
	web.BaseHandler
	roleService *Service
}

func NewHandler(roleService *Service) *Handler {
	return &Handler{roleService: roleService}
}

func (roleHandler *Handler) Create(ctx *gin.Context) {
	var requestBody CreateRequest
	if err := roleHandler.BindJSON(ctx, &requestBody); err != nil {
		roleHandler.Fail(ctx, err)
		return
	}
	record, err := roleHandler.roleService.Create(ctx.Request.Context(), &requestBody)
	if err != nil {
		roleHandler.Fail(ctx, err)
		return
	}
	roleHandler.Success(ctx, record)
}

func (roleHandler *Handler) Get(ctx *gin.Context) {
	id, err := roleHandler.BindIdentifier(ctx)
	if err != nil {
		roleHandler.Fail(ctx, err)
		return
	}
	record, err := roleHandler.roleService.Get(ctx.Request.Context(), id)
	if err != nil {
		roleHandler.Fail(ctx, err)
		return
	}
	roleHandler.Success(ctx, record)
}

func (roleHandler *Handler) List(ctx *gin.Context) {
	var query pagination.KeywordQuery
	if err := roleHandler.BindQuery(ctx, &query); err != nil {
		roleHandler.Fail(ctx, err)
		return
	}
	list, total, err := roleHandler.roleService.List(ctx.Request.Context(), &query)
	if err != nil {
		roleHandler.Fail(ctx, err)
		return
	}
	roleHandler.Page(ctx, list, query.Page, query.Size, total)
}

func (roleHandler *Handler) Update(ctx *gin.Context) {
	id, err := roleHandler.BindIdentifier(ctx)
	if err != nil {
		roleHandler.Fail(ctx, err)
		return
	}
	var requestBody UpdateRequest
	if err := roleHandler.BindJSON(ctx, &requestBody); err != nil {
		roleHandler.Fail(ctx, err)
		return
	}
	record, err := roleHandler.roleService.Update(ctx.Request.Context(), id, &requestBody)
	if err != nil {
		roleHandler.Fail(ctx, err)
		return
	}
	roleHandler.Success(ctx, record)
}

func (roleHandler *Handler) Delete(ctx *gin.Context) {
	id, err := roleHandler.BindIdentifier(ctx)
	if err != nil {
		roleHandler.Fail(ctx, err)
		return
	}
	if err := roleHandler.roleService.Delete(ctx.Request.Context(), id); err != nil {
		roleHandler.Fail(ctx, err)
		return
	}
	roleHandler.Success(ctx, nil)
}
