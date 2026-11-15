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

package auth

import (
	_ "embed"

	"github.com/redis/go-redis/v9"
)

// Lua 脚本单独存放在 script 目录，编译时嵌入。

//go:embed script/create.lua
var createSessionSource string

//go:embed script/touch.lua
var touchSessionSource string

//go:embed script/revoke.lua
var revokeSessionsSource string

var (
	// createSessionScript 清理已过期的会话，按上限踢掉最早登录的会话，然后写入新会话。返回被踢掉的 session id。
	createSessionScript = redis.NewScript(createSessionSource)
	// touchSessionScript 会话存在时续期（可选更新 token），返回 1；不存在返回 0。
	touchSessionScript = redis.NewScript(touchSessionSource)
	// revokeSessionsScript 删除用户的所有会话及会话索引。
	revokeSessionsScript = redis.NewScript(revokeSessionsSource)
)
