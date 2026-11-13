/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 21:51:58 America/Los_Angeles
 *****************************************************************************/

// Package global 全局中间件：链路追踪、访问日志、异常恢复、跨域、限流。
package global

import (
	"jkylin.com/kylin/v2/internal/config"
)

type Middleware struct {
	cfg *config.Config
}

func NewGlobalMiddleware(cfg *config.Config) *Middleware {
	if cfg == nil {
		cfg = config.Default()
	}
	return &Middleware{cfg: cfg}
}
