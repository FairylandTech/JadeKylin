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

package apipermission

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	authorizationKeyPrefix          = "kylin:authz:"
	authorizationAPIKey             = authorizationKeyPrefix + "api"       // HASH: "METHOD path" -> permission
	authorizationUserKeyPrefix      = authorizationKeyPrefix + "user:"     // SET: 单个用户的权限集合
	authorizationUserKeyPattern     = authorizationUserKeyPrefix + "*"     // 所有用户权限集合
	authorizationSentinel           = ""                                   // 占位：区分 "已缓存但为空" 和 "未缓存"
	authorizationSyncTemporaryKey   = authorizationKeyPrefix + "api:sync:" // 同步时的临时 key 前缀
	authorizationInvalidationBatch  = 500                                  // 批量失效时每批扫描 / 删除的 key 数
	authorizationCheckMaximumRounds = 3                                    // 鉴权最多补两次缓存：接口权限表、用户权限
)

// AuthorizationResult 是一次鉴权缓存查询的结果。
type AuthorizationResult int

const (
	AuthorizationAPIMissing  AuthorizationResult = -1 // 接口权限表未缓存，需要从数据库同步
	AuthorizationUserMissing AuthorizationResult = -2 // 用户权限未缓存，需要从数据库加载
	AuthorizationNotMapped   AuthorizationResult = 0  // 接口未登记放行权限
	AuthorizationDenied      AuthorizationResult = 1
	AuthorizationAllowed     AuthorizationResult = 2
)

// AuthorizationCache 在 Redis 中缓存鉴权数据，登录后的请求鉴权不访问数据库。
//
//	kylin:authz:api          HASH "METHOD path" -> 放行权限；api_permissions 变更后整体重建
//	kylin:authz:user:<uid>   SET  用户通过启用角色拥有的权限；按需加载，空闲 TTL 后过期，用户 / 角色变更时删除
type AuthorizationCache struct {
	client         *redis.Client
	userTimeToLive time.Duration
}

func NewAuthorizationCache(client *redis.Client, userTimeToLive time.Duration) *AuthorizationCache {
	return &AuthorizationCache{client: client, userTimeToLive: userTimeToLive}
}

func authorizationUserKey(userID int64) string {
	return authorizationUserKeyPrefix + strconv.FormatInt(userID, 10)
}

// endpointField 返回接口在权限表 HASH 中的 field："METHOD path"。
func endpointField(method, path string) string {
	return method + " " + path
}

// Check 判断用户能否访问接口，一次 Redis 往返完成；命中时为用户权限续期。
func (authorizationCache *AuthorizationCache) Check(ctx context.Context, userID int64, method, path string) (AuthorizationResult, error) {
	keys := []string{authorizationAPIKey, authorizationUserKey(userID)}
	result, err := checkAuthorizationScript.Run(ctx, authorizationCache.client, keys,
		endpointField(method, path), authorizationCache.userTimeToLive.Milliseconds()).Int()
	return AuthorizationResult(result), err
}

// SyncAPIPermissions 用最新的接口放行权限整体替换缓存：先写临时 key 再 RENAME，读者不会看到半成品。
func (authorizationCache *AuthorizationCache) SyncAPIPermissions(ctx context.Context, records []ApiPermission, syncID string) error {
	temporaryKey := authorizationSyncTemporaryKey + syncID
	values := make([]any, 0, 2+2*len(records))
	values = append(values, authorizationSentinel, authorizationSentinel)
	for _, record := range records {
		values = append(values, endpointField(record.Method, record.Path), record.Permission)
	}
	_, err := authorizationCache.client.TxPipelined(ctx, func(pipeline redis.Pipeliner) error {
		pipeline.Del(ctx, temporaryKey)
		pipeline.HSet(ctx, temporaryKey, values...)
		pipeline.Rename(ctx, temporaryKey, authorizationAPIKey)
		return nil
	})
	return err
}

// SetUserPermissions 缓存用户的权限集合（可以为空）。
func (authorizationCache *AuthorizationCache) SetUserPermissions(ctx context.Context, userID int64, permissions []string) error {
	key := authorizationUserKey(userID)
	members := make([]any, 0, len(permissions)+1)
	members = append(members, authorizationSentinel)
	for _, permission := range permissions {
		members = append(members, permission)
	}
	_, err := authorizationCache.client.TxPipelined(ctx, func(pipeline redis.Pipeliner) error {
		pipeline.Del(ctx, key)
		pipeline.SAdd(ctx, key, members...)
		pipeline.PExpire(ctx, key, authorizationCache.userTimeToLive)
		return nil
	})
	return err
}

// InvalidateUser 删除单个用户的权限缓存（用户角色、状态变更或删除时调用）。
func (authorizationCache *AuthorizationCache) InvalidateUser(ctx context.Context, userID int64) error {
	return authorizationCache.client.Del(ctx, authorizationUserKey(userID)).Err()
}

// InvalidateAllUsers 删除所有用户的权限缓存（角色新增、修改、删除时调用）。
func (authorizationCache *AuthorizationCache) InvalidateAllUsers(ctx context.Context) error {
	// 先完整扫描再删除，避免边扫描边删除导致游标跳过部分 key。
	var keys []string
	iterator := authorizationCache.client.Scan(ctx, 0, authorizationUserKeyPattern, authorizationInvalidationBatch).Iterator()
	for iterator.Next(ctx) {
		keys = append(keys, iterator.Val())
	}
	if err := iterator.Err(); err != nil {
		return err
	}
	for start := 0; start < len(keys); start += authorizationInvalidationBatch {
		batch := keys[start:min(start+authorizationInvalidationBatch, len(keys))]
		if err := authorizationCache.client.Unlink(ctx, batch...).Err(); err != nil {
			return err
		}
	}
	return nil
}
