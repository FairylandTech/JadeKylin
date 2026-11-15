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

// Package identifier 生成随机标识。
package identifier

import (
	"crypto/rand"
	"encoding/hex"
)

// randomBytes 是随机标识的字节数，十六进制编码后为 32 个字符。
const randomBytes = 16

// NewRandom 返回 32 位十六进制随机标识，用作验证码 id、会话 id、缓存同步 id 等。
func NewRandom() (string, error) {
	var buffer [randomBytes]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer[:]), nil
}
