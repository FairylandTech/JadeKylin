/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-15 00:40:00 America/Los_Angeles
 *****************************************************************************/

package user_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"jkylin.com/kylin/v2/internal/config"
	"jkylin.com/kylin/v2/internal/contracts/requests"
	"jkylin.com/kylin/v2/internal/repository"
	"jkylin.com/kylin/v2/internal/router"
	"jkylin.com/kylin/v2/internal/service"
)

const (
	testJWTSecret     = "0123456789abcdef0123456789abcdef"
	testAdminUsername = "root"
	testAdminPassword = "root-password"
)

type testServer struct {
	engine *gin.Engine
	redis  *miniredis.Miniredis
}

// newTestServer 启动带 miniredis 的完整路由，并确保初始管理员 root 存在。
func newTestServer(t *testing.T, db *gorm.DB) *testServer {
	t.Helper()
	return newTestServerWith(t, db, router.Dependencies{})
}

// newTestServerWith 同 newTestServer，可通过 extra 设置 SkipAuth / LocalUsername。
func newTestServerWith(t *testing.T, db *gorm.DB, extra router.Dependencies) *testServer {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	authCfg := config.Default().Auth
	authCfg.JWTSecret = testJWTSecret
	engine := gin.New()
	router.Init(engine, router.Dependencies{DB: db, Redis: client, Auth: authCfg, SkipAuth: extra.SkipAuth, LocalUsername: extra.LocalUsername})
	if db != nil {
		userService := service.NewUserService(repository.NewUserRepository(db), repository.NewRoleRepository(db))
		if _, err := userService.EnsureAdmin(context.Background(), testAdminUsername, testAdminPassword); err != nil {
			t.Fatal(err)
		}
	}
	return &testServer{engine: engine, redis: mr}
}

// loginBody 获取验证码并拼出登录请求体。
func (server *testServer) loginBody(t *testing.T, username, password string) string {
	t.Helper()
	res := request(server.engine, http.MethodGet, "/auth/captcha", "", "")
	assertStatus(t, res, http.StatusOK)
	captcha := decodeData[requests.CaptchaResponse](t, res)
	return `{"username":"` + username + `","password":"` + password + `","captcha_id":"` + captcha.CaptchaID + `","captcha_code":"` + captcha.Code + `"}`
}

// login 登录并返回 "Bearer <jwt>"。
func (server *testServer) login(t *testing.T, username, password string) string {
	t.Helper()
	res := request(server.engine, http.MethodPost, "/auth/login", "", server.loginBody(t, username, password))
	assertStatus(t, res, http.StatusOK)
	reply := decodeData[requests.LoginResponse](t, res)
	if reply.AccessToken == "" || reply.TokenType != "Bearer" || reply.ExpiresIn != 900 || reply.User == nil || reply.User.Username != username {
		t.Fatalf("unexpected login response: %+v", reply)
	}
	return "Bearer " + reply.AccessToken
}

func (server *testServer) admin(t *testing.T) string {
	t.Helper()
	return server.login(t, testAdminUsername, testAdminPassword)
}
