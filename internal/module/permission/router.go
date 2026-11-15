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

package permission

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes 注册权限查询接口。
func RegisterRoutes(group *gin.RouterGroup, database *gorm.DB) {
	permissionHandler := NewHandler(NewService(NewRepository(database)))
	group.GET("/permissions", permissionHandler.List)
}
