/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-13 11:54:00 America/Los_Angeles
 *****************************************************************************/

package bootstrap_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"jkylin.com/kylin/v2/internal/bootstrap"
	"jkylin.com/kylin/v2/internal/config"
)

func TestInitializeRedisDisabled(t *testing.T) {
	client, err := bootstrap.InitializeRedis(context.Background(), config.Redis{})
	if err != nil || client != nil {
		t.Fatalf("disabled: client=%v err=%v", client, err)
	}
}

func TestInitializeRedisValidation(t *testing.T) {
	for _, name := range []string{"url", "pool", "timeout", "idle"} {
		t.Run(name, func(t *testing.T) {
			configuration := config.Default().Redis
			configuration.Enabled = true
			switch name {
			case "url":
				configuration.URL = "redis://users:secret@localhost:bad/0"
			case "pool":
				configuration.PoolSize = 0
			case "timeout":
				configuration.ConnectTimeout = 0
			case "idle":
				configuration.MinIdleConnections = configuration.PoolSize + 1
			}
			client, err := bootstrap.InitializeRedis(context.Background(), configuration)
			if err == nil || client != nil {
				t.Fatalf("expected error, client=%v err=%v", client, err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("error exposed password")
			}
		})
	}
}

func TestInitializeRedisCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	configuration := config.Default().Redis
	configuration.Enabled = true
	client, err := bootstrap.InitializeRedis(ctx, configuration)
	if err == nil || client != nil {
		t.Fatalf("expected canceled initialization, client=%v err=%v", client, err)
	}
}

func TestInitializeRedisIntegration(t *testing.T) {
	url := os.Getenv("KYLIN_TEST_REDIS_URL")
	if url == "" {
		t.Skip("KYLIN_TEST_REDIS_URL is not set")
	}
	configuration := config.Default().Redis
	configuration.Enabled, configuration.URL = true, url
	client, err := bootstrap.InitializeRedis(context.Background(), configuration)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
}
