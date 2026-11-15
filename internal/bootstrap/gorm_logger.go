/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-14 20:21:58 America/Los_Angeles
 *****************************************************************************/

package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
	"jkylin.com/kylin/v2/pkg/logger"
)

// ZapGormLogger 把 GORM 日志交给 zap，和应用日志共用输出（控制台 / 文件）与级别：
//   - SQL 报错（不含 record not found）→ Error
//   - 慢查询 → Warn
//   - 其余 SQL 只在 zap 开启 debug 级别时输出 → Debug
//
// 日志器取自 ctx（logger.FromContext），因此请求内的 SQL 日志自带 trace_id。
type ZapGormLogger struct {
	SlowThreshold time.Duration
	level         gormlogger.LogLevel
}

var (
	_ gormlogger.Interface = (*ZapGormLogger)(nil)
	_ gorm.ParamsFilter    = (*ZapGormLogger)(nil)
)

func NewZapGormLogger(slowThreshold time.Duration) *ZapGormLogger {
	return &ZapGormLogger{SlowThreshold: slowThreshold, level: gormlogger.Info}
}

// LogMode 支持 db.Session(&gorm.Session{Logger: l.LogMode(gormlogger.Silent)}) 这类局部调整。
func (l *ZapGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	cp := *l
	cp.level = level
	return &cp
}

func (l *ZapGormLogger) Info(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Info {
		l.logger(ctx).Info(fmt.Sprintf(msg, args...))
	}
}

func (l *ZapGormLogger) Warn(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Warn {
		l.logger(ctx).Warn(fmt.Sprintf(msg, args...))
	}
}

func (l *ZapGormLogger) Error(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Error {
		l.logger(ctx).Error(fmt.Sprintf(msg, args...))
	}
}

func (l *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= gormlogger.Silent {
		return
	}
	log := l.logger(ctx)
	elapsed := time.Since(begin)

	var lvl zapcore.Level
	msg := "sql"
	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && l.level >= gormlogger.Error:
		lvl, msg = zapcore.ErrorLevel, "sql error"
	case l.SlowThreshold > 0 && elapsed > l.SlowThreshold && l.level >= gormlogger.Warn:
		lvl, msg = zapcore.WarnLevel, "slow sql"
	case l.level >= gormlogger.Info:
		lvl = zapcore.DebugLevel
	default:
		return
	}
	ce := log.Check(lvl, msg)
	if ce == nil { // 级别未开启时不拼 SQL
		return
	}
	sql, rows := fc()
	fields := []zap.Field{
		zap.String("sql", sql),
		zap.Int64("rows", rows),
		zap.Duration("elapsed", elapsed),
		zap.String("source", utils.FileWithLineNum()),
	}
	if lvl == zapcore.ErrorLevel {
		fields = append(fields, zap.Error(err))
	}
	ce.Write(fields...)

	// 额外一条纯文本 DEBUG，方便直接复制 SQL 排查。
	if log.Core().Enabled(zapcore.DebugLevel) {
		log.Debug(plainSQLMessage(ctx, sql))
	}
}

func plainSQLMessage(ctx context.Context, sql string) string {
	traceID := logger.TraceID(ctx)
	if traceID == "" {
		traceID = "-" // 非请求内的 SQL（启动、后台任务）没有 trace id
	}
	return fmt.Sprintf("trace id: %s, SQL: %s;", traceID, strings.TrimRight(strings.TrimSpace(sql), ";"))
}

// ParamsFilter 决定日志里 SQL 代入哪些参数值：保留真实值方便排查，
// 但把 bcrypt 哈希（$2a$ / $2b$ / $2y$ 开头）替换成 ***，避免密码哈希落盘。
// 只影响日志，不影响实际执行的 SQL。
func (l *ZapGormLogger) ParamsFilter(_ context.Context, sql string, params ...any) (string, []any) {
	redacted := make([]any, len(params))
	for i, param := range params {
		if isPasswordHash(param) {
			redacted[i] = "***"
		} else {
			redacted[i] = param
		}
	}
	return sql, redacted
}

func isPasswordHash(param any) bool {
	var value string
	switch v := param.(type) {
	case string:
		value = v
	case *string:
		if v == nil {
			return false
		}
		value = *v
	default:
		return false
	}
	return len(value) == 60 && (strings.HasPrefix(value, "$2a$") || strings.HasPrefix(value, "$2b$") || strings.HasPrefix(value, "$2y$"))
}

// logger 关闭 zap 自带的 caller（它总指向本文件），真实调用位置见 source 字段。
func (l *ZapGormLogger) logger(ctx context.Context) *zap.Logger {
	return logger.FromContext(ctx).WithOptions(zap.WithCaller(false), zap.AddStacktrace(zapcore.FatalLevel))
}
