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
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"jkylin.com/kylin/v2/internal/config"
)

// DefaultLocalUsername 是未设置 KYLIN_ADMIN_USERNAME 时，跳过认证所使用的用户名。
const DefaultLocalUsername = "admin"

type Dependencies struct {
	Database *gorm.DB
	Redis    *redis.Client
	Auth     config.Auth // Database、Redis、JWTSecret 都具备时才注册 /auth 和管理接口
	// SkipAuth 为 true（RUN_MODE=local/dev）时，未携带 Authorization 的请求以 LocalUsername 用户身份访问。
	SkipAuth      bool
	LocalUsername string
}

// Enabled 返回登录及管理接口是否可用。
func (dependencies Dependencies) Enabled() bool {
	return dependencies.Database != nil && dependencies.Redis != nil && dependencies.Auth.JWTSecret != ""
}
