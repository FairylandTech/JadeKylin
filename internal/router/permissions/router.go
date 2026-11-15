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

package permissions

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"jkylin.com/kylin/v2/internal/handler"
	"jkylin.com/kylin/v2/internal/repository"
	"jkylin.com/kylin/v2/internal/service"
)

func Router(group *gin.RouterGroup, db *gorm.DB) {
	permissionHandler := handler.NewPermissionHandler(service.NewPermissionService(repository.NewPermissionRepository(db)))
	group.GET("/permissions", permissionHandler.List)
}
