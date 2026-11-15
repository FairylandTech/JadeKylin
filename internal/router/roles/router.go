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
	"gorm.io/gorm"
	"jkylin.com/kylin/v2/internal/handler"
	"jkylin.com/kylin/v2/internal/repository"
	"jkylin.com/kylin/v2/internal/service"
)

func Router(group *gin.RouterGroup, db *gorm.DB) {
	roleHandler := handler.NewRoleHandler(service.NewRoleService(repository.NewRoleRepository(db), repository.NewPermissionRepository(db)))
	roles := group.Group("/roles")
	roles.POST("", roleHandler.Create)
	roles.GET("", roleHandler.List)
	roles.GET("/:id", roleHandler.Get)
	roles.PUT("/:id", roleHandler.Update)
	roles.DELETE("/:id", roleHandler.Delete)
}
