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
			assertStatus(t, request(engine, http.MethodGet, "/users", tc.header, ""), tc.status)
			assertStatus(t, request(engine, http.MethodGet, "/healthz", "", ""), http.StatusOK)
		})
	}

	engine := gin.New()
	router.Init(engine, db, "test-token")
	for _, input := range []struct{ method, path, body string }{
		{http.MethodPost, "/users", `{"username":"   ","password":"test-password"}`},
		{http.MethodPost, "/users", `{"username":"test","password":"` + strings.Repeat("密", 25) + `"}`},
		{http.MethodPost, "/users", `{"username":"test","password":"test-password","roles":[null]}`},
		{http.MethodPut, "/users/1", `{"version":1}`},
		{http.MethodPut, "/users/1", `{"enabled":""}`},
		{http.MethodGet, "/users/0", ""},
		{http.MethodGet, "/users?page=9223372036854775807&size=100", ""},
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

	const payload = `{"username":" Alice ","password":"test-password","nickname":"Example","roles":["operator","role,with,commas"]}`
	created := decodeData[model.User](t, call(http.MethodPost, "/users", payload, http.StatusOK))
	if created.ID <= 0 || created.Username != "Alice" || created.Version != 1 || created.Status != "Y" ||
		created.Enabled != "Y" || created.Roles == nil || len(*created.Roles) != 2 ||
		(*created.Roles)[1] != "role,with,commas" || created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatalf("unexpected user: %+v", created)
	}
	path := "/users/" + strconv.FormatInt(created.ID, 10)
	call(http.MethodPost, "/users", payload, http.StatusConflict)
	second := decodeData[model.User](t, call(http.MethodPost, "/users", `{"username":"Bob","password":"test-password"}`, http.StatusOK))
	if second.Roles == nil || len(*second.Roles) != 0 {
		t.Fatalf("expected default empty roles: %+v", second)
	}
	if err := db.Exec(migration).Error; err != nil {
		t.Fatal(err)
	}
	call(http.MethodGet, path, "", http.StatusOK)
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
		`{"password":"new-password","nickname":"","roles":[],"version":1}`, http.StatusOK))
	if updated.Version != 2 || updated.Nickname != "" || updated.Roles == nil || len(*updated.Roles) != 0 ||
		!updated.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("unexpected updated user: %+v", updated)
	}
	call(http.MethodPut, path, `{"remark":"stale","version":1}`, http.StatusConflict)
	var stored model.User
	if err := db.First(&stored, created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte("new-password")) != nil {
		t.Fatal("updated password was not stored as a bcrypt hash")
	}
	call(http.MethodDelete, path, "", http.StatusOK)
	call(http.MethodGet, path, "", http.StatusNotFound)
	call(http.MethodPut, path, `{"remark":"missing","version":2}`, http.StatusNotFound)
	call(http.MethodDelete, path, "", http.StatusNotFound)
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
