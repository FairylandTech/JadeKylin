package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"jkylin.com/kylin/v2/internal/config"
)

func TestLoadPostgres(t *testing.T) {
	previous := *config.Get()
	t.Cleanup(func() { *config.Get() = previous })
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("postgres:\n  enabled: true\n  dsn: postgres://localhost/from-file\n  max_conns: 20\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KYLIN_POSTGRES_DSN", "postgres://localhost/from-env")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Postgres.Enabled || cfg.Postgres.DSN != "postgres://localhost/from-env" || cfg.Postgres.MaxConns != 20 || cfg.Postgres.ConnectTimeout != 5 {
		t.Fatalf("unexpected postgres config: %+v", cfg.Postgres)
	}
	cfg, err = config.Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Postgres.Enabled || cfg.Postgres.MaxConns != 10 {
		t.Fatalf("previous config leaked into defaults: %+v", cfg.Postgres)
	}
}
