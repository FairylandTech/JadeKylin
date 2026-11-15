/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-15 02:30:00 America/Los_Angeles
 *****************************************************************************/

package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"jkylin.com/kylin/v2/internal/model"
)

const (
	authzKeyPrefix      = "kylin:authz:"
	authzAPIKey         = authzKeyPrefix + "api"       // HASH: "METHOD path" -> permission
	authzUserKeyPattern = authzKeyPrefix + "user:*"    // 所有用户权限集合
	authzSentinel       = ""                           // 占位：区分 "已缓存但为空" 和 "未缓存"
	authzSyncTempPrefix = authzKeyPrefix + "api:sync:" // 同步时的临时 key
)

func authzUserKey(userID int64) string {
	return authzKeyPrefix + "user:" + strconv.FormatInt(userID, 10)
}

// AuthzResult 是一次鉴权缓存查询的结果。
type AuthzResult int

const (
	AuthzAPIMissing  AuthzResult = -1 // 接口权限表未缓存，需要从数据库同步
	AuthzUserMissing AuthzResult = -2 // 用户权限未缓存，需要从数据库加载
	AuthzNotMapped   AuthzResult = 0  // 接口未登记放行权限
	AuthzDenied      AuthzResult = 1
	AuthzAllowed     AuthzResult = 2
)

// AuthzCache 在 Redis 中缓存鉴权数据，登录后的请求鉴权不访问数据库。
//
//	kylin:authz:api          HASH "METHOD path" -> 放行权限；api_permissions 变更后整体重建
//	kylin:authz:user:<uid>   SET  用户通过启用角色拥有的权限；按需加载，空闲 TTL 后过期，用户 / 角色变更时删除
type AuthzCache struct {
	client  *redis.Client
	userTTL time.Duration
}

func NewAuthzCache(client *redis.Client, userTTL time.Duration) *AuthzCache {
	return &AuthzCache{client: client, userTTL: userTTL}
}

var authzCheckScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then return -1 end
if redis.call('EXISTS', KEYS[2]) == 0 then return -2 end
redis.call('PEXPIRE', KEYS[2], ARGV[2])
local permission = redis.call('HGET', KEYS[1], ARGV[1])
if not permission or ARGV[1] == '' then return 0 end
if redis.call('SISMEMBER', KEYS[2], permission) == 1 then return 2 end
return 1
`)

// Check 判断用户能否访问接口，一次 Redis 往返完成；命中时为用户权限续期。
func (cache *AuthzCache) Check(ctx context.Context, userID int64, method, path string) (AuthzResult, error) {
	result, err := authzCheckScript.Run(ctx, cache.client, []string{authzAPIKey, authzUserKey(userID)},
		method+" "+path, cache.userTTL.Milliseconds()).Int()
	return AuthzResult(result), err
}

// SyncAPIPermissions 用最新的接口放行权限整体替换缓存：先写临时 key 再 RENAME，读者不会看到半成品。
func (cache *AuthzCache) SyncAPIPermissions(ctx context.Context, records []model.ApiPermission, syncID string) error {
	temp := authzSyncTempPrefix + syncID
	values := make([]any, 0, 2+2*len(records))
	values = append(values, authzSentinel, authzSentinel)
	for _, record := range records {
		values = append(values, record.Method+" "+record.Path, record.Permission)
	}
	_, err := cache.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, temp)
		pipe.HSet(ctx, temp, values...)
		pipe.Rename(ctx, temp, authzAPIKey)
		return nil
	})
	return err
}

// SetUserPermissions 缓存用户的权限集合（可以为空）。
func (cache *AuthzCache) SetUserPermissions(ctx context.Context, userID int64, permissions []string) error {
	key := authzUserKey(userID)
	members := make([]any, 0, len(permissions)+1)
	members = append(members, authzSentinel)
	for _, permission := range permissions {
		members = append(members, permission)
	}
	_, err := cache.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, key)
		pipe.SAdd(ctx, key, members...)
		pipe.PExpire(ctx, key, cache.userTTL)
		return nil
	})
	return err
}

// InvalidateUser 删除单个用户的权限缓存（用户角色、状态变更或删除时调用）。
func (cache *AuthzCache) InvalidateUser(ctx context.Context, userID int64) error {
	return cache.client.Del(ctx, authzUserKey(userID)).Err()
}

// InvalidateAllUsers 删除所有用户的权限缓存（角色新增、修改、删除时调用）。
func (cache *AuthzCache) InvalidateAllUsers(ctx context.Context) error {
	// 先完整扫描再删除，避免边扫描边删除导致游标跳过部分 key。
	var keys []string
	iter := cache.client.Scan(ctx, 0, authzUserKeyPattern, 500).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	for start := 0; start < len(keys); start += 500 {
		if err := cache.client.Unlink(ctx, keys[start:min(start+500, len(keys))]...).Err(); err != nil {
			return err
		}
	}
	return nil
}
