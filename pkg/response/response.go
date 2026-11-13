/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 04:01:18 America/Los_Angeles
 *****************************************************************************/

package response

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"jkylin.com/kylin/v2/pkg/consts"
	"jkylin.com/kylin/v2/pkg/errcode"
	"jkylin.com/kylin/v2/pkg/logger"
)

type Pagination struct {
	Page  int   `json:"page"`
	Size  int   `json:"size"`
	Total int64 `json:"total"`
	Pages int   `json:"pages"`
	Data  any   `json:"data"`
}

type Response struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	TraceID   string `json:"trace_id"`
	Timestamp int64  `json:"timestamp"`
	Duration  int64  `json:"duration"` // 处理耗时（毫秒）
}

func Wrapper(ctx *gin.Context, httpStatus int, response *Response) {
	response.TraceID = ctx.GetString(consts.TraceKey)
	now := time.Now()
	if response.Timestamp == 0 {
		response.Timestamp = now.UnixMilli()
	}
	if start, ok := ctx.Get(consts.RequestTimestamp); ok {
		if t, ok := start.(time.Time); ok {
			response.Duration = now.Sub(t).Milliseconds()
		}
	}
	if response.Message == "" {
		response.Message = "ok"
	}

	if response.TraceID != "" {
		ctx.Header(consts.TraceHeader, response.TraceID)
	}
	ctx.JSON(httpStatus, response)
}

func Success(ctx *gin.Context, data any) {
	Wrapper(ctx, errcode.Success.HTTPStatus, &Response{Code: errcode.Success.Code, Data: data})
}

func Page(ctx *gin.Context, data any, page, size int, total int64) {
	pages := 0
	if size > 0 {
		pages = int((total + int64(size) - 1) / int64(size))
	}
	Success(ctx, Pagination{Page: page, Size: size, Total: total, Pages: pages, Data: data})
}

// FailSkip 同 Fail，skip 用于修正日志中的调用位置（被封装调用时 +1）。
func FailSkip(ctx *gin.Context, err error, skip int) {
	e := errcode.From(err)
	if e == nil {
		e = errcode.Internal
	}

	log := logger.FromContext(ctx.Request.Context()).WithOptions(zap.AddCallerSkip(skip))
	if e.HTTPStatus >= 500 {
		log.Error("request failed", zap.Int("code", e.Code), zap.Error(err))
	} else {
		log.Warn("request failed", zap.Int("code", e.Code), zap.Error(err))
	}

	_ = ctx.Error(err) // 记录到 gin 的错误列表，供访问日志使用
	Wrapper(ctx, e.HTTPStatus, &Response{Code: e.Code, Message: e.Message})
}

// Fail 错误响应。*errcode.Error 原样输出；其它错误统一视为 500，
// 内部原因只写日志，不暴露给客户端。
func Fail(ctx *gin.Context, err error) {
	FailSkip(ctx, err, 1)
}

// Abort 输出错误响应并终止后续 handler，用于中间件。
func Abort(ctx *gin.Context, err error) {
	FailSkip(ctx, err, 2)
	ctx.Abort()
}

// AbortSilently 输出错误响应并终止，但不打日志（调用方已自行记录）。
func AbortSilently(ctx *gin.Context, err error) {
	e := errcode.From(err)
	_ = ctx.Error(err)
	Wrapper(ctx, e.HTTPStatus, &Response{Code: e.Code, Message: e.Message})
	ctx.Abort()
}
