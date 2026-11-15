/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 20:17:31 America/Los_Angeles
 *****************************************************************************/

package bootstrap_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"jkylin.com/kylin/v2/internal/bootstrap"
	"jkylin.com/kylin/v2/pkg/logger"
)

func TestZapGormLoggerTrace(t *testing.T) {
	const sql = "SELECT * FROM kylin_v2_users WHERE id = $1"
	fc := func() (string, int64) { return sql, 1 }
	recent := time.Now()
	slow := recent.Add(-time.Second)

	for _, tc := range []struct {
		name     string
		zapLevel zapcore.Level
		mode     gormlogger.LogLevel
		begin    time.Time
		err      error
		want     []zapcore.Level // 结构化 SQL 日志的级别
		plain    bool            // 是否额外输出纯文本 DEBUG
	}{
		{name: "debug prints every sql", zapLevel: zap.DebugLevel, begin: recent, want: []zapcore.Level{zap.DebugLevel}, plain: true},
		{name: "debug slow sql", zapLevel: zap.DebugLevel, begin: slow, want: []zapcore.Level{zap.WarnLevel}, plain: true},
		{name: "info hides normal sql", zapLevel: zap.InfoLevel, begin: recent},
		{name: "slow sql warns", zapLevel: zap.InfoLevel, begin: slow, want: []zapcore.Level{zap.WarnLevel}},
		{name: "sql error", zapLevel: zap.InfoLevel, begin: recent, err: errors.New("boom"), want: []zapcore.Level{zap.ErrorLevel}},
		{name: "record not found is not an error", zapLevel: zap.InfoLevel, begin: recent, err: gorm.ErrRecordNotFound},
		{name: "silent mode", zapLevel: zap.DebugLevel, mode: gormlogger.Silent, begin: slow, err: errors.New("boom")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core, logs := observer.New(tc.zapLevel)
			ctx := logger.WithTraceID(logger.NewContext(context.Background(), zap.New(core).With(zap.String("trace_id", "t-1"))), "t-1")

			var gl gormlogger.Interface = bootstrap.NewZapGormLogger(200 * time.Millisecond)
			if tc.mode != 0 {
				gl = gl.LogMode(tc.mode)
			}
			gl.Trace(ctx, tc.begin, fc, tc.err)

			const plainMessage = "trace id: t-1, SQL: " + sql + ";"
			plain := logs.FilterMessage(plainMessage).AllUntimed()
			if len(plain) != map[bool]int{true: 1}[tc.plain] || (tc.plain && plain[0].Level != zap.DebugLevel) {
				t.Fatalf("plain sql entries: %+v (all: %+v)", plain, logs.AllUntimed())
			}
			entries := logs.Filter(func(e observer.LoggedEntry) bool { return e.Message != plainMessage }).AllUntimed()
			if len(entries) != len(tc.want) {
				t.Fatalf("got %d entries, want %d: %+v", len(entries), len(tc.want), entries)
			}
			for i, entry := range entries {
				fields := entry.ContextMap()
				if entry.Level != tc.want[i] || fields["sql"] != sql || fields["trace_id"] != "t-1" || fields["source"] == "" {
					t.Fatalf("unexpected entry: level=%v fields=%v", entry.Level, fields)
				}
			}
		})
	}
}

func TestZapGormLoggerParamsFilter(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	hashString := string(hash)
	sql, params := bootstrap.NewZapGormLogger(0).ParamsFilter(context.Background(),
		"INSERT INTO kylin_v2_users (username,password,version) VALUES ($1,$2,$3)", "Alice", hashString, 1)
	if sql != "INSERT INTO kylin_v2_users (username,password,version) VALUES ($1,$2,$3)" || len(params) != 3 ||
		params[0] != "Alice" || params[1] != "***" || params[2] != 1 {
		t.Fatalf("unexpected filtered params: %q %v", sql, params)
	}
	if _, pointerParams := bootstrap.NewZapGormLogger(0).ParamsFilter(context.Background(), "", &hashString); pointerParams[0] != "***" {
		t.Fatalf("pointer hash should be redacted: %v", pointerParams)
	}

	// 用 postgres dialector 真实的 Explain 代入参数，确认最终写进日志的 SQL 带值且哈希已脱敏。
	explained := postgres.New(postgres.Config{}).Explain(sql, params...)
	if want := "INSERT INTO kylin_v2_users (username,password,version) VALUES ('Alice','***',1)"; explained != want {
		t.Fatalf("explained=%q want=%q", explained, want)
	}
}

func TestZapGormLoggerPlainMessageWithoutTraceID(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	ctx := logger.NewContext(context.Background(), zap.New(core))
	bootstrap.NewZapGormLogger(0).Trace(ctx, time.Now(), func() (string, int64) { return "SELECT 1;", 1 }, nil)
	if logs.FilterMessage("trace id: -, SQL: SELECT 1;").Len() != 1 {
		t.Fatalf("unexpected entries: %+v", logs.AllUntimed())
	}
}
