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

package authorization_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"jkylin.com/kylin/v2/internal/pkg/authorization"
	"jkylin.com/kylin/v2/internal/pkg/identity"
	"jkylin.com/kylin/v2/pkg/errcode"
)

// fakeLoader 记录数据库加载次数，模拟 user.Repository.PermissionCodes。
type fakeLoader struct {
	permissions map[int64][]string
	err         error
	calls       int
}

func (loader *fakeLoader) PermissionCodes(_ context.Context, userID int64) ([]string, error) {
	loader.calls++
	return loader.permissions[userID], loader.err
}

func newCache(t *testing.T) (*miniredis.Miniredis, *authorization.Cache) {
	t.Helper()
	miniRedis := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})
	return miniRedis, authorization.NewCache(client, 10*time.Minute)
}

// contextFor 返回携带登录用户的 request context。
func contextFor(userID int64) context.Context {
	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginContext.Request = httptest.NewRequest("GET", "/", nil)
	identity.Attach(ginContext, &identity.Principal{UserID: userID, Username: "user", SessionID: "session"})
	return ginContext.Request.Context()
}

func assertCode(t *testing.T, err error, want *errcode.Error) {
	t.Helper()
	if want == nil {
		if err != nil {
			t.Fatalf("want nil error, got %v", err)
		}
		return
	}
	if got := errcode.From(err); got == nil || got.Code != want.Code {
		t.Fatalf("want code %d, got %v", want.Code, err)
	}
}

func TestCache(t *testing.T) {
	miniRedis, cache := newCache(t)
	ctx := context.Background()
	assertResult := func(userID int64, permission string, want authorization.CheckResult) {
		t.Helper()
		if got, err := cache.Check(ctx, userID, permission); err != nil || got != want {
			t.Fatalf("user %d %s: got %d, %v; want %d", userID, permission, got, err, want)
		}
	}

	assertResult(1, "system.users", authorization.CheckUserMissing)
	// 空集合也算已缓存：拒绝，而不是缓存缺失。
	if err := cache.SetUserPermissions(ctx, 1, nil); err != nil {
		t.Fatal(err)
	}
	assertResult(1, "system.users", authorization.CheckDenied)
	assertResult(1, "", authorization.CheckDenied) // 占位成员不能当作权限

	if err := cache.SetUserPermissions(ctx, 2, []string{"system.users"}); err != nil {
		t.Fatal(err)
	}
	assertResult(2, "system.users", authorization.CheckAllowed)
	assertResult(2, "system.roles", authorization.CheckDenied)

	// 命中时续期：快进到接近 TTL 后再访问，缓存不应过期。
	miniRedis.FastForward(9 * time.Minute)
	assertResult(2, "system.users", authorization.CheckAllowed)
	miniRedis.FastForward(9 * time.Minute)
	assertResult(2, "system.users", authorization.CheckAllowed)
	miniRedis.FastForward(11 * time.Minute)
	assertResult(2, "system.users", authorization.CheckUserMissing)

	if err := cache.SetUserPermissions(ctx, 2, []string{"system.users"}); err != nil {
		t.Fatal(err)
	}
	if err := cache.InvalidateUser(ctx, 2); err != nil {
		t.Fatal(err)
	}
	assertResult(2, "system.users", authorization.CheckUserMissing)

	// InvalidateAllUsers 跨多个批次删除全部用户缓存。
	for id := int64(1); id <= 1200; id++ {
		if err := cache.SetUserPermissions(ctx, id, []string{"system.users"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := cache.InvalidateAllUsers(ctx); err != nil {
		t.Fatal(err)
	}
	if keys := miniRedis.Keys(); len(keys) != 0 {
		t.Fatalf("user caches should be invalidated, %d left", len(keys))
	}
	assertResult(1000, "system.users", authorization.CheckUserMissing)
}

func TestAuthorizerRequire(t *testing.T) {
	miniRedis, cache := newCache(t)
	loader := &fakeLoader{permissions: map[int64][]string{1: {"system.users"}, 2: {"logs.view"}}}
	authorizer := authorization.NewAuthorizer(cache, loader)

	// 未登录。
	assertCode(t, authorizer.Require(context.Background(), "system.users"), errcode.Unauthorized)

	// 首次访问从数据库加载一次，之后只读 Redis。
	assertCode(t, authorizer.Require(contextFor(1), "system.users"), nil)
	assertCode(t, authorizer.Require(contextFor(1), "system.users"), nil)
	assertCode(t, authorizer.Require(contextFor(1), "system.roles"), errcode.Forbidden)
	assertCode(t, authorizer.Require(contextFor(2), "system.users"), errcode.Forbidden)
	if loader.calls != 2 {
		t.Fatalf("permissions should be loaded once per user, got %d loads", loader.calls)
	}
	if ok, _ := miniRedis.SIsMember("kylin:authz:user:"+strconv.Itoa(1), "system.users"); !ok {
		t.Fatal("user permissions should be cached in Redis")
	}

	// 失效后重新加载最新权限。
	loader.permissions[2] = []string{"system.users"}
	assertCode(t, authorizer.InvalidateUser(context.Background(), 2), nil)
	assertCode(t, authorizer.Require(contextFor(2), "system.users"), nil)

	// 数据库出错返回 500。
	loader.err = errors.New("database down")
	assertCode(t, authorizer.InvalidateAllUsers(context.Background()), nil)
	assertCode(t, authorizer.Require(contextFor(1), "system.users"), errcode.Internal)

	// nil Authorizer（未启用登录）：判断一律失败，失效直接跳过。
	var disabled *authorization.Authorizer
	assertCode(t, disabled.Require(contextFor(1), "system.users"), errcode.Internal)
	assertCode(t, disabled.InvalidateUser(context.Background(), 1), nil)
	assertCode(t, disabled.InvalidateAllUsers(context.Background()), nil)
}
