/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 21:04:11 America/Los_Angeles
 *****************************************************************************/

// Package web 提供 handler 层公共能力：统一响应、参数绑定、日志。
package web

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"jkylin.com/kylin/v2/pkg/logger"
	"jkylin.com/kylin/v2/pkg/response"
)

// failCallerSkip 让错误日志的调用位置指向调用 Fail 的 handler 方法。
const failCallerSkip = 2

// BaseHandler 所有 handler 嵌入的基础能力：统一响应、参数绑定、日志。
type BaseHandler struct{}

func (BaseHandler) Success(ctx *gin.Context, data any) {
	response.Success(ctx, data)
}

func (BaseHandler) Page(ctx *gin.Context, data any, page, size int, total int64) {
	response.Page(ctx, data, page, size, total)
}

func (BaseHandler) Fail(ctx *gin.Context, err error) {
	response.FailSkip(ctx, err, failCallerSkip)
}

// Logger 返回携带 trace_id 的请求级 Logger。
func (BaseHandler) Logger(ctx *gin.Context) *zap.Logger {
	return logger.FromContext(ctx.Request.Context())
}

// DisableCache 禁止任何中间层缓存响应（响应中包含 token、验证码、明文密码等敏感数据时使用）。
func (BaseHandler) DisableCache(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
}
