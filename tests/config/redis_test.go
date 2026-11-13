package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"jkylin.com/kylin/v2/internal/config"
)

func TestLoadRedis(t *testing.T) {
	previous := *config.Get()
	t.Cleanup(func() { *config.Get() = previous })
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("redis:\n  enabled: true\n  url: redis://localhost:6379/1\n  pool_size: 20\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KYLIN_REDIS_URL", "redis://localhost:6379/2")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Redis.Enabled || cfg.Redis.URL != "redis://localhost:6379/2" || cfg.Redis.PoolSize != 20 || cfg.Redis.ConnectTimeout != 5 {
		t.Fatal("unexpected redis config")
	}
	cfg, err = config.Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Redis.Enabled || cfg.Redis.PoolSize != 10 {
		t.Fatal("previous config leaked into defaults")
	}
}
