/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 23:30:00 America/Los_Angeles
 *****************************************************************************/

package global

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"jkylin.com/kylin/v2/pkg/logger"
)

// AccessLog 访问日志：请求结束后按状态码分级输出（5xx=error, 4xx=warn, 其它=info）。
// 依赖 Trace 中间件注入的 Logger，因此需注册在 Trace 之后。
func (m *GlobalMiddleware) AccessLog() gin.HandlerFunc {
	skip := make(map[string]struct{}, len(m.cfg.Log.SkipPaths))
	for _, p := range m.cfg.Log.SkipPaths {
		skip[p] = struct{}{}
	}

	return func(ctx *gin.Context) {
		start := time.Now()
		path := ctx.Request.URL.Path
		query := ctx.Request.URL.RawQuery

		ctx.Next()

		if _, ok := skip[path]; ok {
			return
		}

		status := ctx.Writer.Status()
		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", ctx.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("route", ctx.FullPath()),
			zap.String("ip", ctx.ClientIP()),
			zap.String("user_agent", ctx.Request.UserAgent()),
			zap.Int("size", ctx.Writer.Size()),
			zap.Duration("latency", time.Since(start)),
		}
		if len(ctx.Errors) > 0 {
			fields = append(fields, zap.String("errors", ctx.Errors.ByType(gin.ErrorTypeAny).String()))
		}

		// 访问日志不需要堆栈（panic 堆栈由 Recovery 记录）
		log := logger.FromContext(ctx.Request.Context()).WithOptions(zap.AddStacktrace(zapcore.FatalLevel))
		switch {
		case status >= 500:
			log.Error("access", fields...)
		case status >= 400:
			log.Warn("access", fields...)
		default:
			log.Info("access", fields...)
		}
	}
}
