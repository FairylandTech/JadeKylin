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

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册用户管理接口。
func RegisterRoutes(group *gin.RouterGroup, userService *Service) {
	userHandler := NewHandler(userService)
	users := group.Group("/users")
	users.POST("", userHandler.Create)
	users.GET("", userHandler.List)
	users.GET("/:id", userHandler.Get)
	users.PUT("/:id", userHandler.Update)
	users.DELETE("/:id", userHandler.Delete)
	users.PUT("/:id/password", userHandler.ChangePassword)
	users.POST("/:id/password/reset", userHandler.ResetPassword)
}
