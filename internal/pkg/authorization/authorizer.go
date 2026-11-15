/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-15 10:00:00 America/Los_Angeles
 *****************************************************************************/

// Package authorization 提供业务服务内的权限判断：每个服务方法自行声明需要的权限 code，
// 通过 Authorizer.Require 判断当前登录用户是否拥有该权限（用户权限缓存在 Redis）。
package authorization

import (
	"context"
	"errors"
	"fmt"

	"jkylin.com/kylin/v2/internal/pkg/identity"
	"jkylin.com/kylin/v2/pkg/errcode"
)

// checkMaximumRounds 鉴权最多补一次缓存：用户权限缺失时从数据库加载后再查一次。
const checkMaximumRounds = 2

// PermissionLoader 从数据库读取用户拥有的全部权限 code（*user.Repository 实现）。
type PermissionLoader interface {
	PermissionCodes(ctx context.Context, userID int64) ([]string, error)
}

// Authorizer 判断当前登录用户是否拥有某个权限，并负责失效用户权限缓存。
//
// nil *Authorizer 表示未启用登录（例如启动时创建初始管理员）：Require 一律拒绝，失效操作直接跳过。
type Authorizer struct {
	cache            *Cache
	permissionLoader PermissionLoader
}

func NewAuthorizer(cache *Cache, permissionLoader PermissionLoader) *Authorizer {
	return &Authorizer{cache: cache, permissionLoader: permissionLoader}
}

// Require 要求当前登录用户（从 ctx 中读取）拥有 permission：未登录返回 Unauthorized，没有权限返回 Forbidden。
// 数据全部从 Redis 读取；缓存缺失时（首次请求、Redis 重启、缓存被失效）才从数据库加载一次。
func (authorizer *Authorizer) Require(ctx context.Context, permission string) error {
	if authorizer == nil {
		return errcode.Internal.WithMessage("authorization is not configured")
	}
	userID, ok := identity.UserID(ctx)
	if !ok {
		return errcode.Unauthorized
	}
	for range checkMaximumRounds {
		result, err := authorizer.cache.Check(ctx, userID, permission)
		if err != nil {
			return errcode.Internal.Wrap(err)
		}
		switch result {
		case CheckUserMissing:
			if err := authorizer.loadUser(ctx, userID); err != nil {
				return errcode.Internal.Wrap(err)
			}
		case CheckDenied:
			return errcode.Forbidden.WithMessage("permission denied: %s is required", permission)
		case CheckAllowed:
			return nil
		default:
			return errcode.Internal.Wrap(fmt.Errorf("unexpected authorization cache result %d", result))
		}
	}
	return errcode.Internal.Wrap(errors.New("authorization cache did not settle"))
}

// loadUser 从数据库加载用户权限并写入缓存。
func (authorizer *Authorizer) loadUser(ctx context.Context, userID int64) error {
	permissions, err := authorizer.permissionLoader.PermissionCodes(ctx, userID)
	if err != nil {
		return err
	}
	return authorizer.cache.SetUserPermissions(ctx, userID, permissions)
}

// InvalidateUser 用户角色、状态变更或删除后删除其权限缓存，下次请求从数据库重新加载。
func (authorizer *Authorizer) InvalidateUser(ctx context.Context, userID int64) error {
	if authorizer == nil {
		return nil
	}
	if err := authorizer.cache.InvalidateUser(ctx, userID); err != nil {
		return cacheRefreshError(err)
	}
	return nil
}

// InvalidateAllUsers 角色变更可能影响任意用户，删除所有用户的权限缓存。
func (authorizer *Authorizer) InvalidateAllUsers(ctx context.Context) error {
	if authorizer == nil {
		return nil
	}
	if err := authorizer.cache.InvalidateAllUsers(ctx); err != nil {
		return cacheRefreshError(err)
	}
	return nil
}

// cacheRefreshError 表示数据已保存但权限缓存刷新失败。
func cacheRefreshError(err error) error {
	return errcode.Internal.WithMessage("saved but failed to refresh permission cache").Wrap(err)
}
