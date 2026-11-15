/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 23:50:00 America/Los_Angeles
 *****************************************************************************/

package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"jkylin.com/kylin/v2/internal/config"
	"jkylin.com/kylin/v2/internal/middleware/security"
	"jkylin.com/kylin/v2/internal/module/auth"
	"jkylin.com/kylin/v2/internal/pkg/identity"
	"jkylin.com/kylin/v2/pkg/consts"
	"jkylin.com/kylin/v2/pkg/errcode"
)

const secret = "0123456789abcdef0123456789abcdef"

func setup(t *testing.T) (*miniredis.Miniredis, *auth.Store, *auth.Service, config.Auth) {
	t.Helper()
	miniRedis := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})
	configuration := config.Default().Auth
	configuration.JWTSecret = secret
	store := auth.NewStore(client)
	return miniRedis, store, auth.NewService(nil, store, configuration), configuration
}

func sign(t *testing.T, key string, userID int64, sessionID string, issuedAt time.Time, timeToLive time.Duration) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "jade-kylin", "sub": strconv.FormatInt(userID, 10), "user_id": userID, "jti": sessionID, "username": "alice",
		"iat": issuedAt.Unix(), "exp": issuedAt.Add(timeToLive).Unix(),
	}).SignedString([]byte(key))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func createSession(t *testing.T, store *auth.Store, userID int64, sessionID string, loginAt time.Time, maxSessions int) []string {
	t.Helper()
	evicted, err := store.CreateSession(context.Background(), &auth.Session{
		SessionID: sessionID, UserID: userID, Username: "alice", Token: "token-" + sessionID, LoginAt: loginAt,
	}, 10*time.Minute, maxSessions)
	if err != nil {
		t.Fatal(err)
	}
	return evicted
}

func assertCode(t *testing.T, err error, want *errcode.Error) {
	t.Helper()
	got := errcode.From(err)
	if got == nil || got.Code != want.Code {
		t.Fatalf("error=%v want code %d", err, want.Code)
	}
}

func TestSessionLimit(t *testing.T) {
	miniRedis, store, _, _ := setup(t)
	ctx := context.Background()
	now := time.Now()

	// 默认 max_sessions=1：新设备登录会把旧设备踢下线。
	createSession(t, store, 1, "phone", now, 1)
	if evicted := createSession(t, store, 1, "laptop", now.Add(time.Second), 1); len(evicted) != 1 || evicted[0] != "phone" {
		t.Fatalf("expected phone to be evicted: %v", evicted)
	}
	if session, err := store.GetSession(ctx, 1, "phone"); err != nil || session != nil {
		t.Fatalf("evicted session still present: %+v, %v", session, err)
	}
	if session, err := store.GetSession(ctx, 1, "laptop"); err != nil || session == nil || session.Token != "token-laptop" {
		t.Fatalf("new session missing: %+v, %v", session, err)
	}

	// max_sessions=2 保留最近的两个；0 不限制。
	createSession(t, store, 2, "a", now, 2)
	createSession(t, store, 2, "b", now.Add(time.Second), 2)
	if evicted := createSession(t, store, 2, "c", now.Add(2*time.Second), 2); len(evicted) != 1 || evicted[0] != "a" {
		t.Fatalf("expected oldest session a to be evicted: %v", evicted)
	}
	for _, sessionID := range []string{"x", "y", "z"} {
		if evicted := createSession(t, store, 3, sessionID, now, 0); len(evicted) != 0 {
			t.Fatalf("unlimited sessions should not evict: %v", evicted)
		}
	}
	if members, _ := miniRedis.ZMembers("kylin:auth:{3}:sessions"); len(members) != 3 {
		t.Fatalf("expected 3 sessions: %v", members)
	}

	// 已过期的会话不占用名额。
	createSession(t, store, 4, "old", now, 1)
	miniRedis.FastForward(11 * time.Minute)
	if evicted := createSession(t, store, 4, "new", now, 1); len(evicted) != 0 {
		t.Fatalf("expired session should be pruned, not evicted: %v", evicted)
	}

	if err := store.DeleteUserSessions(ctx, 2); err != nil {
		t.Fatal(err)
	}
	for _, sessionID := range []string{"b", "c"} {
		if session, _ := store.GetSession(ctx, 2, sessionID); session != nil {
			t.Fatalf("session %s should be revoked", sessionID)
		}
	}
	if miniRedis.Exists("kylin:auth:{2}:sessions") {
		t.Fatal("session index should be deleted")
	}
}

func TestAuthenticateIdleTimeoutAndRenewal(t *testing.T) {
	miniRedis, store, authService, _ := setup(t)
	ctx := context.Background()
	start := time.Now().Truncate(time.Second)
	clock := start
	authService.SetClock(func() time.Time {
		return clock
	})
	createSession(t, store, 7, "sid", start, 1)
	token := sign(t, secret, 7, "sid", start, 15*time.Minute)

	// 9 分钟后有请求：JWT 剩余 6 分钟 < 10 分钟空闲窗口 → 续签；会话 TTL 重置为 10 分钟。
	miniRedis.FastForward(9 * time.Minute)
	clock = start.Add(9 * time.Minute)
	principal, renewed, err := authService.Authenticate(ctx, token)
	if err != nil || principal.UserID != 7 || principal.SessionID != "sid" || renewed == "" {
		t.Fatalf("authenticate: %+v, %q, %v", principal, renewed, err)
	}
	if timeToLive := miniRedis.TTL("kylin:auth:{7}:session:sid"); timeToLive != 10*time.Minute {
		t.Fatalf("session idle TTL should be refreshed: %v", timeToLive)
	}
	if stored, _ := store.GetSession(ctx, 7, "sid"); stored.Token != renewed {
		t.Fatal("renewed JWT should be stored in Redis")
	}
	var claims jwt.RegisteredClaims
	if _, err := jwt.ParseWithClaims(renewed, &claims, func(*jwt.Token) (any, error) {
		return []byte(secret), nil
	}); err != nil ||
		claims.ID != "sid" || !claims.ExpiresAt.Equal(clock.Add(15*time.Minute)) {
		t.Fatalf("renewed token: %+v, %v", claims, err)
	}

	// 刚续签的 token：剩余 15 分钟 ≥ 10 分钟，不再续签。
	miniRedis.FastForward(time.Minute)
	clock = clock.Add(time.Minute)
	if _, again, err := authService.Authenticate(ctx, renewed); err != nil || again != "" {
		t.Fatalf("should not renew fresh token: %q, %v", again, err)
	}

	// 原 token 在 15 分钟后过期。
	clock = start.Add(15 * time.Minute)
	_, _, err = authService.Authenticate(ctx, token)
	assertCode(t, err, errcode.Unauthorized)

	// 10 分钟没有任何请求 → 会话过期，即便 JWT 本身尚未过期。
	miniRedis.FastForward(10 * time.Minute)
	clock = start.Add(20 * time.Minute)
	_, _, err = authService.Authenticate(ctx, renewed)
	if err == nil || !strings.Contains(err.Error(), "session") {
		t.Fatalf("expected session expired: %v", err)
	}
}

func TestAuthenticateRejectsBadTokens(t *testing.T) {
	_, store, authService, _ := setup(t)
	ctx := context.Background()
	now := time.Now()
	createSession(t, store, 1, "sid", now, 1)
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"iss": "jade-kylin", "sub": "1", "user_id": 1, "jti": "sid", "exp": now.Add(time.Hour).Unix(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	mismatch, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "jade-kylin", "sub": "2", "user_id": 1, "jti": "sid", "exp": now.Add(time.Hour).Unix(),
	}).SignedString([]byte(secret))
	missingUserID, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "jade-kylin", "sub": "1", "jti": "sid", "exp": now.Add(time.Hour).Unix(),
	}).SignedString([]byte(secret))
	for name, token := range map[string]string{
		"sub mismatch":    mismatch,
		"missing user_id": missingUserID,
		"garbage":         "not-a-jwt",
		"wrong secret":    sign(t, "ffffffffffffffffffffffffffffffff", 1, "sid", now, 15*time.Minute),
		"alg none":        none,
		"unknown sid":     sign(t, secret, 1, "other", now, 15*time.Minute),
		"wrong subject":   sign(t, secret, 2, "sid", now, 15*time.Minute),
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := authService.Authenticate(ctx, token)
			assertCode(t, err, errcode.Unauthorized)
		})
	}
}

func TestJWTMiddleware(t *testing.T) {
	_, store, authService, _ := setup(t)
	start := time.Now().Truncate(time.Second)
	authService.SetClock(func() time.Time {
		return start.Add(6 * time.Minute)
	})
	createSession(t, store, 9, "sid", start, 1)
	token := sign(t, secret, 9, "sid", start, 15*time.Minute)

	engine := gin.New()
	engine.GET("/private", security.Authentication(authService), func(ctx *gin.Context) {
		userID, ok := identity.UserID(ctx)
		if !ok || userID != 9 || ctx.GetInt64(consts.UserIDKey) != 9 {
			t.Errorf("user id not stored in context: %d %v", userID, ok)
		}
		ctx.String(http.StatusOK, identity.Current(ctx).Username)
	})
	call := func(header string) *httptest.ResponseRecorder {
		httpRequest := httptest.NewRequest(http.MethodGet, "/private", nil)
		httpRequest.Header.Set("Authorization", header)
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httpRequest)
		return recorder
	}
	for _, header := range []string{"", "Bearer ", "Basic " + token, "Bearer bad"} {
		if recorder := call(header); recorder.Code != http.StatusUnauthorized || recorder.Header().Get("WWW-Authenticate") == "" {
			t.Fatalf("header %q: status=%d", header, recorder.Code)
		}
	}
	recorder := call("Bearer " + token)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "alice" || recorder.Header().Get(consts.AccessTokenHeader) == "" {
		t.Fatalf("authorized request: %d %s %v", recorder.Code, recorder.Body.String(), recorder.Header())
	}
	if err := authService.Logout(context.Background(), &identity.Principal{UserID: 9, SessionID: "sid"}); err != nil {
		t.Fatal(err)
	}
	if recorder := call("Bearer " + token); recorder.Code != http.StatusUnauthorized {
		t.Fatalf("token should be rejected after logout: %d", recorder.Code)
	}
}

func TestCaptcha(t *testing.T) {
	miniRedis, store, authService, configuration := setup(t)
	ctx := context.Background()
	result, err := authService.Captcha(ctx)
	if err != nil || result.CaptchaID == "" || result.ExpiresIn != configuration.CaptchaTTL {
		t.Fatalf("captcha: %+v, %v", result, err)
	}
	key := "kylin:auth:captcha:" + result.CaptchaID
	code, err := miniRedis.Get(key)
	if err != nil || code != result.Code || len(code) != 4 || strings.Trim(code, "0123456789") != "" {
		t.Fatalf("stored captcha code %q: %v", code, err)
	}
	if timeToLive := miniRedis.TTL(key); timeToLive != time.Duration(configuration.CaptchaTTL)*time.Second {
		t.Fatalf("captcha TTL: %v", timeToLive)
	}

	// 验证码错误：返回 400，且验证码被消费，不能用正确的码重试。
	wrong := "0000"
	if code == wrong {
		wrong = "1111"
	}
	login := func(captchaCode string) error {
		_, err := authService.Login(ctx, &auth.LoginRequest{Username: "alice", Password: "pw", CaptchaID: result.CaptchaID, CaptchaCode: captchaCode}, "", "")
		return err
	}
	assertCode(t, login(wrong), errcode.InvalidParameters)
	assertCode(t, login(code), errcode.InvalidParameters)

	// 过期验证码同样无效。
	other, _ := authService.Captcha(ctx)
	miniRedis.FastForward(time.Duration(configuration.CaptchaTTL+1) * time.Second)
	if got, err := store.TakeCaptcha(ctx, other.CaptchaID); err != nil || got != "" {
		t.Fatalf("expired captcha should be gone: %q, %v", got, err)
	}
}

func TestLocalAuthentication(t *testing.T) {
	_, store, authService, _ := setup(t)
	now := time.Now()
	createSession(t, store, 5, "sid", now, 1)
	token := "Bearer " + sign(t, secret, 5, "sid", now, 15*time.Minute)

	var resolveErr error
	resolved := 0
	resolve := func(context.Context) (*identity.Principal, error) {
		resolved++
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &identity.Principal{UserID: 1, Username: "admin", SessionID: auth.LocalSessionID}, nil
	}
	engine := gin.New()
	engine.GET("/private", security.LocalAuthentication(authService, resolve), func(ctx *gin.Context) {
		userID, _ := identity.UserID(ctx)
		ctx.String(http.StatusOK, strconv.FormatInt(userID, 10)+":"+strconv.FormatInt(ctx.GetInt64(consts.UserIDKey), 10))
	})
	call := func(header string) *httptest.ResponseRecorder {
		httpRequest := httptest.NewRequest(http.MethodGet, "/private", nil)
		if header != "" {
			httpRequest.Header.Set("Authorization", header)
		}
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httpRequest)
		return recorder
	}

	// 不带 Authorization：跳过认证，以默认用户（admin）身份访问。
	if recorder := call(""); recorder.Code != http.StatusOK || recorder.Body.String() != "1:1" {
		t.Fatalf("local auth: %d %s", recorder.Code, recorder.Body.String())
	}
	// 带 token：仍按正常 JWT 认证，使用 token 中的用户。
	if recorder := call(token); recorder.Code != http.StatusOK || recorder.Body.String() != "5:5" || resolved != 1 {
		t.Fatalf("jwt auth in local mode: %d %s resolved=%d", recorder.Code, recorder.Body.String(), resolved)
	}
	// 带无效 token 不会回退为管理员。
	if recorder := call("Bearer bad"); recorder.Code != http.StatusUnauthorized {
		t.Fatalf("invalid token should be rejected: %d", recorder.Code)
	}
	resolveErr = errcode.Unauthorized.WithMessage("local auth user not found")
	if recorder := call(""); recorder.Code != http.StatusUnauthorized {
		t.Fatalf("resolver error: %d", recorder.Code)
	}
}
