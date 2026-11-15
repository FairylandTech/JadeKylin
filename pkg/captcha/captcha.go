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

// Length 是验证码的位数。
const Length = 4

// digitCount 是单个验证码字符的取值个数（0-9）。
const digitCount = 10

// NewCode 生成 Length 位随机数字。
func NewCode() (string, error) {
	code := make([]byte, Length)
	for index := range code {
		digit, err := rand.Int(rand.Reader, big.NewInt(digitCount))
		if err != nil {
			return "", err
		}
		code[index] = byte('0' + digit.Int64())
	}
	return string(code), nil
}
