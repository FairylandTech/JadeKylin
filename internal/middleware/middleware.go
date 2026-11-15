/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 22:11:10 America/Los_Angeles
 *****************************************************************************/

package middleware

import (
	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/config"
	"jkylin.com/kylin/v2/internal/middleware/global"
	"jkylin.com/kylin/v2/pkg/errcode"
	"jkylin.com/kylin/v2/pkg/response"
)

// Initialize 注册全局中间件以及 404 / 405 处理。
func Initialize(engine *gin.Engine, configuration *config.Config) {
	globalMiddleware := global.NewMiddleware(configuration)

	engine.Use(globalMiddleware.Trace(), globalMiddleware.AccessLog(), globalMiddleware.Recovery())

	if configuration.CORS.Enabled {
		engine.Use(globalMiddleware.CORS())
	}
	if configuration.RateLimit.Enabled {
		engine.Use(globalMiddleware.RateLimit())
	}

	// 404, 405 处理
	engine.HandleMethodNotAllowed = true // 启用 405 Method Not Allowed
	engine.NoRoute(func(ctx *gin.Context) {
		response.Fail(ctx, errcode.RouteNotFound)
	})
	engine.NoMethod(func(ctx *gin.Context) {
		response.Fail(ctx, errcode.MethodNotAllow)
	})
}
