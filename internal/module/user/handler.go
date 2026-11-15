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

package user

import (
	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/pkg/pagination"
	"jkylin.com/kylin/v2/internal/pkg/web"
)

type Handler struct {
	web.BaseHandler
	userService *Service
}

func NewHandler(userService *Service) *Handler {
	return &Handler{userService: userService}
}

func (userHandler *Handler) Create(ctx *gin.Context) {
	var requestBody CreateRequest
	if err := userHandler.BindJSON(ctx, &requestBody); err != nil {
		userHandler.Fail(ctx, err)
		return
	}
	record, err := userHandler.userService.Create(ctx.Request.Context(), &requestBody)
	if err != nil {
		userHandler.Fail(ctx, err)
		return
	}
	userHandler.Success(ctx, record)
}

func (userHandler *Handler) Get(ctx *gin.Context) {
	id, err := userHandler.BindIdentifier(ctx)
	if err != nil {
		userHandler.Fail(ctx, err)
		return
	}
	record, err := userHandler.userService.Get(ctx.Request.Context(), id)
	if err != nil {
		userHandler.Fail(ctx, err)
		return
	}
	userHandler.Success(ctx, record)
}

func (userHandler *Handler) List(ctx *gin.Context) {
	var query pagination.KeywordQuery
	if err := userHandler.BindQuery(ctx, &query); err != nil {
		userHandler.Fail(ctx, err)
		return
	}
	list, total, err := userHandler.userService.List(ctx.Request.Context(), &query)
	if err != nil {
		userHandler.Fail(ctx, err)
		return
	}
	userHandler.Page(ctx, list, query.Page, query.Size, total)
}

func (userHandler *Handler) Update(ctx *gin.Context) {
	id, err := userHandler.BindIdentifier(ctx)
	if err != nil {
		userHandler.Fail(ctx, err)
		return
	}
	var requestBody UpdateRequest
	if err := userHandler.BindJSON(ctx, &requestBody); err != nil {
		userHandler.Fail(ctx, err)
		return
	}
	record, err := userHandler.userService.Update(ctx.Request.Context(), id, &requestBody)
	if err != nil {
		userHandler.Fail(ctx, err)
		return
	}
	userHandler.Success(ctx, record)
}

func (userHandler *Handler) ChangePassword(ctx *gin.Context) {
	id, err := userHandler.BindIdentifier(ctx)
	if err != nil {
		userHandler.Fail(ctx, err)
		return
	}
	var requestBody ChangePasswordRequest
	if err := userHandler.BindJSON(ctx, &requestBody); err != nil {
		userHandler.Fail(ctx, err)
		return
	}
	if err := userHandler.userService.ChangePassword(ctx.Request.Context(), id, &requestBody); err != nil {
		userHandler.Fail(ctx, err)
		return
	}
	userHandler.Success(ctx, nil)
}

func (userHandler *Handler) ResetPassword(ctx *gin.Context) {
	id, err := userHandler.BindIdentifier(ctx)
	if err != nil {
		userHandler.Fail(ctx, err)
		return
	}
	result, err := userHandler.userService.ResetPassword(ctx.Request.Context(), id)
	if err != nil {
		userHandler.Fail(ctx, err)
		return
	}
	// 响应包含明文密码，禁止任何中间层缓存。
	userHandler.DisableCache(ctx)
	userHandler.Success(ctx, result)
}

func (userHandler *Handler) Delete(ctx *gin.Context) {
	id, err := userHandler.BindIdentifier(ctx)
	if err != nil {
		userHandler.Fail(ctx, err)
		return
	}
	if err := userHandler.userService.Delete(ctx.Request.Context(), id); err != nil {
		userHandler.Fail(ctx, err)
		return
	}
	userHandler.Success(ctx, nil)
}
