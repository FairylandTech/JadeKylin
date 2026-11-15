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

package roles

import (
	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/handler"
	"jkylin.com/kylin/v2/internal/service"
)

func Router(group *gin.RouterGroup, roleService *service.RoleService) {
	roleHandler := handler.NewRoleHandler(roleService)
	roles := group.Group("/roles")
	roles.POST("", roleHandler.Create)
	roles.GET("", roleHandler.List)
	roles.GET("/:id", roleHandler.Get)
	roles.PUT("/:id", roleHandler.Update)
	roles.DELETE("/:id", roleHandler.Delete)
}
