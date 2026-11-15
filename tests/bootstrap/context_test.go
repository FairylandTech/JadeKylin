/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 23:15:00 America/Los_Angeles
 *****************************************************************************/

package bootstrap_test

import (
	"context"

	"go.uber.org/zap"
	"jkylin.com/kylin/v2/pkg/logger"
)

// gorm_test.go 引用的是 GORM 的 logger 包，应用日志相关的上下文构造集中放在本文件，避免导入别名。

// loggerContext 返回携带 zapLogger 的 context。
func loggerContext(zapLogger *zap.Logger) context.Context {
	return logger.NewContext(context.Background(), zapLogger)
}

// tracedContext 返回携带 zapLogger 和 trace id 的 context。
func tracedContext(zapLogger *zap.Logger, traceID string) context.Context {
	return logger.WithTraceID(loggerContext(zapLogger), traceID)
}
