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

func TestInitializePostgresDisabled(t *testing.T) {
	database, closePostgres, err := bootstrap.InitializePostgres(context.Background(), config.Postgres{})
	if err != nil || database != nil || closePostgres != nil {
		t.Fatalf("disabled: database=%v cleanup provided=%t err=%v", database, closePostgres != nil, err)
	}
}

func TestInitializePostgresValidation(t *testing.T) {
	for _, name := range []string{"missing dsn", "invalid pool", "invalid timeout", "malformed dsn"} {
		t.Run(name, func(t *testing.T) {
			configuration := config.Default().Postgres
			configuration.Enabled = true
			configuration.DSN = "postgres://localhost/test"
			switch name {
			case "missing dsn":
				configuration.DSN = ""
			case "invalid pool":
				configuration.MinConnections = configuration.MaxConnections + 1
			case "invalid timeout":
				configuration.ConnectTimeout = 0
			case "malformed dsn":
				configuration.DSN = "postgres://users:secret@localhost:bad/test"
			}
			database, closePostgres, err := bootstrap.InitializePostgres(context.Background(), configuration)
			if err == nil || database != nil || closePostgres != nil {
				t.Fatalf("expected validation error, database=%v cleanup provided=%t err=%v", database, closePostgres != nil, err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("error exposed password")
			}
		})
	}
}

// 设置 KYLIN_TEST_POSTGRES_DSN 后执行真实数据库连通性验证。
func TestInitializePostgresIntegration(t *testing.T) {
	dsn := os.Getenv("KYLIN_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("KYLIN_TEST_POSTGRES_DSN is not set")
	}
	configuration := config.Default().Postgres
	configuration.Enabled, configuration.DSN = true, dsn
	database, closePostgres, err := bootstrap.InitializePostgres(context.Background(), configuration)
	if err != nil {
		t.Fatal(err)
	}
	defer closePostgres()
	var result int
	if err := database.WithContext(context.Background()).Raw("SELECT 1").Scan(&result).Error; err != nil {
		t.Fatal(err)
	}
	if result != 1 {
		t.Fatalf("SELECT 1 returned %d", result)
	}
}
