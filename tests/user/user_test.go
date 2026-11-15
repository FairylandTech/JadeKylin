/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-13 13:58:43 America/Los_Angeles
 *****************************************************************************/

package user_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"jkylin.com/kylin/v2/internal/bootstrap"
	"jkylin.com/kylin/v2/internal/config"
	"jkylin.com/kylin/v2/internal/contracts/requests"
	"jkylin.com/kylin/v2/internal/model"
	"jkylin.com/kylin/v2/internal/router"
)

func TestUserRouteConfiguration(t *testing.T) {
	db := new(gorm.DB)
	for _, tc := range []struct {
		name   string
		db     *gorm.DB
		token  string
		header string
		status int
	}{
		{name: "database disabled", token: "test-token", status: http.StatusNotFound},
		{name: "token absent", db: db, status: http.StatusNotFound},
		{name: "unauthenticated", db: db, token: "test-token", status: http.StatusUnauthorized},
		{name: "wrong token", db: db, token: "test-token", header: "Bearer wrong", status: http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := gin.New()
			router.Init(engine, tc.db, tc.token)
			for _, path := range []string{"/users", "/roles", "/permissions"} {
				assertStatus(t, request(engine, http.MethodGet, path, tc.header, ""), tc.status)
			}
			assertStatus(t, request(engine, http.MethodGet, "/healthz", "", ""), http.StatusOK)
		})
	}

	engine := gin.New()
	router.Init(engine, db, "test-token")
	for _, input := range []struct{ method, path, body string }{
		{http.MethodPost, "/users", `{"username":"   ","password":"test-password"}`},
		{http.MethodPost, "/users", `{"username":"test","password":"` + strings.Repeat("密", 25) + `"}`},
		{http.MethodPost, "/users", `{"username":"test","password":"test-password","roles":[null]}`},
		{http.MethodPut, "/users/1/password", `{"old_password":"old"}`},
		{http.MethodPut, "/users/1/password", `{"new_password":"new"}`},
		{http.MethodPut, "/users/0/password", `{"old_password":"old","new_password":"new"}`},
		{http.MethodPost, "/users/0/password/reset", ""},
		{http.MethodPut, "/users/1", `{"status":"X"}`},
		{http.MethodPut, "/users/1", `{"version":1}`},
		{http.MethodPut, "/users/1", `{"enabled":""}`},
		{http.MethodGet, "/users/0", ""},
		{http.MethodGet, "/users?page=9223372036854775807&size=100", ""},
		{http.MethodPost, "/roles", `{"code":"   ","name":"test"}`},
		{http.MethodPost, "/roles", `{"code":"test","name":"   "}`},
		{http.MethodPost, "/roles", `{"code":"test"}`},
		{http.MethodPost, "/roles", `{"code":"test","name":"test","permissions":[null]}`},
		{http.MethodPut, "/roles/1", `{}`},
		{http.MethodPut, "/roles/1", `{"code":"renamed"}`},
		{http.MethodPut, "/roles/1", `{"name":"  "}`},
		{http.MethodGet, "/roles/0", ""},
		{http.MethodGet, "/roles?page=9223372036854775807&size=100", ""},
	} {
		assertStatus(t, request(engine, input.method, input.path, "Bearer test-token", input.body), http.StatusBadRequest)
	}
}

func TestUserPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("KYLIN_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("KYLIN_TEST_POSTGRES_DSN is not set")
	}
	cfg := config.Default().Postgres
	cfg.Enabled, cfg.DSN = true, dsn
	db, closeDB, err := bootstrap.InitPostgres(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(closeDB)

	migration, err := os.ReadFile("../../migrations/01_v2_user.sql")
	if err != nil {
		t.Fatal(err)
	}
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatal(err)
	}
	schema := "user_test_" + hex.EncodeToString(suffix[:])
	if err := db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
			t.Error(err)
		}
	})
	if err := db.Connection(func(conn *gorm.DB) error {
		// 保留专用连接，同时让每次调用从独立的 GORM statement 开始。
		conn = conn.Session(&gorm.Session{NewDB: true})
		if err := conn.Exec("SET search_path TO " + schema).Error; err != nil {
			return err
		}
		defer func() {
			if err := conn.Exec("RESET search_path").Error; err != nil {
				t.Error(err)
			}
		}()
		if err := conn.Exec(string(migration)).Error; err != nil {
			return err
		}
		verifyCRUD(t, conn, string(migration))
		t.Run("role permission API", func(t *testing.T) {
			verifyRolePermissionAPI(t, conn)
		})
		t.Run("roles and permissions", func(t *testing.T) {
			verifyRoleAndPermissionRepositories(t, conn)
		})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func verifyCRUD(t *testing.T, db *gorm.DB, migration string) {
	t.Helper()
	engine := gin.New()
	const token = "test-token"
	router.Init(engine, db, token)
	call := func(method, path, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		res := request(engine, method, path, "Bearer "+token, body)
		assertStatus(t, res, want)
		if strings.Contains(res.Body.String(), `"password":`) {
			t.Fatalf("response contains password field: %s", res.Body.String())
		}
		return res
	}

	if err := db.Exec("INSERT INTO kylin_v2_roles (code, name) VALUES (?, ?)", "role,with,commas", "Comma Role").Error; err != nil {
		t.Fatal(err)
	}
	call(http.MethodPost, "/users", `{"username":"ghost","password":"test-password","roles":["no-such-role"]}`, http.StatusBadRequest)
	const payload = `{"username":" Alice ","password":"test-password","nickname":"Example","roles":["operator","role,with,commas"]}`
	created := decodeData[model.User](t, call(http.MethodPost, "/users", payload, http.StatusOK))
	if created.ID <= 0 || created.Username != "Alice" || created.Version != 1 || created.Status != "Y" ||
		created.Enabled != "Y" || created.Roles == nil || len(*created.Roles) != 2 ||
		(*created.Roles)[1] != "role,with,commas" || created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatalf("unexpected users: %+v", created)
	}
	path := "/users/" + strconv.FormatInt(created.ID, 10)
	call(http.MethodPost, "/users", payload, http.StatusConflict)
	second := decodeData[model.User](t, call(http.MethodPost, "/users", `{"username":"Bob","password":"test-password","enabled":"N","status":"N"}`, http.StatusOK))
	if second.Roles == nil || len(*second.Roles) != 0 || second.Enabled != "Y" || second.Status != "Y" {
		t.Fatalf("expected default empty roles and ignored enabled/status: %+v", second)
	}
	if err := db.Exec(migration).Error; err != nil {
		t.Fatal(err)
	}
	call(http.MethodGet, path, "", http.StatusOK)
	defaultPage := decodeData[struct {
		Page  int          `json:"page"`
		Size  int          `json:"size"`
		Total int64        `json:"total"`
		Pages int          `json:"pages"`
		Data  []model.User `json:"data"`
	}](t, call(http.MethodGet, "/users", "", http.StatusOK))
	if defaultPage.Page != 1 || defaultPage.Size != 10 || defaultPage.Total != 2 || defaultPage.Pages != 1 || len(defaultPage.Data) != 2 {
		t.Fatalf("unexpected default page: %+v", defaultPage)
	}
	for _, keyword := range []string{"%25", "_"} {
		list := decodeData[struct {
			Total int64 `json:"total"`
		}](t, call(http.MethodGet, "/users?keyword="+keyword, "", http.StatusOK))
		if list.Total != 0 {
			t.Fatalf("LIKE wildcard %q should be matched literally: %+v", keyword, list)
		}
	}
	for _, keyword := range []string{"ali", "AMP"} {
		list := decodeData[struct {
			Data  []model.User `json:"data"`
			Total int64        `json:"total"`
		}](t, call(http.MethodGet, "/users?page=1&size=1&keyword="+keyword, "", http.StatusOK))
		if list.Total != 1 || len(list.Data) != 1 || list.Data[0].ID != created.ID {
			t.Fatalf("unexpected page for %q: %+v", keyword, list)
		}
	}
	call(http.MethodPut, path, `{"username":"Bob","version":1}`, http.StatusConflict)
	updated := decodeData[model.User](t, call(http.MethodPut, path,
		`{"password":"ignored-password","nickname":"Updated Example","roles":[],"enabled":"N","version":1}`, http.StatusOK))
	if updated.Version != 2 || updated.Enabled != "Y" || updated.Nickname != "Updated Example" || updated.Roles == nil || len(*updated.Roles) != 0 ||
		!updated.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("unexpected updated users: %+v", updated)
	}
	call(http.MethodPut, path, `{"remark":"stale","version":1}`, http.StatusConflict)
	var stored model.User
	if err := db.First(&stored, created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte("test-password")) != nil {
		t.Fatal("PUT /users/:id must not change the password")
	}

	passwordPath := path + "/password"
	call(http.MethodPut, passwordPath, `{"old_password":"wrong-password","new_password":"new-password"}`, http.StatusBadRequest)
	call(http.MethodPut, passwordPath, `{"old_password":"test-password","new_password":"test-password"}`, http.StatusBadRequest)
	call(http.MethodPut, passwordPath, `{"old_password":"test-password","new_password":"   "}`, http.StatusBadRequest)
	call(http.MethodPut, passwordPath, `{"old_password":"test-password","new_password":"new-password"}`, http.StatusOK)
	if err := db.First(&stored, created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte("new-password")) != nil || stored.Version != updated.Version+1 {
		t.Fatalf("changed password not stored as bcrypt hash or version not bumped: version=%d", stored.Version)
	}
	call(http.MethodPut, passwordPath, `{"old_password":"test-password","new_password":"other-password"}`, http.StatusBadRequest)

	resetRes := call(http.MethodPost, passwordPath+"/reset", "", http.StatusOK)
	if resetRes.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("reset response must not be cached: %v", resetRes.Header())
	}
	reset := decodeData[requests.ResetPasswordResponse](t, resetRes)
	if reset.ID != created.ID || len(reset.NewPassword) != 16 {
		t.Fatalf("unexpected reset response: %+v", reset)
	}
	if err := db.First(&stored, created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Password == reset.NewPassword || bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte(reset.NewPassword)) != nil {
		t.Fatal("reset password must be stored as a bcrypt hash of the returned plaintext")
	}
	call(http.MethodPut, passwordPath, `{"old_password":"`+reset.NewPassword+`","new_password":"after-reset"}`, http.StatusOK)
	updated.Version = stored.Version + 1
	call(http.MethodDelete, path, "", http.StatusOK)
	var deleted model.User
	if err := db.First(&deleted, created.ID).Error; err != nil {
		t.Fatalf("deleted user should remain in the database: %v", err)
	}
	if deleted.Enabled != "N" || deleted.Version != updated.Version+1 || !deleted.UpdatedAt.After(updated.UpdatedAt) {
		t.Fatalf("unexpected soft-deleted user state: enabled=%q version=%d updated_at=%v", deleted.Enabled, deleted.Version, deleted.UpdatedAt)
	}
	for _, tc := range []struct {
		path  string
		total int64
	}{
		{path: "/users", total: 1},
		{path: "/users?keyword=alice", total: 0},
		{path: "/users?keyword=example", total: 0},
	} {
		list := decodeData[struct {
			Data  []model.User `json:"data"`
			Total int64        `json:"total"`
		}](t, call(http.MethodGet, tc.path, "", http.StatusOK))
		if list.Total != tc.total || len(list.Data) != int(tc.total) || (tc.total == 1 && list.Data[0].ID != second.ID) {
			t.Fatalf("unexpected page after soft delete for %s: %+v", tc.path, list)
		}
	}
	call(http.MethodGet, path, "", http.StatusNotFound)
	call(http.MethodPut, path, `{"remark":"missing","version":2}`, http.StatusNotFound)
	call(http.MethodPut, path, `{"remark":"missing"}`, http.StatusNotFound)
	call(http.MethodDelete, path, "", http.StatusNotFound)
	call(http.MethodPut, passwordPath, `{"old_password":"after-reset","new_password":"x-password"}`, http.StatusNotFound)
	call(http.MethodPost, passwordPath+"/reset", "", http.StatusNotFound)
	// 软删除的用户不占用用户名，可以重建；重建后的用户名仍然唯一。
	recreated := decodeData[model.User](t, call(http.MethodPost, "/users", payload, http.StatusOK))
	if recreated.ID == created.ID || recreated.Username != "Alice" {
		t.Fatalf("unexpected recreated user: %+v", recreated)
	}
	call(http.MethodPost, "/users", payload, http.StatusConflict)
}

func request(engine http.Handler, method, path, header, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", header)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, req)
	return res
}

func assertStatus(t *testing.T, res *httptest.ResponseRecorder, want int) {
	t.Helper()
	if res.Code != want {
		t.Fatalf("status=%d want=%d body=%s", res.Code, want, res.Body.String())
	}
}

func decodeData[T any](t *testing.T, res *httptest.ResponseRecorder) T {
	t.Helper()
	var reply struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	return reply.Data
}

func verifyRolePermissionAPI(t *testing.T, db *gorm.DB) {
	t.Helper()
	engine := gin.New()
	const token = "test-token"
	router.Init(engine, db, token)
	call := func(method, path, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		res := request(engine, method, path, "Bearer "+token, body)
		assertStatus(t, res, want)
		return res
	}

	allPermissions := decodeData[[]model.Permission](t, call(http.MethodGet, "/permissions", "", http.StatusOK))
	if len(allPermissions) < 15 {
		t.Fatalf("permissions should return every seeded permission without paging: %d", len(allPermissions))
	}
	// 按分类排序：同一分类的权限必须连续出现（具体先后取决于数据库排序规则）。
	seen := make(map[string]bool)
	for i, permission := range allPermissions {
		if i > 0 && allPermissions[i-1].Category == permission.Category {
			continue
		}
		if seen[permission.Category] {
			t.Fatalf("permissions should be grouped by category: %+v", allPermissions)
		}
		seen[permission.Category] = true
	}

	call(http.MethodPost, "/roles", `{"code":"ghost","name":"Ghost","permissions":["no.such.permission"]}`, http.StatusBadRequest)
	created := decodeData[model.Role](t, call(http.MethodPost, "/roles",
		`{"code":" viewer ","name":" Viewer ","permissions":["logs.view"," logs.view ","dashboard.view"]}`, http.StatusOK))
	if created.ID <= 0 || created.Code != "viewer" || created.Name != "Viewer" || created.Enabled != "Y" ||
		created.Permissions == nil || len(*created.Permissions) != 2 || (*created.Permissions)[0] != "logs.view" {
		t.Fatalf("unexpected created role: %+v", created)
	}
	call(http.MethodPost, "/roles", `{"code":"viewer","name":"Another"}`, http.StatusConflict)
	empty := decodeData[model.Role](t, call(http.MethodPost, "/roles", `{"code":"empty","name":"Empty"}`, http.StatusOK))
	if empty.Permissions == nil || len(*empty.Permissions) != 0 {
		t.Fatalf("expected default empty permissions: %+v", empty)
	}

	path := "/roles/" + strconv.FormatInt(created.ID, 10)
	fetched := decodeData[model.Role](t, call(http.MethodGet, path, "", http.StatusOK))
	if fetched.Permissions == nil || len(*fetched.Permissions) != 2 {
		t.Fatalf("unexpected fetched role: %+v", fetched)
	}
	page := decodeData[struct {
		Total int64        `json:"total"`
		Data  []model.Role `json:"data"`
	}](t, call(http.MethodGet, "/roles?keyword=VIEW&size=1", "", http.StatusOK))
	if page.Total != 1 || len(page.Data) != 1 || page.Data[0].ID != created.ID {
		t.Fatalf("unexpected role page: %+v", page)
	}

	call(http.MethodPut, path, `{"permissions":["logs.view","no.such.permission"]}`, http.StatusBadRequest)
	call(http.MethodPut, path, `{"name":"Empty"}`, http.StatusConflict)
	updated := decodeData[model.Role](t, call(http.MethodPut, path, `{"description":"Read only","permissions":["research.view"]}`, http.StatusOK))
	if updated.Code != "viewer" || updated.Description != "Read only" || updated.Permissions == nil ||
		len(*updated.Permissions) != 1 || (*updated.Permissions)[0] != "research.view" {
		t.Fatalf("unexpected updated role: %+v", updated)
	}
	cleared := decodeData[model.Role](t, call(http.MethodPut, path, `{"permissions":[]}`, http.StatusOK))
	if cleared.Permissions == nil || len(*cleared.Permissions) != 0 {
		t.Fatalf("expected cleared permissions: %+v", cleared)
	}

	// 用户只能关联启用的角色：角色删除后不能再被分配。
	call(http.MethodPost, "/users", `{"username":"viewer-user","password":"test-password","roles":["viewer"]}`, http.StatusOK)
	call(http.MethodDelete, path, "", http.StatusOK)
	call(http.MethodGet, path, "", http.StatusNotFound)
	call(http.MethodPut, path, `{"name":"Gone"}`, http.StatusNotFound)
	call(http.MethodDelete, path, "", http.StatusNotFound)
	call(http.MethodPost, "/users", `{"username":"viewer-user-2","password":"test-password","roles":["viewer"]}`, http.StatusBadRequest)
}
