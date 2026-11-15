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

package errcode

import (
	"errors"
	"fmt"
	"net/http"
)

type Error struct {
	Code       int    // 业务码
	HTTPStatus int    // HTTP 状态码
	Message    string // 对外展示的信息
	cause      error  // 内部原因，仅记录日志，不返回给客户端
}

func New(code, httpStatus int, message string) *Error {
	return &Error{Code: code, HTTPStatus: httpStatus, Message: message}
}

func (codeError *Error) Error() string {
	if codeError.cause != nil {
		return fmt.Sprintf("code=%d message=%s cause=%v", codeError.Code, codeError.Message, codeError.cause)
	}
	return fmt.Sprintf("code=%d message=%s", codeError.Code, codeError.Message)
}

func (codeError *Error) Unwrap() error {
	return codeError.cause
}

func (codeError *Error) Is(target error) bool {
	var targetError *Error
	if errors.As(target, &targetError) {
		return codeError.Code == targetError.Code
	}
	return false
}

// WithMessage 返回替换了对外信息的副本。
func (codeError *Error) WithMessage(format string, arguments ...any) *Error {
	copied := *codeError
	copied.Message = fmt.Sprintf(format, arguments...)
	return &copied
}

// Wrap 返回附带内部原因的副本。
func (codeError *Error) Wrap(cause error) *Error {
	copied := *codeError
	copied.cause = cause
	return &copied
}

// From 把任意错误转换为 *Error；非 *Error 的错误统一视为 Internal。
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var codeError *Error
	if errors.As(err, &codeError) {
		return codeError
	}
	return Internal.Wrap(err)
}

var (
	Success           = New(200, http.StatusOK, "ok")
	InvalidParameters = New(40000, http.StatusBadRequest, "invalid params")
	Unauthorized      = New(40100, http.StatusUnauthorized, "unauthorized")
	Forbidden         = New(40300, http.StatusForbidden, "forbidden")
	NotFound          = New(40400, http.StatusNotFound, "resource not found")
	RouteNotFound     = New(40401, http.StatusNotFound, "route not found")
	MethodNotAllow    = New(40500, http.StatusMethodNotAllowed, "method not allowed")
	Conflict          = New(40900, http.StatusConflict, "resource conflict")
	TooManyRequests   = New(42900, http.StatusTooManyRequests, "too many requests")
	Internal          = New(50000, http.StatusInternalServerError, "internal server error")
)
