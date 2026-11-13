/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 19:32:34 America/Los_Angeles
 *****************************************************************************/

package example

import (
	"github.com/gin-gonic/gin"
	"jkylin.com/kylin/v2/internal/handler"
	"jkylin.com/kylin/v2/internal/repository"
	"jkylin.com/kylin/v2/internal/service"
)

// Router 组装依赖（repository -> service -> handler）并注册路由。
func Router(engine *gin.Engine) {
	repo := repository.NewMemoryExampleRepository()
	svc := service.NewExampleService(repo)
	h := handler.NewExampleHandler(svc)

	router := engine.Group("/example")
	router.GET("", h.Example)

	items := router.Group("/items")
	items.POST("", h.Create)
	items.GET("", h.List)
	items.GET("/:id", h.Get)
	items.PUT("/:id", h.Update)
	items.DELETE("/:id", h.Delete)

	if gin.Mode() != gin.ReleaseMode {
		router.GET("/panic", h.Panic)
	}
}
