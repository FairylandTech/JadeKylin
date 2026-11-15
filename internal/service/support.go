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

package service

import (
	"context"
	"slices"
	"strings"

	"jkylin.com/kylin/v2/internal/model/valueobject"
	"jkylin.com/kylin/v2/pkg/errcode"
)

type existingCodesFunc func(ctx context.Context, codes []string) ([]string, error)

// normalizeCodes 去除首尾空白、去重，并通过 existing 确认每个 code 都已存在且启用。
func normalizeCodes(ctx context.Context, field string, codes []string, existing existingCodesFunc) (valueobject.StringArray, error) {
	normalized := make(valueobject.StringArray, 0, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			return nil, errcode.InvalidParams.WithMessage("%s must not contain blank values", field)
		}
		if !slices.Contains(normalized, code) {
			normalized = append(normalized, code)
		}
	}
	found, err := existing(ctx, normalized)
	if err != nil {
		return nil, errcode.Internal.Wrap(err)
	}
	var unknown []string
	for _, code := range normalized {
		if !slices.Contains(found, code) {
			unknown = append(unknown, code)
		}
	}
	if len(unknown) > 0 {
		return nil, errcode.InvalidParams.WithMessage("unknown %s: %s", field, strings.Join(unknown, ", "))
	}
	return normalized, nil
}

// AuthzInvalidator 失效 Redis 中的鉴权缓存（*repository.AuthzCache 实现）。
type AuthzInvalidator interface {
	InvalidateUser(ctx context.Context, userID int64) error
	InvalidateAllUsers(ctx context.Context) error
}

func authzSyncError(err error) error {
	return errcode.Internal.WithMessage("saved but failed to refresh permission cache").Wrap(err)
}

func requireText(field, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errcode.InvalidParams.WithMessage("%s must not be blank", field)
	}
	return value, nil
}
