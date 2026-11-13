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

func (m *GlobalMiddleware) Trace() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		traceID := ctx.GetHeader(consts.TraceHeader)
		if !validTraceID(traceID) {
			traceID = newTraceID()
		}

		ctx.Set(consts.TraceKey, traceID)
		ctx.Set(consts.RequestTimestamp, time.Now())
		ctx.Header(consts.TraceHeader, traceID)

		l := logger.L().With(zap.String(consts.TraceKey, traceID))
		ctx.Request = ctx.Request.WithContext(logger.NewContext(ctx.Request.Context(), l))

		ctx.Next()
	}
}

func newTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func validTraceID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_':
		default:
			return false
		}
	}
	return true
}
