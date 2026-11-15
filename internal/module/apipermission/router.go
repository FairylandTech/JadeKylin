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

package apipermission

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册接口放行权限管理接口。
func RegisterRoutes(group *gin.RouterGroup, apiPermissionService *Service) {
	apiPermissionHandler := NewHandler(apiPermissionService)
	apiPermissions := group.Group("/api-permissions")
	apiPermissions.GET("", apiPermissionHandler.List)
	apiPermissions.PUT("/:id", apiPermissionHandler.Update)
}
