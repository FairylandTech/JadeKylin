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

package validation

import (
	"context"
	"slices"
	"strings"

	"jkylin.com/kylin/v2/internal/pkg/valueobject"
	"jkylin.com/kylin/v2/pkg/errcode"
)

// ExistingCodesFinder 返回 codes 中已存在且启用的 code。
type ExistingCodesFinder func(ctx context.Context, codes []string) ([]string, error)

// NormalizeCodes 去除首尾空白、去重，并通过 existing 确认每个 code 都已存在且启用。
func NormalizeCodes(ctx context.Context, field string, codes []string, existing ExistingCodesFinder) (valueobject.StringArray, error) {
	normalized := make(valueobject.StringArray, 0, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			return nil, errcode.InvalidParameters.WithMessage("%s must not contain blank values", field)
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
		return nil, errcode.InvalidParameters.WithMessage("unknown %s: %s", field, strings.Join(unknown, ", "))
	}
	return normalized, nil
}
