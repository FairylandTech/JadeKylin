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
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"jkylin.com/kylin/v2/pkg/consts"
	"jkylin.com/kylin/v2/pkg/logger"
)

const (
	// traceIDBytes 是自动生成的 trace id 的字节数。
	traceIDBytes = 16
	// maximumTraceIDLength 是接受的客户端 trace id 的最大长度。
	maximumTraceIDLength = 64
)

func (globalMiddleware *Middleware) Trace() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		traceID := ctx.GetHeader(consts.TraceHeader)
		if !validTraceID(traceID) {
			traceID = newTraceID()
		}

		ctx.Set(consts.TraceKey, traceID)
		ctx.Set(consts.RequestTimestamp, time.Now())
		ctx.Header(consts.TraceHeader, traceID)

		// 创建一个带有 trace_id 的日志对象
		requestLogger := logger.Global().With(zap.String(consts.TraceKey, traceID))
		// 将日志对象添加到请求上下文中
		ctx.Request = ctx.Request.WithContext(logger.WithTraceID(logger.NewContext(ctx.Request.Context(), requestLogger), traceID))

		ctx.Next()
	}
}

func newTraceID() string {
	buffer := make([]byte, traceIDBytes)
	_, _ = rand.Read(buffer)
	return hex.EncodeToString(buffer)
}

func validTraceID(traceID string) bool {
	if traceID == "" || len(traceID) > maximumTraceIDLength {
		return false
	}
	for _, character := range traceID {
		switch {
		case character >= 'a' && character <= 'z',
			character >= 'A' && character <= 'Z',
			character >= '0' && character <= '9',
			character == '-',
			character == '_':
		default:
			return false
		}
	}
	return true
}
