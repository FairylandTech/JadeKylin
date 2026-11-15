/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 04:20:18 America/Los_Angeles
 *****************************************************************************/

package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
	"jkylin.com/kylin/v2/internal/middleware"
	"jkylin.com/kylin/v2/internal/router/permissions"
	"jkylin.com/kylin/v2/internal/router/roles"
	"jkylin.com/kylin/v2/internal/router/users"
	"jkylin.com/kylin/v2/pkg/response"
)

func Init(engine *gin.Engine, db *gorm.DB, userAdminToken string) {
	// 健康检查（已在访问日志中跳过）
	engine.GET("/healthz", func(ctx *gin.Context) { ctx.String(http.StatusOK, "ok") })

	engine.GET("/example", func(ctx *gin.Context) {
		response.Success(ctx, gin.H{"example": "Hello Gin Server."})
	})

	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 管理接口（用户 / 角色 / 权限）需要数据库和管理令牌同时配置才注册。
	if db != nil && strings.TrimSpace(userAdminToken) != "" {
		admin := engine.Group("", middleware.AdminToken(userAdminToken))
		users.Router(admin, db)
		roles.Router(admin, db)
		permissions.Router(admin, db)
	}
}
