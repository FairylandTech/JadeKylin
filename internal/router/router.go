/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 04:20:18 America/Los_Angeles
 *****************************************************************************/

package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"jkylin.com/kylin/v2/internal/router/user"
	"jkylin.com/kylin/v2/pkg/response"
)

func Init(engine *gin.Engine, db *gorm.DB, userAdminToken string) {
	// 健康检查（已在访问日志中跳过）
	engine.GET("/healthz", func(ctx *gin.Context) { ctx.String(http.StatusOK, "ok") })

	engine.GET("/example", func(ctx *gin.Context) {
		response.Success(ctx, gin.H{"example": "Hello Gin Server."})
	})
	user.Router(engine, db, userAdminToken)
}
