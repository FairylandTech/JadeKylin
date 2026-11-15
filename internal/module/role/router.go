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

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册角色管理接口。
func RegisterRoutes(group *gin.RouterGroup, roleService *Service) {
	roleHandler := NewHandler(roleService)
	roles := group.Group("/roles")
	roles.POST("", roleHandler.Create)
	roles.GET("", roleHandler.List)
	roles.GET("/:id", roleHandler.Get)
	roles.PUT("/:id", roleHandler.Update)
	roles.DELETE("/:id", roleHandler.Delete)
}
