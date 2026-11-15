/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-15 02:50:00 America/Los_Angeles
 *****************************************************************************/

package auth_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/internal/repository"
	"jkylin.com/kylin/v2/internal/service"
)

func newAuthzCache(t *testing.T) (*miniredis.Miniredis, *repository.AuthzCache) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return mr, repository.NewAuthzCache(client, 10*time.Minute)
}

func assertAuthz(t *testing.T, cache *repository.AuthzCache, userID int64, method, path string, want repository.AuthzResult) {
	t.Helper()
	got, err := cache.Check(context.Background(), userID, method, path)
	if err != nil || got != want {
		t.Fatalf("%s %s for user %d: got %d, %v; want %d", method, path, userID, got, err, want)
	}
}

func TestAuthzCache(t *testing.T) {
	mr, cache := newAuthzCache(t)
	ctx := context.Background()

	assertAuthz(t, cache, 1, "GET", "/users", repository.AuthzAPIMissing)
	// 空表也算已同步：接口未登记，而不是缓存缺失。
	if err := cache.SyncAPIPermissions(ctx, nil, "empty"); err != nil {
		t.Fatal(err)
	}
	assertAuthz(t, cache, 1, "GET", "/users", repository.AuthzUserMissing)
	if err := cache.SetUserPermissions(ctx, 1, nil); err != nil {
		t.Fatal(err)
	}
	assertAuthz(t, cache, 1, "GET", "/users", repository.AuthzNotMapped)

	if err := cache.SyncAPIPermissions(ctx, []model.ApiPermission{
		{Method: "GET", Path: "/users", Permission: "system.users"},
		{Method: "DELETE", Path: "/users/:id", Permission: "system.users"},
		{Method: "GET", Path: "/roles", Permission: "system.roles"},
	}, "v1"); err != nil {
		t.Fatal(err)
	}
	// 没有任何权限的用户（空集合）：拒绝，不会再次触发加载。
	assertAuthz(t, cache, 1, "GET", "/users", repository.AuthzDenied)
	if err := cache.SetUserPermissions(ctx, 2, []string{"system.users"}); err != nil {
		t.Fatal(err)
	}
	assertAuthz(t, cache, 2, "GET", "/users", repository.AuthzAllowed)
	assertAuthz(t, cache, 2, "DELETE", "/users/:id", repository.AuthzAllowed)
	assertAuthz(t, cache, 2, "GET", "/roles", repository.AuthzDenied)
	assertAuthz(t, cache, 2, "GET", "/unknown", repository.AuthzNotMapped)

	// 每次命中都为用户权限续期（空闲 TTL）。
	mr.FastForward(9 * time.Minute)
	assertAuthz(t, cache, 2, "GET", "/users", repository.AuthzAllowed)
	if ttl := mr.TTL("kylin:authz:user:2"); ttl != 10*time.Minute {
		t.Fatalf("user permission TTL should be refreshed: %v", ttl)
	}
	mr.FastForward(11 * time.Minute)
	assertAuthz(t, cache, 2, "GET", "/users", repository.AuthzUserMissing)

	// 重新同步整体替换：被移除的接口变为未登记，修改的接口立即使用新权限。
	if err := cache.SetUserPermissions(ctx, 2, []string{"system.users"}); err != nil {
		t.Fatal(err)
	}
	if err := cache.SyncAPIPermissions(ctx, []model.ApiPermission{{Method: "GET", Path: "/users", Permission: "logs.view"}}, "v2"); err != nil {
		t.Fatal(err)
	}
	assertAuthz(t, cache, 2, "GET", "/users", repository.AuthzDenied)
	assertAuthz(t, cache, 2, "DELETE", "/users/:id", repository.AuthzNotMapped)
	if keys := mr.Keys(); len(keys) != 2 {
		t.Fatalf("sync should not leave temp keys: %v", keys)
	}

	if err := cache.InvalidateUser(ctx, 2); err != nil {
		t.Fatal(err)
	}
	assertAuthz(t, cache, 2, "GET", "/users", repository.AuthzUserMissing)

	for i := int64(10); i < 1210; i++ { // 超过一个 SCAN 批次
		if err := cache.SetUserPermissions(ctx, i, []string{"logs.view"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := cache.InvalidateAllUsers(ctx); err != nil {
		t.Fatal(err)
	}
	for _, key := range mr.Keys() {
		if key != "kylin:authz:api" {
			t.Fatalf("user cache %s should be invalidated", key)
		}
	}
	assertAuthz(t, cache, 1000, "GET", "/users", repository.AuthzUserMissing)
}

// 缓存齐全时 Authorize 完全不访问数据库：仓储全部为 nil，访问即 panic。
func TestAuthorizeReadsOnlyRedis(t *testing.T) {
	_, cache := newAuthzCache(t)
	ctx := context.Background()
	if err := cache.SyncAPIPermissions(ctx, []model.ApiPermission{{Method: "GET", Path: "/users", Permission: "system.users"}}, "v1"); err != nil {
		t.Fatal(err)
	}
	for id, permissions := range map[int64][]string{1: {"system.users"}, 2: {"logs.view"}} {
		if err := cache.SetUserPermissions(ctx, id, permissions); err != nil {
			t.Fatal(err)
		}
	}
	authorizer := service.NewApiPermissionService(nil, nil, nil, cache)
	for _, tc := range []struct {
		userID          int64
		path            string
		mapped, allowed bool
	}{
		{userID: 1, path: "/users", mapped: true, allowed: true},
		{userID: 2, path: "/users", mapped: true, allowed: false},
		{userID: 1, path: "/roles", mapped: false, allowed: false},
	} {
		t.Run(strconv.FormatInt(tc.userID, 10)+tc.path, func(t *testing.T) {
			mapped, allowed, err := authorizer.Authorize(ctx, tc.userID, "GET", tc.path)
			if err != nil || mapped != tc.mapped || allowed != tc.allowed {
				t.Fatalf("got mapped=%v allowed=%v err=%v", mapped, allowed, err)
			}
		})
	}
}
