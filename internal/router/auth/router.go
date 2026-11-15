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
	"jkylin.com/kylin/v2/internal/handler"
	"jkylin.com/kylin/v2/internal/service"
)

// Router 注册 /auth 接口：public 为无需登录的分组，authenticated 为已经过 Authentication 的分组。
// /auth 下的自助接口只要求登录，不做接口权限判断。
func Router(public *gin.RouterGroup, authenticated *gin.RouterGroup, authService *service.AuthService, userService *service.UserService) {
	authHandler := handler.NewAuthHandler(authService, userService)
	public.GET("/auth/captcha", authHandler.Captcha)
	public.POST("/auth/login", authHandler.Login)

	authenticated.POST("/auth/logout", authHandler.Logout)
	authenticated.GET("/auth/me", authHandler.Me)
	authenticated.PUT("/auth/password", authHandler.ChangePassword)
}
