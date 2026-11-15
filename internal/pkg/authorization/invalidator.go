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

// Package authorization 提供各业务模块共用的鉴权缓存失效约定。
package authorization

import (
	"context"

	"jkylin.com/kylin/v2/pkg/errcode"
)

// Invalidator 失效 Redis 中的鉴权缓存（*apipermission.AuthorizationCache 实现）。
type Invalidator interface {
	InvalidateUser(ctx context.Context, userID int64) error
	InvalidateAllUsers(ctx context.Context) error
}

// CacheRefreshError 表示数据已保存但鉴权缓存刷新失败。
func CacheRefreshError(err error) error {
	return errcode.Internal.WithMessage("saved but failed to refresh permission cache").Wrap(err)
}
