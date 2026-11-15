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
	"gorm.io/gorm/logger"
	"jkylin.com/kylin/v2/internal/bootstrap"
)

func TestZapGormLoggerTrace(t *testing.T) {
	const sql = "SELECT * FROM kylin_v2_users WHERE id = $1"
	describeStatement := func() (string, int64) {
		return sql, 1
	}
	recent := time.Now()
	slow := recent.Add(-time.Second)

	for _, testCase := range []struct {
		name     string
		zapLevel zapcore.Level
		mode     logger.LogLevel
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
		{name: "silent mode", zapLevel: zap.DebugLevel, mode: logger.Silent, begin: slow, err: errors.New("boom")},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			core, logs := observer.New(testCase.zapLevel)
			ctx := tracedContext(zap.New(core).With(zap.String("trace_id", "t-1")), "t-1")

			var gormLogger logger.Interface = bootstrap.NewZapGormLogger(200 * time.Millisecond)
			if testCase.mode != 0 {
				gormLogger = gormLogger.LogMode(testCase.mode)
			}
			gormLogger.Trace(ctx, testCase.begin, describeStatement, testCase.err)

			const plainMessage = "trace id: t-1, SQL: " + sql + ";"
			plain := logs.FilterMessage(plainMessage).AllUntimed()
			if len(plain) != map[bool]int{true: 1}[testCase.plain] || (testCase.plain && plain[0].Level != zap.DebugLevel) {
				t.Fatalf("plain sql entries: %+v (all: %+v)", plain, logs.AllUntimed())
			}
			entries := logs.Filter(func(entry observer.LoggedEntry) bool {
				return entry.Message != plainMessage
			}).AllUntimed()
			if len(entries) != len(testCase.want) {
				t.Fatalf("got %d entries, want %d: %+v", len(entries), len(testCase.want), entries)
			}
			for index, entry := range entries {
				fields := entry.ContextMap()
				if entry.Level != testCase.want[index] || fields["sql"] != sql || fields["trace_id"] != "t-1" || fields["source"] == "" {
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
	sql, parameters := bootstrap.NewZapGormLogger(0).ParamsFilter(context.Background(),
		"INSERT INTO kylin_v2_users (username,password,version) VALUES ($1,$2,$3)", "Alice", hashString, 1)
	if sql != "INSERT INTO kylin_v2_users (username,password,version) VALUES ($1,$2,$3)" || len(parameters) != 3 ||
		parameters[0] != "Alice" || parameters[1] != "***" || parameters[2] != 1 {
		t.Fatalf("unexpected filtered parameters: %q %v", sql, parameters)
	}
	if _, pointerParameters := bootstrap.NewZapGormLogger(0).ParamsFilter(context.Background(), "", &hashString); pointerParameters[0] != "***" {
		t.Fatalf("pointer hash should be redacted: %v", pointerParameters)
	}

	// 用 postgres dialector 真实的 Explain 代入参数，确认最终写进日志的 SQL 带值且哈希已脱敏。
	explained := postgres.New(postgres.Config{}).Explain(sql, parameters...)
	if want := "INSERT INTO kylin_v2_users (username,password,version) VALUES ('Alice','***',1)"; explained != want {
		t.Fatalf("explained=%q want=%q", explained, want)
	}
}

func TestZapGormLoggerPlainMessageWithoutTraceID(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	ctx := loggerContext(zap.New(core))
	bootstrap.NewZapGormLogger(0).Trace(ctx, time.Now(), func() (string, int64) {
		return "SELECT 1;", 1
	}, nil)
	if logs.FilterMessage("trace id: -, SQL: SELECT 1;").Len() != 1 {
		t.Fatalf("unexpected entries: %+v", logs.AllUntimed())
	}
}
