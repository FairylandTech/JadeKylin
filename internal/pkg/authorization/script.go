/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-15 10:00:00 America/Los_Angeles
 *****************************************************************************/

package authorization

import (
	_ "embed"

	"github.com/redis/go-redis/v9"
)

// Lua 脚本单独存放在 script 目录，编译时嵌入。

//go:embed script/check.lua
var checkPermissionSource string

// checkPermissionScript 判断用户是否拥有权限，返回值对应 CheckResult。
var checkPermissionScript = redis.NewScript(checkPermissionSource)
