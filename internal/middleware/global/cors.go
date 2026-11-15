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
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/pkg/consts"
)

// CORS 跨域处理。allow_origins 包含 "*" 时允许任意来源。
func (globalMiddleware *Middleware) CORS() gin.HandlerFunc {
	allowAll := false
	allowedOrigins := make(map[string]struct{}, len(globalMiddleware.configuration.CORS.AllowOrigins))
	for _, allowedOrigin := range globalMiddleware.configuration.CORS.AllowOrigins {
		if allowedOrigin == "*" {
			allowAll = true
		}
		allowedOrigins[strings.TrimRight(allowedOrigin, "/")] = struct{}{}
	}

	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")
		if origin == "" {
			ctx.Next()
			return
		}

		header := ctx.Writer.Header()
		header.Add("Vary", "Origin")
		if allowAll {
			header.Set("Access-Control-Allow-Origin", "*")
		} else if _, ok := allowedOrigins[origin]; ok {
			header.Set("Access-Control-Allow-Origin", origin)
			header.Set("Access-Control-Allow-Credentials", "true")
		} else if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusForbidden)
			return
		} else {
			ctx.Next()
			return
		}

		header.Set("Access-Control-Expose-Headers", consts.TraceHeader+","+consts.AccessTokenHeader)

		if ctx.Request.Method == http.MethodOptions {
			header.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			header.Set("Access-Control-Allow-Headers", "Origin,Content-Type,Accept,Authorization,"+consts.TraceHeader)
			header.Set("Access-Control-Max-Age", "86400")
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}
}
