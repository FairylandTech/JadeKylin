/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 21:30:00 America/Los_Angeles
 *****************************************************************************/

// Package validation 提供服务层公共的入参校验与规范化。
package validation

import (
	"strings"

	"jkylin.com/kylin/v2/pkg/errcode"
)

// RequireText 去除首尾空白，结果为空时返回 InvalidParameters。
func RequireText(field, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errcode.InvalidParameters.WithMessage("%s must not be blank", field)
	}
	return value, nil
}
