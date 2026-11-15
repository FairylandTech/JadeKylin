/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-15 01:30:00 America/Los_Angeles
 *****************************************************************************/

// Package security 提供认证（Authentication）和鉴权（Authorization）中间件。
//
//	Authentication：校验 JWT 与 Redis 会话，把 user id 写入上下文，并为会话续期。
//	Authorization：按 "method + 路由模式" 查找接口放行权限，判断当前用户是否拥有该权限。
package security

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/pkg/identity"
	"jkylin.com/kylin/v2/pkg/consts"
	"jkylin.com/kylin/v2/pkg/errcode"
	"jkylin.com/kylin/v2/pkg/response"
)

const (
	authorizationHeader   = "Authorization"
	authenticateHeader    = "WWW-Authenticate"
	bearerScheme          = "Bearer"
	invalidTokenChallenge = `Bearer error="invalid_token"`
)

type TokenAuthenticator interface {
	Authenticate(ctx context.Context, rawToken string) (*identity.Principal, string, error)
}

// PrincipalResolver 返回本地 / 开发模式下默认使用的用户身份。
type PrincipalResolver func(ctx context.Context) (*identity.Principal, error)

// Authentication 校验 Authorization: Bearer <jwt>，解析出 user id 并写入上下文：
// gin 上下文键 consts.UserIDKey 和 request context 都可以读取（见 identity.UserID）。
// JWT 被续签时通过 Kylin-Access-Token 响应头返回新 token。
func Authentication(authenticator TokenAuthenticator) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		scheme, token, ok := strings.Cut(ctx.GetHeader(authorizationHeader), " ")
		if !ok || !strings.EqualFold(scheme, bearerScheme) || strings.TrimSpace(token) == "" {
			ctx.Header(authenticateHeader, bearerScheme)
			response.AbortSilently(ctx, errcode.Unauthorized)
			return
		}
		principal, renewedToken, err := authenticator.Authenticate(ctx.Request.Context(), strings.TrimSpace(token))
		if err != nil {
			if errcode.From(err).HTTPStatus < 500 {
				ctx.Header(authenticateHeader, invalidTokenChallenge)
				response.AbortSilently(ctx, err)
			} else {
				response.Abort(ctx, err)
			}
			return
		}
		if renewedToken != "" {
			ctx.Header(consts.AccessTokenHeader, renewedToken)
		}
		identity.Attach(ctx, principal)
		ctx.Next()
	}
}

// LocalAuthentication 仅用于 local / dev 运行环境：请求携带 Authorization 头时按正常 JWT 认证，
// 否则跳过认证，以 resolve 返回的用户（默认初始管理员）身份继续，后续 Authorization 按该用户的权限鉴权。
func LocalAuthentication(authenticator TokenAuthenticator, resolve PrincipalResolver) gin.HandlerFunc {
	authenticate := Authentication(authenticator)
	return func(ctx *gin.Context) {
		if ctx.GetHeader(authorizationHeader) != "" {
			authenticate(ctx)
			return
		}
		principal, err := resolve(ctx.Request.Context())
		if err != nil {
			response.Abort(ctx, err)
			return
		}
		identity.Attach(ctx, principal)
		ctx.Next()
	}
}
