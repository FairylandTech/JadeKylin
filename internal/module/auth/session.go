/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 23:10:00 America/Los_Angeles
 *****************************************************************************/

// Package auth 是认证模块：验证码、登录、JWT 签发与校验、登录会话管理。
package auth

import "time"

// Session 是 Redis 中保存的登录会话。
type Session struct {
	SessionID string
	UserID    int64
	Username  string
	Token     string // 当前有效的 JWT（续签后更新）
	ClientIP  string
	UserAgent string
	LoginAt   time.Time
}
