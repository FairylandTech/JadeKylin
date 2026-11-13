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
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"jkylin.com/kylin/v2/internal/config"
	"jkylin.com/kylin/v2/pkg/logger"
)

func InitLogger(cfg config.Log) error {
	err := logger.Init(logger.Options{
		Level:      cfg.Level,
		Format:     cfg.Format,
		Output:     cfg.Output,
		Dir:        cfg.Dir,
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
		Caller:     cfg.Caller,
	})
	if err != nil {
		return err
	}

	// gin 路由注册信息 / 调试信息走 zap
	gin.DebugPrintRouteFunc = func(method, path, handler string, handlers int) {
		logger.L().Debug("route registered",
			zap.String("method", method),
			zap.String("path", path),
			zap.String("handler", handler),
			zap.Int("handlers", handlers),
		)
	}
	gin.DebugPrintFunc = func(format string, values ...any) {
		logger.L().Debug(strings.TrimSpace(fmt.Sprintf(format, values...)), zap.String("component", "gin"))
	}
	return nil
}
