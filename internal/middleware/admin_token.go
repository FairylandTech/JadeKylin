/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 21:30:00 America/Los_Angeles
 *****************************************************************************/

package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"strings"

	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/pkg/errcode"
	"jkylin.com/kylin/v2/pkg/response"
)

// AdminToken 校验 Authorization: Bearer <token>，用于用户、角色、权限等管理接口。
func AdminToken(adminToken string) gin.HandlerFunc {
	expected := sha256.Sum256([]byte(adminToken))
	return func(ctx *gin.Context) {
		scheme, token, ok := strings.Cut(ctx.GetHeader("Authorization"), " ")
		actual := sha256.Sum256([]byte(token))
		if !ok || !strings.EqualFold(scheme, "Bearer") || subtle.ConstantTimeCompare(expected[:], actual[:]) != 1 {
			ctx.Header("WWW-Authenticate", "Bearer")
			response.AbortSilently(ctx, errcode.Unauthorized)
		}
	}
}
