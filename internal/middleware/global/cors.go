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
func (m *Middleware) CORS() gin.HandlerFunc {
	allowAll := false
	allowed := make(map[string]struct{}, len(m.cfg.CORS.AllowOrigins))
	for _, o := range m.cfg.CORS.AllowOrigins {
		if o == "*" {
			allowAll = true
		}
		allowed[strings.TrimRight(o, "/")] = struct{}{}
	}

	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")
		if origin == "" {
			ctx.Next()
			return
		}

		h := ctx.Writer.Header()
		h.Add("Vary", "Origin")
		if allowAll {
			h.Set("Access-Control-Allow-Origin", "*")
		} else if _, ok := allowed[origin]; ok {
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
		} else if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusForbidden)
			return
		} else {
			ctx.Next()
			return
		}

		h.Set("Access-Control-Expose-Headers", consts.TraceHeader)

		if ctx.Request.Method == http.MethodOptions {
			h.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Origin,Content-Type,Accept,Authorization,"+consts.TraceHeader)
			h.Set("Access-Control-Max-Age", "86400")
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}
}
