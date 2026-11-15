/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 23:30:00 America/Los_Angeles
 *****************************************************************************/

package auth

import (
	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/module/user"
	"jkylin.com/kylin/v2/internal/pkg/identity"
	"jkylin.com/kylin/v2/internal/pkg/web"
)

type Handler struct {
	web.BaseHandler
	authService *Service
	userService *user.Service
}

func NewHandler(authService *Service, userService *user.Service) *Handler {
	return &Handler{authService: authService, userService: userService}
}

func (authHandler *Handler) Captcha(ctx *gin.Context) {
	result, err := authHandler.authService.Captcha(ctx.Request.Context())
	if err != nil {
		authHandler.Fail(ctx, err)
		return
	}
	authHandler.DisableCache(ctx)
	authHandler.Success(ctx, result)
}

func (authHandler *Handler) Login(ctx *gin.Context) {
	var requestBody LoginRequest
	if err := authHandler.BindJSON(ctx, &requestBody); err != nil {
		authHandler.Fail(ctx, err)
		return
	}
	result, err := authHandler.authService.Login(ctx.Request.Context(), &requestBody, ctx.ClientIP(), ctx.Request.UserAgent())
	if err != nil {
		authHandler.Fail(ctx, err)
		return
	}
	authHandler.DisableCache(ctx)
	authHandler.Success(ctx, result)
}

func (authHandler *Handler) Logout(ctx *gin.Context) {
	principal, err := identity.Require(ctx)
	if err != nil {
		authHandler.Fail(ctx, err)
		return
	}
	if err := authHandler.authService.Logout(ctx.Request.Context(), principal); err != nil {
		authHandler.Fail(ctx, err)
		return
	}
	authHandler.Success(ctx, nil)
}

// ChangePassword 当前用户修改自己的密码；成功后该用户所有设备（包括当前设备）下线，需要重新登录。
func (authHandler *Handler) ChangePassword(ctx *gin.Context) {
	var requestBody user.ChangePasswordRequest
	if err := authHandler.BindJSON(ctx, &requestBody); err != nil {
		authHandler.Fail(ctx, err)
		return
	}
	if err := authHandler.userService.ChangeOwnPassword(ctx.Request.Context(), &requestBody); err != nil {
		authHandler.Fail(ctx, err)
		return
	}
	authHandler.Success(ctx, nil)
}

func (authHandler *Handler) Me(ctx *gin.Context) {
	principal, err := identity.Require(ctx)
	if err != nil {
		authHandler.Fail(ctx, err)
		return
	}
	record, err := authHandler.authService.Me(ctx.Request.Context(), principal)
	if err != nil {
		authHandler.Fail(ctx, err)
		return
	}
	authHandler.Success(ctx, record)
}
