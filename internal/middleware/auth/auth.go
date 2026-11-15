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

// Package auth 提供认证（Authentication）和鉴权（Authorization）中间件。
//
//	Authentication：校验 JWT 与 Redis 会话，把 user id 写入上下文，并为会话续期。
//	Authorization：按 "method + 路由模式" 查找接口放行权限，判断当前用户是否拥有该权限。
package auth

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/pkg/consts"
	"jkylin.com/kylin/v2/pkg/errcode"
	"jkylin.com/kylin/v2/pkg/response"
)

type principalContextKey struct{}

type TokenAuthenticator interface {
	Authenticate(ctx context.Context, rawToken string) (*model.Principal, string, error)
}

type APIAuthorizer interface {
	Authorize(ctx context.Context, userID int64, method, path string) (mapped, allowed bool, err error)
}

// Authentication 校验 Authorization: Bearer <jwt>，解析出 user id 并写入上下文：
// gin 上下文键 consts.UserIDKey 和 request context 都可以读取（见 UserID）。
// JWT 被续签时通过 Kylin-Access-Token 响应头返回新 token。
func Authentication(authenticator TokenAuthenticator) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		scheme, token, ok := strings.Cut(ctx.GetHeader("Authorization"), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
			ctx.Header("WWW-Authenticate", "Bearer")
			response.AbortSilently(ctx, errcode.Unauthorized)
			return
		}
		principal, renewed, err := authenticator.Authenticate(ctx.Request.Context(), strings.TrimSpace(token))
		if err != nil {
			if errcode.From(err).HTTPStatus < 500 {
				ctx.Header("WWW-Authenticate", `Bearer error="invalid_token"`)
				response.AbortSilently(ctx, err)
			} else {
				response.Abort(ctx, err)
			}
			return
		}
		if renewed != "" {
			ctx.Header(consts.AccessTokenHeader, renewed)
		}
		setPrincipal(ctx, principal)
		ctx.Next()
	}
}

// PrincipalResolver 返回本地 / 开发模式下默认使用的用户身份。
type PrincipalResolver func(ctx context.Context) (*model.Principal, error)

// LocalAuthentication 仅用于 local / dev 运行环境：请求携带 Authorization 头时按正常 JWT 认证，
// 否则跳过认证，以 resolve 返回的用户（默认初始管理员）身份继续，后续 Authorization 按该用户的权限鉴权。
func LocalAuthentication(authenticator TokenAuthenticator, resolve PrincipalResolver) gin.HandlerFunc {
	authenticate := Authentication(authenticator)
	return func(ctx *gin.Context) {
		if ctx.GetHeader("Authorization") != "" {
			authenticate(ctx)
			return
		}
		principal, err := resolve(ctx.Request.Context())
		if err != nil {
			response.Abort(ctx, err)
			return
		}
		setPrincipal(ctx, principal)
		ctx.Next()
	}
}

func setPrincipal(ctx *gin.Context, principal *model.Principal) {
	ctx.Set(consts.UserIDKey, principal.UserID)
	ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), principalContextKey{}, principal))
}

// Authorization 必须放在 Authentication 之后。接口以 "method + gin 路由模式"（如 GET /users/:id）
// 标识，放行权限保存在数据库中，每次请求实时判断；未登记放行权限的接口一律拒绝。
func Authorization(authorizer APIAuthorizer) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID, ok := UserID(ctx)
		if !ok {
			response.AbortSilently(ctx, errcode.Unauthorized)
			return
		}
		method, path := ctx.Request.Method, ctx.FullPath()
		mapped, allowed, err := authorizer.Authorize(ctx.Request.Context(), userID, method, path)
		switch {
		case err != nil:
			response.Abort(ctx, errcode.Internal.Wrap(err))
		case !mapped:
			response.Abort(ctx, errcode.Forbidden.WithMessage("no permission is configured for %s %s", method, path))
		case !allowed:
			response.AbortSilently(ctx, errcode.Forbidden.WithMessage("permission denied for %s %s", method, path))
		default:
			ctx.Next()
		}
	}
}

// Principal 返回当前登录用户；未经过 Authentication 时返回 nil。
// 同时接受 *gin.Context（不依赖 engine.ContextWithFallback）和 request context。
func Principal(ctx context.Context) *model.Principal {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		if ginCtx.Request == nil {
			return nil
		}
		ctx = ginCtx.Request.Context()
	}
	principal, _ := ctx.Value(principalContextKey{}).(*model.Principal)
	return principal
}

// UserID 返回 Authentication 从 JWT 中解析出的 user id。
func UserID(ctx context.Context) (int64, bool) {
	if principal := Principal(ctx); principal != nil {
		return principal.UserID, true
	}
	return 0, false
}
