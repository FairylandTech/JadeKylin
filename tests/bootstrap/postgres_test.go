package bootstrap_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"jkylin.com/kylin/v2/internal/bootstrap"
	"jkylin.com/kylin/v2/internal/config"
)

func TestInitPostgresDisabled(t *testing.T) {
	db, closePostgres, err := bootstrap.InitPostgres(context.Background(), config.Postgres{})
	if err != nil || db != nil || closePostgres != nil {
		t.Fatalf("disabled: db=%v cleanup provided=%t err=%v", db, closePostgres != nil, err)
	}
}

func TestInitPostgresValidation(t *testing.T) {
	for _, name := range []string{"missing dsn", "invalid pool", "invalid timeout", "malformed dsn"} {
		t.Run(name, func(t *testing.T) {
			cfg := config.Default().Postgres
			cfg.Enabled = true
			cfg.DSN = "postgres://localhost/test"
			switch name {
			case "missing dsn":
				cfg.DSN = ""
			case "invalid pool":
				cfg.MinConns = cfg.MaxConns + 1
			case "invalid timeout":
				cfg.ConnectTimeout = 0
			case "malformed dsn":
				cfg.DSN = "postgres://user:secret@localhost:bad/test"
			}
			db, closePostgres, err := bootstrap.InitPostgres(context.Background(), cfg)
			if err == nil || db != nil || closePostgres != nil {
				t.Fatalf("expected validation error, db=%v cleanup provided=%t err=%v", db, closePostgres != nil, err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("error exposed password")
			}
		})
	}
}

// 设置 KYLIN_TEST_POSTGRES_DSN 后执行真实数据库连通性验证。
func TestInitPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("KYLIN_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("KYLIN_TEST_POSTGRES_DSN is not set")
	}
	cfg := config.Default().Postgres
	cfg.Enabled, cfg.DSN = true, dsn
	db, closePostgres, err := bootstrap.InitPostgres(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer closePostgres()
	var result int
	if err := db.WithContext(context.Background()).Raw("SELECT 1").Scan(&result).Error; err != nil {
		t.Fatal(err)
	}
	if result != 1 {
		t.Fatalf("SELECT 1 returned %d", result)
	}
}
