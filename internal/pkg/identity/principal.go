/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 23:30:00 America/Los_Angeles
 *****************************************************************************/

// Package identity 保存并读取当前登录用户（Principal）。
package identity

import (
	"context"

	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/pkg/consts"
	"jkylin.com/kylin/v2/pkg/errcode"
)

// Principal 是通过 JWT 认证的当前登录用户。
type Principal struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	SessionID string `json:"session_id"`
}

type principalContextKey struct{}

// Attach 把当前登录用户写入上下文：gin 上下文键 consts.UserIDKey 和 request context 都可以读取。
func Attach(ctx *gin.Context, principal *Principal) {
	ctx.Set(consts.UserIDKey, principal.UserID)
	ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), principalContextKey{}, principal))
}

// Current 返回当前登录用户；未经过认证时返回 nil。
// 同时接受 *gin.Context（不依赖 engine.ContextWithFallback）和 request context。
func Current(ctx context.Context) *Principal {
	if ginContext, ok := ctx.(*gin.Context); ok {
		if ginContext.Request == nil {
			return nil
		}
		ctx = ginContext.Request.Context()
	}
	principal, _ := ctx.Value(principalContextKey{}).(*Principal)
	return principal
}

// Require 返回当前登录用户；未经过认证时返回 Unauthorized。
func Require(ctx context.Context) (*Principal, error) {
	principal := Current(ctx)
	if principal == nil {
		return nil, errcode.Unauthorized
	}
	return principal, nil
}

// UserID 返回认证中间件从 JWT 中解析出的 user id。
func UserID(ctx context.Context) (int64, bool) {
	if principal := Current(ctx); principal != nil {
		return principal.UserID, true
	}
	return 0, false
}
