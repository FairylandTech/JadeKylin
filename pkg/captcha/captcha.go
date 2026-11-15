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

// Package captcha 生成数字验证码。
package captcha

import (
	"crypto/rand"
	"math/big"
)

const Length = 4

// NewCode 生成 Length 位随机数字。
func NewCode() (string, error) {
	code := make([]byte, Length)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code[i] = byte('0' + n.Int64())
	}
	return string(code), nil
}
