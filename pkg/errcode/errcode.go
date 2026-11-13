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

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("code=%d message=%s cause=%v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("code=%d message=%s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	return e.cause
}

func (e *Error) Is(target error) bool {
	var t *Error
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}

func (e *Error) WithMessage(format string, args ...any) *Error {
	cp := *e
	cp.Message = fmt.Sprintf(format, args...)
	return &cp
}

func (e *Error) Wrap(cause error) *Error {
	cp := *e
	cp.cause = cause
	return &cp
}

func From(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return Internal.Wrap(err)
}

var (
	Success         = New(200, http.StatusOK, "ok")
	InvalidParams   = New(40000, http.StatusBadRequest, "invalid params")
	Unauthorized    = New(40100, http.StatusUnauthorized, "unauthorized")
	Forbidden       = New(40300, http.StatusForbidden, "forbidden")
	NotFound        = New(40400, http.StatusNotFound, "resource not found")
	RouteNotFound   = New(40401, http.StatusNotFound, "route not found")
	MethodNotAllow  = New(40500, http.StatusMethodNotAllowed, "method not allowed")
	Conflict        = New(40900, http.StatusConflict, "resource conflict")
	TooManyRequests = New(42900, http.StatusTooManyRequests, "too many requests")
	Internal        = New(50000, http.StatusInternalServerError, "internal server error")
)
