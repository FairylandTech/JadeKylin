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

package auth

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const keyPrefix = "kylin:auth:"

// 会话 Hash 的 field。
const (
	sessionFieldUserID    = "user_id"
	sessionFieldUsername  = "username"
	sessionFieldToken     = "token"
	sessionFieldClientIP  = "client_ip"
	sessionFieldUserAgent = "user_agent"
	sessionFieldLoginAt   = "login_at"
)

// Store 用 Redis 保存验证码和登录会话。
//
//	kylin:auth:captcha:<id>              验证码，一次性
//	kylin:auth:{<uid>}:session:<sid>     会话 Hash，TTL = idle_timeout，每次请求续期
//	kylin:auth:{<uid>}:sessions          ZSET(sid -> 登录时间)，用于限制在线设备数
//
// 同一用户的 key 共享 {<uid>} hash tag，Lua 脚本在 Redis Cluster 下也落在同一个 slot。
type Store struct {
	client *redis.Client
}

func NewStore(client *redis.Client) *Store {
	return &Store{client: client}
}

func captchaKey(captchaID string) string {
	return keyPrefix + "captcha:" + captchaID
}

func userTag(userID int64) string {
	return keyPrefix + "{" + strconv.FormatInt(userID, 10) + "}:"
}

func sessionKeyPrefix(userID int64) string {
	return userTag(userID) + "session:"
}

func sessionKey(userID int64, sessionID string) string {
	return sessionKeyPrefix(userID) + sessionID
}

func sessionIndexKey(userID int64) string {
	return userTag(userID) + "sessions"
}

func (store *Store) SaveCaptcha(ctx context.Context, captchaID, code string, timeToLive time.Duration) error {
	return store.client.Set(ctx, captchaKey(captchaID), code, timeToLive).Err()
}

// TakeCaptcha 读取并删除验证码（无论校验成功与否都只能用一次）。不存在时返回 "" 和 nil。
func (store *Store) TakeCaptcha(ctx context.Context, captchaID string) (string, error) {
	code, err := store.client.GetDel(ctx, captchaKey(captchaID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return code, err
}

// CreateSession 保存新会话；maxSessions > 0 时同一用户最多保留 maxSessions 个会话，返回被踢下线的 session id。
func (store *Store) CreateSession(ctx context.Context, session *Session, idleTimeout time.Duration, maxSessions int) ([]string, error) {
	keys := []string{sessionIndexKey(session.UserID), sessionKey(session.UserID, session.SessionID)}
	arguments := []any{
		sessionKeyPrefix(session.UserID), session.SessionID, session.LoginAt.UnixMilli(), idleTimeout.Milliseconds(), maxSessions,
		sessionFieldUserID, session.UserID,
		sessionFieldUsername, session.Username,
		sessionFieldToken, session.Token,
		sessionFieldClientIP, session.ClientIP,
		sessionFieldUserAgent, session.UserAgent,
		sessionFieldLoginAt, session.LoginAt.UnixMilli(),
	}
	return createSessionScript.Run(ctx, store.client, keys, arguments...).StringSlice()
}

// TouchSession 刷新会话的空闲过期时间；renewedToken 非空时一并替换保存的 JWT。会话不存在返回 false。
func (store *Store) TouchSession(ctx context.Context, userID int64, sessionID string, idleTimeout time.Duration, renewedToken string) (bool, error) {
	keys := []string{sessionKey(userID, sessionID), sessionIndexKey(userID)}
	alive, err := touchSessionScript.Run(ctx, store.client, keys, idleTimeout.Milliseconds(), renewedToken).Int()
	return alive == 1, err
}

// GetSession 读取会话；不存在时返回 nil, nil。
func (store *Store) GetSession(ctx context.Context, userID int64, sessionID string) (*Session, error) {
	values, err := store.client.HGetAll(ctx, sessionKey(userID, sessionID)).Result()
	if err != nil || len(values) == 0 {
		return nil, err
	}
	loginAt, _ := strconv.ParseInt(values[sessionFieldLoginAt], 10, 64)
	return &Session{
		SessionID: sessionID,
		UserID:    userID,
		Username:  values[sessionFieldUsername],
		Token:     values[sessionFieldToken],
		ClientIP:  values[sessionFieldClientIP],
		UserAgent: values[sessionFieldUserAgent],
		LoginAt:   time.UnixMilli(loginAt),
	}, nil
}

// DeleteSession 注销单个会话。
func (store *Store) DeleteSession(ctx context.Context, userID int64, sessionID string) error {
	_, err := store.client.TxPipelined(ctx, func(pipeline redis.Pipeliner) error {
		pipeline.Del(ctx, sessionKey(userID, sessionID))
		pipeline.ZRem(ctx, sessionIndexKey(userID), sessionID)
		return nil
	})
	return err
}

// DeleteUserSessions 让用户的所有会话下线（修改 / 重置密码、禁用或删除用户时调用）。
func (store *Store) DeleteUserSessions(ctx context.Context, userID int64) error {
	return revokeSessionsScript.Run(ctx, store.client, []string{sessionIndexKey(userID)}, sessionKeyPrefix(userID)).Err()
}
