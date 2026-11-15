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

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"jkylin.com/kylin/v2/internal/config"
)

func TestLoadPostgres(t *testing.T) {
	previous := *config.Get()
	t.Cleanup(func() {
		*config.Get() = previous
	})
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("postgres:\n  enabled: true\n  dsn: postgres://localhost/from-file\n  max_conns: 20\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KYLIN_DATABASE_DSN", "postgres://localhost/from-env")
	configuration, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !configuration.Postgres.Enabled || configuration.Postgres.DSN != "postgres://localhost/from-env" || configuration.Postgres.MaxConnections != 20 || configuration.Postgres.ConnectTimeout != 5 {
		t.Fatalf("unexpected postgres config: %+v", configuration.Postgres)
	}
	configuration, err = config.Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Postgres.Enabled || configuration.Postgres.MaxConnections != 10 {
		t.Fatalf("previous config leaked into defaults: %+v", configuration.Postgres)
	}
	if err := os.Unsetenv("KYLIN_DATABASE_DSN"); err != nil {
		t.Fatal(err)
	}
	configuration, err = config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Postgres.DSN != "postgres://localhost/from-file" {
		t.Fatal("PostgreSQL DSN was not loaded from the config file")
	}
}

func TestLoadAuth(t *testing.T) {
	previous := *config.Get()
	t.Cleanup(func() {
		*config.Get() = previous
	})
	directory := t.TempDir()
	configuration, err := config.Load(filepath.Join(directory, "missing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Auth.TokenTTL != 900 || configuration.Auth.IdleTimeout != 600 || configuration.Auth.MaxSessions != 1 || configuration.Auth.CaptchaTTL != 120 || configuration.Auth.JWTSecret != "" {
		t.Fatalf("unexpected default auth config: %+v", configuration.Auth)
	}
	t.Setenv("KYLIN_JWT_SECRET", "0123456789abcdef0123456789abcdef")
	if configuration, err = config.Load(filepath.Join(directory, "missing.yaml")); err != nil || configuration.Auth.JWTSecret != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("jwt secret from env: %+v, %v", configuration, err)
	}
	for name, body := range map[string]string{
		"short secret": "auth:\n  jwt_secret: short\n",
		"idle >= ttl":  "auth:\n  token_ttl: 600\n  idle_timeout: 600\n",
		"negative max": "auth:\n  max_sessions: -1\n",
		"zero captcha": "auth:\n  captcha_ttl: 0\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("KYLIN_JWT_SECRET", "")
			_ = os.Unsetenv("KYLIN_JWT_SECRET")
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := config.Load(path); err == nil {
				t.Fatalf("expected validation error for %q", body)
			}
		})
	}
}

func TestRunModeSkipAuth(t *testing.T) {
	previous := *config.Get()
	t.Cleanup(func() {
		*config.Get() = previous
	})
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	t.Setenv("GIN_MODE", "debug")
	for runMode, skip := range map[string]bool{"local": true, "dev": true, " LOCAL ": true, "prod": false, "test": false, "": false} {
		t.Setenv("RUN_MODE", runMode)
		configuration, err := config.Load(missing)
		if err != nil || configuration.App.SkipAuth() != skip {
			t.Fatalf("RUN_MODE=%q: skip=%v err=%v", runMode, configuration != nil && configuration.App.SkipAuth(), err)
		}
	}
	// 跳过认证不能与 release 同时使用。
	t.Setenv("GIN_MODE", "release")
	for _, runMode := range []string{"local", "dev"} {
		t.Setenv("RUN_MODE", runMode)
		if _, err := config.Load(missing); err == nil {
			t.Fatalf("RUN_MODE=%s with release mode should be rejected", runMode)
		}
	}
	t.Setenv("RUN_MODE", "prod")
	if _, err := config.Load(missing); err != nil {
		t.Fatal(err)
	}
}
