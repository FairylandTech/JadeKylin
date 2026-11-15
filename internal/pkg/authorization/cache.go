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

package authorization

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	userKeyPrefix     = "kylin:authz:user:" // SET: 单个用户的权限集合
	userKeyPattern    = userKeyPrefix + "*" // 所有用户权限集合
	sentinel          = ""                  // 占位：区分 "已缓存但为空" 和 "未缓存"
	invalidationBatch = 500                 // 批量失效时每批扫描 / 删除的 key 数
)

// CheckResult 是一次权限缓存查询的结果。
type CheckResult int

const (
	CheckUserMissing CheckResult = -1 // 用户权限未缓存，需要从数据库加载
	CheckDenied      CheckResult = 0
	CheckAllowed     CheckResult = 1
)

// Cache 在 Redis 中缓存用户权限，登录后的请求鉴权不访问数据库。
//
//	kylin:authz:user:<uid>   SET  用户通过启用角色拥有的权限；按需加载，空闲 TTL 后过期，用户 / 角色变更时删除
type Cache struct {
	client         *redis.Client
	userTimeToLive time.Duration
}

func NewCache(client *redis.Client, userTimeToLive time.Duration) *Cache {
	return &Cache{client: client, userTimeToLive: userTimeToLive}
}

func userKey(userID int64) string {
	return userKeyPrefix + strconv.FormatInt(userID, 10)
}

// Check 判断用户是否拥有权限，一次 Redis 往返完成；命中时为用户权限续期。
func (cache *Cache) Check(ctx context.Context, userID int64, permission string) (CheckResult, error) {
	result, err := checkPermissionScript.Run(ctx, cache.client, []string{userKey(userID)},
		permission, cache.userTimeToLive.Milliseconds()).Int()
	return CheckResult(result), err
}

// SetUserPermissions 缓存用户的权限集合（可以为空）。
func (cache *Cache) SetUserPermissions(ctx context.Context, userID int64, permissions []string) error {
	key := userKey(userID)
	members := make([]any, 0, len(permissions)+1)
	members = append(members, sentinel)
	for _, permission := range permissions {
		members = append(members, permission)
	}
	_, err := cache.client.TxPipelined(ctx, func(pipeline redis.Pipeliner) error {
		pipeline.Del(ctx, key)
		pipeline.SAdd(ctx, key, members...)
		pipeline.PExpire(ctx, key, cache.userTimeToLive)
		return nil
	})
	return err
}

// InvalidateUser 删除单个用户的权限缓存（用户角色、状态变更或删除时调用）。
func (cache *Cache) InvalidateUser(ctx context.Context, userID int64) error {
	return cache.client.Del(ctx, userKey(userID)).Err()
}

// InvalidateAllUsers 删除所有用户的权限缓存（角色新增、修改、删除时调用）。
func (cache *Cache) InvalidateAllUsers(ctx context.Context) error {
	// 先完整扫描再删除，避免边扫描边删除导致游标跳过部分 key。
	var keys []string
	iterator := cache.client.Scan(ctx, 0, userKeyPattern, invalidationBatch).Iterator()
	for iterator.Next(ctx) {
		keys = append(keys, iterator.Val())
	}
	if err := iterator.Err(); err != nil {
		return err
	}
	for start := 0; start < len(keys); start += invalidationBatch {
		batch := keys[start:min(start+invalidationBatch, len(keys))]
		if err := cache.client.Unlink(ctx, batch...).Err(); err != nil {
			return err
		}
	}
	return nil
}
