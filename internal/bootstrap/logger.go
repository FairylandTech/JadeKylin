/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 04:19:52 America/Los_Angeles
 *****************************************************************************/

package bootstrap

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"jkylin.com/kylin/v2/internal/config"
	"jkylin.com/kylin/v2/pkg/logger"
)

// InitializeLogger 按配置初始化全局日志，并让 gin 的调试输出走 zap。
func InitializeLogger(configuration config.Log) error {
	err := logger.Initialize(logger.Options{
		Level:      configuration.Level,
		Format:     configuration.Format,
		Output:     configuration.Output,
		Directory:  configuration.Directory,
		Filename:   configuration.Filename,
		MaxSize:    configuration.MaxSize,
		MaxBackups: configuration.MaxBackups,
		MaxAge:     configuration.MaxAge,
		Compress:   configuration.Compress,
		Caller:     configuration.Caller,
	})
	if err != nil {
		return err
	}

	// gin 路由注册信息 / 调试信息走 zap
	gin.DebugPrintRouteFunc = func(method, path, handler string, handlers int) {
		logger.Global().Debug("route registered",
			zap.String("method", method),
			zap.String("path", path),
			zap.String("handler", handler),
			zap.Int("handlers", handlers),
		)
	}
	gin.DebugPrintFunc = func(format string, values ...any) {
		logger.Global().Debug(strings.TrimSpace(fmt.Sprintf(format, values...)), zap.String("component", "gin"))
	}
	return nil
}

// contextLogger 返回 ctx 携带的请求级日志器；不存在时返回全局日志器。
// gorm.go 引用的是 GORM 的 logger 包，应用日志相关的访问集中放在本文件，避免导入别名。
func contextLogger(ctx context.Context) *zap.Logger {
	return logger.FromContext(ctx)
}

// contextTraceID 返回 ctx 携带的 trace id；不存在时返回空字符串。
func contextTraceID(ctx context.Context) string {
	return logger.TraceID(ctx)
}
