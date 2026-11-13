package bootstrap_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"jkylin.com/kylin/v2/internal/bootstrap"
	"jkylin.com/kylin/v2/internal/config"
)

func TestInitRedisDisabled(t *testing.T) {
	client, err := bootstrap.InitRedis(context.Background(), config.Redis{})
	if err != nil || client != nil {
		t.Fatalf("disabled: client=%v err=%v", client, err)
	}
}

func TestInitRedisValidation(t *testing.T) {
	for _, name := range []string{"url", "pool", "timeout", "idle"} {
		t.Run(name, func(t *testing.T) {
			cfg := config.Default().Redis
			cfg.Enabled = true
			switch name {
			case "url":
				cfg.URL = "redis://user:secret@localhost:bad/0"
			case "pool":
				cfg.PoolSize = 0
			case "timeout":
				cfg.ConnectTimeout = 0
			case "idle":
				cfg.MinIdleConns = cfg.PoolSize + 1
			}
			client, err := bootstrap.InitRedis(context.Background(), cfg)
			if err == nil || client != nil {
				t.Fatalf("expected error, client=%v err=%v", client, err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("error exposed password")
			}
		})
	}
}

func TestInitRedisCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cfg := config.Default().Redis
	cfg.Enabled = true
	client, err := bootstrap.InitRedis(ctx, cfg)
	if err == nil || client != nil {
		t.Fatalf("expected canceled initialization, client=%v err=%v", client, err)
	}
}

func TestInitRedisIntegration(t *testing.T) {
	url := os.Getenv("KYLIN_TEST_REDIS_URL")
	if url == "" {
		t.Skip("KYLIN_TEST_REDIS_URL is not set")
	}
	cfg := config.Default().Redis
	cfg.Enabled, cfg.URL = true, url
	client, err := bootstrap.InitRedis(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
}
