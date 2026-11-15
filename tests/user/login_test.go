/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 23:55:00 America/Los_Angeles
 *****************************************************************************/

package user_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"gorm.io/gorm"
	"jkylin.com/kylin/v2/internal/module/user"
	"jkylin.com/kylin/v2/internal/router"
)

func verifyLoginFlow(t *testing.T, database *gorm.DB) {
	t.Helper()
	server := newTestServer(t, database)
	call := func(method, path, header, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		recorder := request(server.engine, method, path, header, body)
		assertStatus(t, recorder, want)
		return recorder
	}
	loginBody := func(username, password string) string {
		return server.loginBody(t, username, password)
	}
	login := func(username, password string) string {
		return server.login(t, username, password)
	}
	admin := server.admin(t)

	createdUser := decodeData[user.User](t, call(http.MethodPost, "/users", admin, `{"username":"login-user","password":"login-password"}`, http.StatusOK))
	userPath := "/users/" + strconv.FormatInt(createdUser.ID, 10)

	call(http.MethodPost, "/auth/login", "", `{"username":"login-user","password":"login-password"}`, http.StatusBadRequest)
	call(http.MethodPost, "/auth/login", "", `{"username":"login-user","password":"login-password","captcha_id":"x","captcha_code":"12a4"}`, http.StatusBadRequest)
	call(http.MethodPost, "/auth/login", "", loginBody("login-user", "wrong-password"), http.StatusUnauthorized)
	call(http.MethodPost, "/auth/login", "", loginBody("no-such-user", "login-password"), http.StatusUnauthorized)

	first := login("login-user", "login-password")
	meRecorder := call(http.MethodGet, "/auth/me", first, "", http.StatusOK)
	if me := decodeData[user.User](t, meRecorder); me.ID != createdUser.ID || strings.Contains(meRecorder.Body.String(), `"password":`) {
		t.Fatalf("unexpected me: %s", meRecorder.Body.String())
	}

	// RBAC：没有 system.users / system.roles 的用户访问管理接口返回 403；授权后同一个 token 立即可用。
	call(http.MethodGet, "/users", first, "", http.StatusForbidden)
	call(http.MethodGet, "/roles", first, "", http.StatusForbidden)
	call(http.MethodGet, "/permissions", first, "", http.StatusForbidden)
	call(http.MethodPut, userPath, admin, `{"roles":["auditor"]}`, http.StatusOK)
	call(http.MethodGet, "/users", first, "", http.StatusForbidden)
	call(http.MethodPost, "/roles", admin, `{"code":"user_manager","name":"User Manager","permissions":["system.users"]}`, http.StatusOK)
	call(http.MethodPut, userPath, admin, `{"roles":["auditor","user_manager"]}`, http.StatusOK)
	call(http.MethodGet, "/users", first, "", http.StatusOK)
	call(http.MethodGet, "/roles", first, "", http.StatusForbidden)
	call(http.MethodPut, userPath, admin, `{"roles":[]}`, http.StatusOK)
	call(http.MethodGet, "/users", first, "", http.StatusForbidden)

	// 用户权限缓存在 Redis：授权后加载到用户权限集合；缓存被清空（如 Redis 重启）时自动从数据库重新加载。
	call(http.MethodPut, userPath, admin, `{"roles":["user_manager"]}`, http.StatusOK)
	call(http.MethodGet, "/users", first, "", http.StatusOK)
	userCacheKey := "kylin:authz:user:" + strconv.FormatInt(createdUser.ID, 10)
	if ok, _ := server.redis.SIsMember(userCacheKey, "system.users"); !ok {
		t.Fatal("user permissions should be cached in Redis")
	}
	server.redis.Del(userCacheKey)
	call(http.MethodGet, "/users", first, "", http.StatusOK)
	call(http.MethodGet, userPath, first, "", http.StatusOK)
	call(http.MethodGet, "/permissions", first, "", http.StatusForbidden)
	call(http.MethodPut, userPath, admin, `{"roles":[]}`, http.StatusOK)
	call(http.MethodGet, "/users", first, "", http.StatusForbidden)

	// 默认只允许一个设备在线：第二次登录后第一个 token 失效。
	second := login("login-user", "login-password")
	call(http.MethodGet, "/auth/me", first, "", http.StatusUnauthorized)
	call(http.MethodGet, "/auth/me", second, "", http.StatusOK)

	// 用户自助修改密码：旧密码错误返回 400；成功后所有设备下线，需要用新密码重新登录。
	call(http.MethodPut, "/auth/password", second, `{"old_password":"wrong","new_password":"self-password"}`, http.StatusBadRequest)
	call(http.MethodPut, "/auth/password", second, `{"old_password":"login-password","new_password":"self-password"}`, http.StatusOK)
	call(http.MethodGet, "/auth/me", second, "", http.StatusUnauthorized)
	call(http.MethodPost, "/auth/login", "", loginBody("login-user", "login-password"), http.StatusUnauthorized)
	afterChange := login("login-user", "self-password")

	// 管理员重置密码 → 所有设备下线。
	reset := decodeData[user.ResetPasswordResponse](t, call(http.MethodPost, userPath+"/password/reset", admin, "", http.StatusOK))
	call(http.MethodGet, "/auth/me", afterChange, "", http.StatusUnauthorized)
	call(http.MethodPost, "/auth/login", "", loginBody("login-user", "self-password"), http.StatusUnauthorized)

	third := login("login-user", reset.NewPassword)
	call(http.MethodPost, "/auth/logout", third, "", http.StatusOK)
	call(http.MethodGet, "/auth/me", third, "", http.StatusUnauthorized)
	call(http.MethodPost, "/auth/logout", third, "", http.StatusUnauthorized)

	// 禁用用户：会话下线，且不能再登录。
	fourth := login("login-user", reset.NewPassword)
	call(http.MethodPut, userPath, admin, `{"status":"N"}`, http.StatusOK)
	call(http.MethodGet, "/auth/me", fourth, "", http.StatusUnauthorized)
	call(http.MethodPost, "/auth/login", "", loginBody("login-user", reset.NewPassword), http.StatusForbidden)

	// 删除用户：会话下线，且不能再登录。
	call(http.MethodPut, userPath, admin, `{"status":"Y"}`, http.StatusOK)
	fifth := login("login-user", reset.NewPassword)
	call(http.MethodDelete, userPath, admin, "", http.StatusOK)
	call(http.MethodGet, "/auth/me", fifth, "", http.StatusUnauthorized)
	call(http.MethodPost, "/auth/login", "", loginBody("login-user", reset.NewPassword), http.StatusUnauthorized)

	// 初始管理员已存在时 EnsureAdmin 不会修改其密码。
	call(http.MethodGet, "/users", server.admin(t), "", http.StatusOK)
}

// verifyLocalAuth：RUN_MODE=local/dev 时不带 token 的请求以初始管理员身份访问，带 token 仍按 token 中的用户鉴权。
func verifyLocalAuth(t *testing.T, database *gorm.DB) {
	t.Helper()
	server := newTestServerWith(t, database, router.Dependencies{SkipAuth: true, LocalUsername: testAdminUsername})
	call := func(method, path, header, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		recorder := request(server.engine, method, path, header, body)
		assertStatus(t, recorder, want)
		return recorder
	}
	if me := decodeData[user.User](t, call(http.MethodGet, "/auth/me", "", "", http.StatusOK)); me.Username != testAdminUsername {
		t.Fatalf("local auth should act as admin: %+v", me)
	}
	call(http.MethodGet, "/users", "", "", http.StatusOK)
	call(http.MethodGet, "/permissions", "", "", http.StatusOK)

	call(http.MethodPost, "/users", "", `{"username":"local-plain","password":"plain-password"}`, http.StatusOK)
	plain := server.login(t, "local-plain", "plain-password")
	call(http.MethodGet, "/users", plain, "", http.StatusForbidden)
	call(http.MethodGet, "/users", "Bearer bad", "", http.StatusUnauthorized)

	// 默认用户不存在时返回 401 并提示如何创建。
	missing := newTestServerWith(t, database, router.Dependencies{SkipAuth: true, LocalUsername: "no-such-admin"})
	assertStatus(t, request(missing.engine, http.MethodGet, "/users", "", ""), http.StatusUnauthorized)
}
