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

package web

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"jkylin.com/kylin/v2/pkg/errcode"
)

// IdentifierURI 是路径中的记录 id（/:id）。
type IdentifierURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// BindJSON 绑定并校验 JSON 请求体。
func (BaseHandler) BindJSON(ctx *gin.Context, target any) error {
	return bindingError(ctx.ShouldBindWith(target, binding.JSON))
}

// BindQuery 绑定并校验查询参数。
func (BaseHandler) BindQuery(ctx *gin.Context, target any) error {
	return bindingError(ctx.ShouldBindWith(target, binding.Query))
}

// BindURI 绑定并校验路径参数。
func (BaseHandler) BindURI(ctx *gin.Context, target any) error {
	return bindingError(ctx.ShouldBindUri(target))
}

// BindIdentifier 绑定并校验路径中的记录 id（/:id）。
func (BaseHandler) BindIdentifier(ctx *gin.Context) (int64, error) {
	var uri IdentifierURI
	if err := bindingError(ctx.ShouldBindUri(&uri)); err != nil {
		return 0, err
	}
	return uri.ID, nil
}

// bindingError 将绑定 / 校验错误转换为可读的 InvalidParameters。
func bindingError(err error) error {
	if err == nil {
		return nil
	}
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		messages := make([]string, 0, len(validationErrors))
		for _, fieldError := range validationErrors {
			messages = append(messages, fieldMessage(fieldError))
		}
		return errcode.InvalidParameters.WithMessage("%s", strings.Join(messages, "; ")).Wrap(err)
	}
	return errcode.InvalidParameters.WithMessage("malformed request: %v", err).Wrap(err)
}

func fieldMessage(fieldError validator.FieldError) string {
	field := strings.ToLower(fieldError.Field())
	switch fieldError.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, fieldError.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, fieldError.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of [%s]", field, fieldError.Param())
	default:
		return fmt.Sprintf("%s failed on '%s'", field, fieldError.Tag())
	}
}
