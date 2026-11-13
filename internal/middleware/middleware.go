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

// Init 注册全局中间件。顺序很重要：
//
//	Trace      -> 最先执行，生成 trace_id 并注入 Logger，后续日志都能带上
//	AccessLog  -> 包住 Recovery，确保 panic 后仍能记录到 500 状态码
//	Recovery   -> 捕获业务 panic
//	CORS       -> 预检请求在限流之前直接返回
//	RateLimit  -> 按 IP 限流
func Init(engine *gin.Engine, cfg *config.Config) {
	m := global.New(cfg)

	engine.Use(m.Trace(), m.AccessLog(), m.Recovery())
	if cfg.CORS.Enabled {
		engine.Use(m.CORS())
	}
	if cfg.RateLimit.Enabled {
		engine.Use(m.RateLimit())
	}

	// 404 / 405 统一响应格式
	engine.HandleMethodNotAllowed = true
	engine.NoRoute(func(ctx *gin.Context) { response.Fail(ctx, errcode.RouteNotFound) })
	engine.NoMethod(func(ctx *gin.Context) { response.Fail(ctx, errcode.MethodNotAllow) })
}
