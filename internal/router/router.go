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

// Package router 组装各模块的依赖并注册全部路由。
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"jkylin.com/kylin/v2/internal/middleware/security"
	"jkylin.com/kylin/v2/internal/module/auth"
	"jkylin.com/kylin/v2/internal/module/permission"
	"jkylin.com/kylin/v2/internal/module/role"
	"jkylin.com/kylin/v2/internal/module/user"
	"jkylin.com/kylin/v2/pkg/response"
)

// Initialize 注册公共接口；登录及管理接口只在 dependencies.Enabled() 时注册。
func Initialize(engine *gin.Engine, dependencies Dependencies) {
	// 健康检查（已在访问日志中跳过）
	engine.GET("/healthz", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "ok")
	})

	engine.GET("/example", func(ctx *gin.Context) {
		response.Success(ctx, gin.H{"example": "Hello Gin Server."})
	})

	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	if !dependencies.Enabled() {
		return
	}
	services := NewServices(dependencies)

	// 认证：解析 JWT 得到 user id 写入上下文。鉴权在各服务方法内完成：每个方法声明自己需要的权限 code。
	authentication := security.Authentication(services.Auth)
	if dependencies.SkipAuth {
		authentication = security.LocalAuthentication(services.Auth, localPrincipalResolver(services.Auth, dependencies.LocalUsername))
	}
	authenticated := engine.Group("", authentication)
	auth.RegisterRoutes(engine.Group(""), authenticated, services.Auth, services.Users)
	user.RegisterRoutes(authenticated, services.Users)
	role.RegisterRoutes(authenticated, services.Roles)
	permission.RegisterRoutes(authenticated, services.Permissions)
}
