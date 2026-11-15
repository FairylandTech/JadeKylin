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
	"jkylin.com/kylin/v2/internal/contracts/requests"
	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/internal/router"
)

func verifyLoginFlow(t *testing.T, db *gorm.DB) {
	t.Helper()
	server := newTestServer(t, db)
	call := func(method, path, header, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		res := request(server.engine, method, path, header, body)
		assertStatus(t, res, want)
		return res
	}
	loginBody := func(username, password string) string { return server.loginBody(t, username, password) }
	login := func(username, password string) string { return server.login(t, username, password) }
	admin := server.admin(t)

	user := decodeData[model.User](t, call(http.MethodPost, "/users", admin, `{"username":"login-user","password":"login-password"}`, http.StatusOK))
	userPath := "/users/" + strconv.FormatInt(user.ID, 10)

	call(http.MethodPost, "/auth/login", "", `{"username":"login-user","password":"login-password"}`, http.StatusBadRequest)
	call(http.MethodPost, "/auth/login", "", `{"username":"login-user","password":"login-password","captcha_id":"x","captcha_code":"12a4"}`, http.StatusBadRequest)
	call(http.MethodPost, "/auth/login", "", loginBody("login-user", "wrong-password"), http.StatusUnauthorized)
	call(http.MethodPost, "/auth/login", "", loginBody("no-such-user", "login-password"), http.StatusUnauthorized)

	first := login("login-user", "login-password")
	meRes := call(http.MethodGet, "/auth/me", first, "", http.StatusOK)
	if me := decodeData[model.User](t, meRes); me.ID != user.ID || strings.Contains(meRes.Body.String(), `"password":`) {
		t.Fatalf("unexpected me: %s", meRes.Body.String())
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

	// 接口放行权限可直接修改且立即生效：把 GET /users 改为 logs.view 后，拥有 auditor（含 logs.view）的用户可以访问。
	apiPermissions := decodeData[[]model.ApiPermission](t, call(http.MethodGet, "/api-permissions", admin, "", http.StatusOK))
	var listUsers *model.ApiPermission
	for i := range apiPermissions {
		if apiPermissions[i].Method == http.MethodGet && apiPermissions[i].Path == "/users" {
			listUsers = &apiPermissions[i]
		}
	}
	if listUsers == nil || listUsers.Permission != "system.users" || len(apiPermissions) < 15 {
		t.Fatalf("unexpected api permissions: %+v", apiPermissions)
	}
	call(http.MethodGet, "/api-permissions", first, "", http.StatusForbidden)
	listUsersPath := "/api-permissions/" + strconv.FormatInt(listUsers.ID, 10)
	call(http.MethodPut, listUsersPath, admin, `{"permission":"no.such.permission"}`, http.StatusBadRequest)
	call(http.MethodPut, listUsersPath, admin, `{}`, http.StatusBadRequest)
	call(http.MethodPut, "/api-permissions/999999", admin, `{"permission":"logs.view"}`, http.StatusNotFound)
	call(http.MethodPut, userPath, admin, `{"roles":["auditor"]}`, http.StatusOK)
	changed := decodeData[model.ApiPermission](t, call(http.MethodPut, listUsersPath, admin, `{"permission":"logs.view"}`, http.StatusOK))
	if changed.Permission != "logs.view" || changed.Path != "/users" {
		t.Fatalf("unexpected changed api permission: %+v", changed)
	}
	call(http.MethodGet, "/users", first, "", http.StatusOK)
	call(http.MethodGet, userPath, first, "", http.StatusForbidden) // 其它接口不受影响
	// 鉴权数据已缓存到 Redis：接口表同步了最新权限，用户权限集合已加载。
	if permission := server.redis.HGet("kylin:authz:api", "GET /users"); permission != "logs.view" {
		t.Fatalf("api permission cache not synced: %q", permission)
	}
	if ok, _ := server.redis.SIsMember("kylin:authz:user:"+strconv.FormatInt(user.ID, 10), "logs.view"); !ok {
		t.Fatal("user permissions should be cached in Redis")
	}
	// Redis 中的接口表被清空（如 Redis 重启）时自动从数据库重新同步。
	server.redis.Del("kylin:authz:api")
	call(http.MethodGet, "/users", first, "", http.StatusOK)
	call(http.MethodPut, listUsersPath, admin, `{"permission":"system.users"}`, http.StatusOK)
	call(http.MethodGet, "/users", first, "", http.StatusForbidden)
	call(http.MethodPut, userPath, admin, `{"roles":[]}`, http.StatusOK)

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
	reset := decodeData[requests.ResetPasswordResponse](t, call(http.MethodPost, userPath+"/password/reset", admin, "", http.StatusOK))
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
func verifyLocalAuth(t *testing.T, db *gorm.DB) {
	t.Helper()
	server := newTestServerWith(t, db, router.Dependencies{SkipAuth: true, LocalUsername: testAdminUsername})
	call := func(method, path, header, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		res := request(server.engine, method, path, header, body)
		assertStatus(t, res, want)
		return res
	}
	if me := decodeData[model.User](t, call(http.MethodGet, "/auth/me", "", "", http.StatusOK)); me.Username != testAdminUsername {
		t.Fatalf("local auth should act as admin: %+v", me)
	}
	call(http.MethodGet, "/users", "", "", http.StatusOK)
	call(http.MethodGet, "/api-permissions", "", "", http.StatusOK)

	call(http.MethodPost, "/users", "", `{"username":"local-plain","password":"plain-password"}`, http.StatusOK)
	plain := server.login(t, "local-plain", "plain-password")
	call(http.MethodGet, "/users", plain, "", http.StatusForbidden)
	call(http.MethodGet, "/users", "Bearer bad", "", http.StatusUnauthorized)

	// 默认用户不存在时返回 401 并提示如何创建。
	missing := newTestServerWith(t, db, router.Dependencies{SkipAuth: true, LocalUsername: "no-such-admin"})
	assertStatus(t, request(missing.engine, http.MethodGet, "/users", "", ""), http.StatusUnauthorized)
}
