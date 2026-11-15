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

package security

import (
	"context"

	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/pkg/identity"
	"jkylin.com/kylin/v2/pkg/errcode"
	"jkylin.com/kylin/v2/pkg/response"
)

type APIAuthorizer interface {
	Authorize(ctx context.Context, userID int64, method, path string) (mapped, allowed bool, err error)
}

// Authorization 必须放在 Authentication 之后。接口以 "method + gin 路由模式"（如 GET /users/:id）
// 标识，放行权限保存在数据库中，每次请求实时判断；未登记放行权限的接口一律拒绝。
func Authorization(authorizer APIAuthorizer) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID, ok := identity.UserID(ctx)
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
