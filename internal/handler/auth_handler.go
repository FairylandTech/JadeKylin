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

package handler

import (
	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/contracts/requests"
	"jkylin.com/kylin/v2/internal/middleware/auth"
	"jkylin.com/kylin/v2/internal/pkg/support"
	"jkylin.com/kylin/v2/internal/service"
	"jkylin.com/kylin/v2/pkg/errcode"
)

type AuthHandler struct {
	support.HandlerSupportBase
	authService *service.AuthService
	userService *service.UserService
}

func NewAuthHandler(authService *service.AuthService, userService *service.UserService) *AuthHandler {
	return &AuthHandler{authService: authService, userService: userService}
}

func (handler *AuthHandler) Captcha(ctx *gin.Context) {
	result, err := handler.authService.Captcha(ctx.Request.Context())
	if err != nil {
		handler.Fail(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	handler.Success(ctx, result)
}

func (handler *AuthHandler) Login(ctx *gin.Context) {
	var requestBody requests.LoginRequest
	if err := handler.BindJSON(ctx, &requestBody); err != nil {
		handler.Fail(ctx, err)
		return
	}
	result, err := handler.authService.Login(ctx.Request.Context(), &requestBody, ctx.ClientIP(), ctx.Request.UserAgent())
	if err != nil {
		handler.Fail(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	handler.Success(ctx, result)
}

func (handler *AuthHandler) Logout(ctx *gin.Context) {
	principal := auth.Principal(ctx)
	if principal == nil {
		handler.Fail(ctx, errcode.Unauthorized)
		return
	}
	if err := handler.authService.Logout(ctx.Request.Context(), principal); err != nil {
		handler.Fail(ctx, err)
		return
	}
	handler.Success(ctx, nil)
}

// ChangePassword 当前用户修改自己的密码；成功后该用户所有设备（包括当前设备）下线，需要重新登录。
func (handler *AuthHandler) ChangePassword(ctx *gin.Context) {
	principal := auth.Principal(ctx)
	if principal == nil {
		handler.Fail(ctx, errcode.Unauthorized)
		return
	}
	var requestBody requests.ChangePasswordRequest
	if err := handler.BindJSON(ctx, &requestBody); err != nil {
		handler.Fail(ctx, err)
		return
	}
	if err := handler.userService.ChangePassword(ctx.Request.Context(), principal.UserID, &requestBody); err != nil {
		handler.Fail(ctx, err)
		return
	}
	handler.Success(ctx, nil)
}

func (handler *AuthHandler) Me(ctx *gin.Context) {
	principal := auth.Principal(ctx)
	if principal == nil {
		handler.Fail(ctx, errcode.Unauthorized)
		return
	}
	record, err := handler.authService.Me(ctx.Request.Context(), principal)
	if err != nil {
		handler.Fail(ctx, err)
		return
	}
	handler.Success(ctx, record)
}
