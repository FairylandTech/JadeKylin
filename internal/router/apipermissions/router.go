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

package apipermissions

import (
	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/handler"
	"jkylin.com/kylin/v2/internal/service"
)

func Router(group *gin.RouterGroup, apiPermissionService *service.ApiPermissionService) {
	apiPermissionHandler := handler.NewApiPermissionHandler(apiPermissionService)
	apiPermissions := group.Group("/api-permissions")
	apiPermissions.GET("", apiPermissionHandler.List)
	apiPermissions.PUT("/:id", apiPermissionHandler.Update)
}
