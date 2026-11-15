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

package repository

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const authKeyPrefix = "kylin:auth:"

// AuthSession 是 Redis 中保存的登录会话。
type AuthSession struct {
	SessionID string
	UserID    int64
	Username  string
	Token     string // 当前有效的 JWT（续签后更新）
	ClientIP  string
	UserAgent string
	LoginAt   time.Time
}

// AuthStore 用 Redis 保存验证码和登录会话。
//
//	kylin:auth:captcha:<id>              验证码，一次性
//	kylin:auth:{<uid>}:session:<sid>     会话 Hash，TTL = idle_timeout，每次请求续期
//	kylin:auth:{<uid>}:sessions          ZSET(sid -> 登录时间)，用于限制在线设备数
//
// 同一用户的 key 共享 {<uid>} hash tag，Lua 脚本在 Redis Cluster 下也落在同一个 slot。
type AuthStore struct {
	client *redis.Client
}

func NewAuthStore(client *redis.Client) *AuthStore {
	return &AuthStore{client: client}
}

func captchaKey(id string) string { return authKeyPrefix + "captcha:" + id }

func userTag(userID int64) string { return authKeyPrefix + "{" + strconv.FormatInt(userID, 10) + "}:" }

func sessionKey(userID int64, sessionID string) string {
	return userTag(userID) + "session:" + sessionID
}

func sessionsKey(userID int64) string { return userTag(userID) + "sessions" }

func (store *AuthStore) SaveCaptcha(ctx context.Context, id, code string, ttl time.Duration) error {
	return store.client.Set(ctx, captchaKey(id), code, ttl).Err()
}

// TakeCaptcha 读取并删除验证码（无论校验成功与否都只能用一次）。不存在时返回 "" 和 nil。
func (store *AuthStore) TakeCaptcha(ctx context.Context, id string) (string, error) {
	code, err := store.client.GetDel(ctx, captchaKey(id)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return code, err
}

// createSessionScript 清理已过期的会话，按 max 踢掉最早登录的会话，然后写入新会话。返回被踢掉的 sid。
var createSessionScript = redis.NewScript(`
local zkey, skey, prefix, sid = KEYS[1], KEYS[2], ARGV[1], ARGV[2]
local now, idle, max = tonumber(ARGV[3]), tonumber(ARGV[4]), tonumber(ARGV[5])
for _, member in ipairs(redis.call('ZRANGE', zkey, 0, -1)) do
  if redis.call('EXISTS', prefix .. member) == 0 then redis.call('ZREM', zkey, member) end
end
local evicted = {}
if max > 0 then
  local count = redis.call('ZCARD', zkey)
  if count >= max then
    for _, member in ipairs(redis.call('ZRANGE', zkey, 0, count - max)) do
      redis.call('DEL', prefix .. member)
      redis.call('ZREM', zkey, member)
      table.insert(evicted, member)
    end
  end
end
redis.call('HSET', skey, unpack(ARGV, 6))
redis.call('PEXPIRE', skey, idle)
redis.call('ZADD', zkey, now, sid)
redis.call('PEXPIRE', zkey, idle)
return evicted
`)

// CreateSession 保存新会话；maxSessions > 0 时同一用户最多保留 maxSessions 个会话，返回被踢下线的 sid。
func (store *AuthStore) CreateSession(ctx context.Context, session *AuthSession, idle time.Duration, maxSessions int) ([]string, error) {
	keys := []string{sessionsKey(session.UserID), sessionKey(session.UserID, session.SessionID)}
	args := []any{
		userTag(session.UserID) + "session:", session.SessionID, session.LoginAt.UnixMilli(), idle.Milliseconds(), maxSessions,
		"user_id", session.UserID, "username", session.Username, "token", session.Token,
		"client_ip", session.ClientIP, "user_agent", session.UserAgent, "login_at", session.LoginAt.UnixMilli(),
	}
	return createSessionScript.Run(ctx, store.client, keys, args...).StringSlice()
}

// touchSessionScript 会话存在时续期（可选更新 token），返回 1；不存在返回 0。
var touchSessionScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then return 0 end
if ARGV[2] ~= '' then redis.call('HSET', KEYS[1], 'token', ARGV[2]) end
redis.call('PEXPIRE', KEYS[1], ARGV[1])
redis.call('PEXPIRE', KEYS[2], ARGV[1])
return 1
`)

// TouchSession 刷新会话的空闲过期时间；renewedToken 非空时一并替换保存的 JWT。会话不存在返回 false。
func (store *AuthStore) TouchSession(ctx context.Context, userID int64, sessionID string, idle time.Duration, renewedToken string) (bool, error) {
	keys := []string{sessionKey(userID, sessionID), sessionsKey(userID)}
	alive, err := touchSessionScript.Run(ctx, store.client, keys, idle.Milliseconds(), renewedToken).Int()
	return alive == 1, err
}

// GetSession 读取会话；不存在时返回 nil, nil。
func (store *AuthStore) GetSession(ctx context.Context, userID int64, sessionID string) (*AuthSession, error) {
	values, err := store.client.HGetAll(ctx, sessionKey(userID, sessionID)).Result()
	if err != nil || len(values) == 0 {
		return nil, err
	}
	loginAt, _ := strconv.ParseInt(values["login_at"], 10, 64)
	return &AuthSession{
		SessionID: sessionID,
		UserID:    userID,
		Username:  values["username"],
		Token:     values["token"],
		ClientIP:  values["client_ip"],
		UserAgent: values["user_agent"],
		LoginAt:   time.UnixMilli(loginAt),
	}, nil
}

// DeleteSession 注销单个会话。
func (store *AuthStore) DeleteSession(ctx context.Context, userID int64, sessionID string) error {
	_, err := store.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, sessionKey(userID, sessionID))
		pipe.ZRem(ctx, sessionsKey(userID), sessionID)
		return nil
	})
	return err
}

var deleteUserSessionsScript = redis.NewScript(`
for _, member in ipairs(redis.call('ZRANGE', KEYS[1], 0, -1)) do redis.call('DEL', ARGV[1] .. member) end
return redis.call('DEL', KEYS[1])
`)

// DeleteUserSessions 让用户的所有会话下线（修改 / 重置密码、禁用或删除用户时调用）。
func (store *AuthStore) DeleteUserSessions(ctx context.Context, userID int64) error {
	return deleteUserSessionsScript.Run(ctx, store.client, []string{sessionsKey(userID)}, userTag(userID)+"session:").Err()
}
