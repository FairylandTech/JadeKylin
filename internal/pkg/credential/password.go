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

// Package credential 提供密码哈希、校验与随机密码生成。
package credential

import (
	"crypto/rand"
	"math/big"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
	"jkylin.com/kylin/v2/pkg/errcode"
)

const (
	// maximumPasswordBytes 是 bcrypt 能处理的最大密码长度。
	maximumPasswordBytes = 72
	// generatedPasswordAlphabet 去掉了易混淆的 0 O 1 l I。
	generatedPasswordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
)

// dummyPasswordHash 用于用户名不存在时仍执行一次 bcrypt 比较，避免通过响应耗时枚举用户名。
var dummyPasswordHash = sync.OnceValue(func() []byte {
	hash, _ := bcrypt.GenerateFromPassword([]byte("kylin-dummy-password"), bcrypt.DefaultCost)
	return hash
})

// HashPassword 校验密码格式并返回 bcrypt 哈希。
func HashPassword(password string) (string, error) {
	if strings.TrimSpace(password) == "" || len(password) > maximumPasswordBytes {
		return "", errcode.InvalidParameters.WithMessage("password must be non-blank and at most 72 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", errcode.Internal.Wrap(err)
	}
	return string(hash), nil
}

// MatchesPassword 判断明文密码是否与 bcrypt 哈希匹配。
func MatchesPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// CompareWithDummyPassword 执行一次结果必然失败的 bcrypt 比较，让 "用户不存在" 与 "密码错误" 的耗时一致。
func CompareWithDummyPassword(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyPasswordHash(), []byte(password))
}

// GeneratePassword 生成指定长度的随机密码。
func GeneratePassword(length int) (string, error) {
	limit := big.NewInt(int64(len(generatedPasswordAlphabet)))
	password := make([]byte, length)
	for index := range password {
		position, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		password[index] = generatedPasswordAlphabet[position.Int64()]
	}
	return string(password), nil
}
