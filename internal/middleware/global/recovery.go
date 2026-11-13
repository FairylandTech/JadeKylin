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
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"jkylin.com/kylin/v2/pkg/errcode"
	"jkylin.com/kylin/v2/pkg/logger"
	"jkylin.com/kylin/v2/pkg/response"
)

// Recovery 捕获 panic，记录堆栈并返回统一 500 响应。
func (m *Middleware) Recovery() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			// http.ErrAbortHandler 是标准库约定的“静默中断”，继续向上抛出
			if rec == http.ErrAbortHandler {
				panic(rec)
			}

			log := logger.FromContext(ctx.Request.Context())

			if isBrokenPipe(rec) {
				log.Warn("client connection broken",
					zap.Any("error", rec),
					zap.String("path", ctx.Request.URL.Path))
				_ = ctx.Error(fmt.Errorf("%v", rec))
				ctx.Abort()
				return
			}

			log.Error("panic recovered",
				zap.Any("error", rec),
				zap.String("method", ctx.Request.Method),
				zap.String("path", ctx.Request.URL.Path),
				zap.Stack("stack"),
			)

			if ctx.Writer.Written() {
				ctx.Abort()
				return
			}
			// 上面已记录含堆栈的日志，这里不再重复记录
			response.AbortSilently(ctx, errcode.Internal.Wrap(fmt.Errorf("panic: %v", rec)))
		}()
		ctx.Next()
	}
}

func isBrokenPipe(rec any) bool {
	err, ok := rec.(error)
	if !ok {
		return false
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var sysErr *os.SyscallError
		if errors.As(opErr, &sysErr) {
			msg := strings.ToLower(sysErr.Error())
			return strings.Contains(msg, "broken pipe") || strings.Contains(msg, "connection reset by peer")
		}
	}
	return false
}
