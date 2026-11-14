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

package support

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"jkylin.com/kylin/v2/pkg/errcode"
	"jkylin.com/kylin/v2/pkg/logger"
	"jkylin.com/kylin/v2/pkg/response"
)

// HandlerSupportBase 所有 handler 嵌入的基础能力：统一响应、参数绑定、日志。
type HandlerSupportBase struct{}

func (HandlerSupportBase) Success(ctx *gin.Context, data any) {
	response.Success(ctx, data)
}

func (HandlerSupportBase) Page(ctx *gin.Context, data any, page, size int, total int64) {
	response.Page(ctx, data, page, size, total)
}

func (HandlerSupportBase) Fail(ctx *gin.Context, err error) {
	response.FailSkip(ctx, err, 2)
}

// Logger 返回携带 trace_id 的请求级 Logger。
func (HandlerSupportBase) Logger(ctx *gin.Context) *zap.Logger {
	return logger.FromContext(ctx.Request.Context())
}

// BindJSON 绑定并校验 JSON 请求体。
func (HandlerSupportBase) BindJSON(ctx *gin.Context, obj any) error {
	return bindErr(ctx.ShouldBindWith(obj, binding.JSON))
}

// BindQuery 绑定并校验查询参数。
func (HandlerSupportBase) BindQuery(ctx *gin.Context, obj any) error {
	return bindErr(ctx.ShouldBindWith(obj, binding.Query))
}

// BindURI 绑定并校验路径参数。
func (HandlerSupportBase) BindURI(ctx *gin.Context, obj any) error {
	return bindErr(ctx.ShouldBindUri(obj))
}

// bindErr 将绑定 / 校验错误转换为可读的 InvalidParams。
func bindErr(err error) error {
	if err == nil {
		return nil
	}
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		msgs := make([]string, 0, len(verrs))
		for _, fe := range verrs {
			msgs = append(msgs, fieldMessage(fe))
		}
		return errcode.InvalidParams.WithMessage("%s", strings.Join(msgs, "; ")).Wrap(err)
	}
	return errcode.InvalidParams.WithMessage("malformed request: %v", err).Wrap(err)
}

func fieldMessage(fe validator.FieldError) string {
	field := strings.ToLower(fe.Field())
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, fe.Param())
	default:
		return fmt.Sprintf("%s failed on '%s'", field, fe.Tag())
	}
}
