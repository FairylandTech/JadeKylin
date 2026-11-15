/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 23:15:00 America/Los_Angeles
 *****************************************************************************/

package apipermission

import (
	_ "embed"

	"github.com/redis/go-redis/v9"
)

// Lua 脚本单独存放在 script 目录，编译时嵌入。

//go:embed script/check.lua
var checkAuthorizationSource string

// checkAuthorizationScript 判断用户能否访问接口，返回值对应 AuthorizationResult。
var checkAuthorizationScript = redis.NewScript(checkAuthorizationSource)
