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
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

const (
	// slowQueryThreshold 是 SQL 被记为慢查询的耗时阈值。
	slowQueryThreshold = 200 * time.Millisecond
	// bcryptHashLength 是 bcrypt 哈希的固定长度。
	bcryptHashLength = 60
	// redactedParameter 是日志中替换密码哈希的占位符。
	redactedParameter = "***"
	// missingTraceID 用于非请求内的 SQL（启动、后台任务），它们没有 trace id。
	missingTraceID = "-"
)

// bcryptHashPrefixes 是 bcrypt 哈希可能的版本前缀。
var bcryptHashPrefixes = []string{"$2a$", "$2b$", "$2y$"}

// ZapGormLogger 把 GORM 日志交给 zap，和应用日志共用输出（控制台 / 文件）与级别：
//   - SQL 报错（不含 record not found）→ Error
//   - 慢查询 → Warn
//   - 其余 SQL 只在 zap 开启 debug 级别时输出 → Debug
//
// 日志器取自 ctx（见 contextLogger），因此请求内的 SQL 日志自带 trace_id。
type ZapGormLogger struct {
	SlowThreshold time.Duration
	level         logger.LogLevel
}

var (
	_ logger.Interface  = (*ZapGormLogger)(nil)
	_ gorm.ParamsFilter = (*ZapGormLogger)(nil)
)

func NewZapGormLogger(slowThreshold time.Duration) *ZapGormLogger {
	return &ZapGormLogger{SlowThreshold: slowThreshold, level: logger.Info}
}

// LogMode 支持 database.Session(&gorm.Session{Logger: zapGormLogger.LogMode(logger.Silent)}) 这类局部调整。
func (zapGormLogger *ZapGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	copied := *zapGormLogger
	copied.level = level
	return &copied
}

func (zapGormLogger *ZapGormLogger) Info(ctx context.Context, message string, arguments ...any) {
	if zapGormLogger.level >= logger.Info {
		zapGormLogger.statementLogger(ctx).Info(fmt.Sprintf(message, arguments...))
	}
}

func (zapGormLogger *ZapGormLogger) Warn(ctx context.Context, message string, arguments ...any) {
	if zapGormLogger.level >= logger.Warn {
		zapGormLogger.statementLogger(ctx).Warn(fmt.Sprintf(message, arguments...))
	}
}

func (zapGormLogger *ZapGormLogger) Error(ctx context.Context, message string, arguments ...any) {
	if zapGormLogger.level >= logger.Error {
		zapGormLogger.statementLogger(ctx).Error(fmt.Sprintf(message, arguments...))
	}
}

func (zapGormLogger *ZapGormLogger) Trace(ctx context.Context, begin time.Time, describeStatement func() (string, int64), err error) {
	if zapGormLogger.level <= logger.Silent {
		return
	}
	statementLogger := zapGormLogger.statementLogger(ctx)
	elapsed := time.Since(begin)

	var level zapcore.Level
	message := "sql"
	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && zapGormLogger.level >= logger.Error:
		level, message = zapcore.ErrorLevel, "sql error"
	case zapGormLogger.SlowThreshold > 0 && elapsed > zapGormLogger.SlowThreshold && zapGormLogger.level >= logger.Warn:
		level, message = zapcore.WarnLevel, "slow sql"
	case zapGormLogger.level >= logger.Info:
		level = zapcore.DebugLevel
	default:
		return
	}
	checkedEntry := statementLogger.Check(level, message)
	if checkedEntry == nil { // 级别未开启时不拼 SQL
		return
	}
	sql, rows := describeStatement()
	fields := []zap.Field{
		zap.String("sql", sql),
		zap.Int64("rows", rows),
		zap.Duration("elapsed", elapsed),
		zap.String("source", utils.FileWithLineNum()),
	}
	if level == zapcore.ErrorLevel {
		fields = append(fields, zap.Error(err))
	}
	checkedEntry.Write(fields...)

	// 额外一条纯文本 DEBUG，方便直接复制 SQL 排查。
	if statementLogger.Core().Enabled(zapcore.DebugLevel) {
		statementLogger.Debug(plainSQLMessage(ctx, sql))
	}
}

func plainSQLMessage(ctx context.Context, sql string) string {
	traceID := contextTraceID(ctx)
	if traceID == "" {
		traceID = missingTraceID
	}
	return fmt.Sprintf("trace id: %s, SQL: %s;", traceID, strings.TrimRight(strings.TrimSpace(sql), ";"))
}

// ParamsFilter 决定日志里 SQL 代入哪些参数值：保留真实值方便排查，
// 但把 bcrypt 哈希（$2a$ / $2b$ / $2y$ 开头）替换成 ***，避免密码哈希落盘。
// 只影响日志，不影响实际执行的 SQL。
func (zapGormLogger *ZapGormLogger) ParamsFilter(_ context.Context, sql string, parameters ...any) (string, []any) {
	redacted := make([]any, len(parameters))
	for index, parameter := range parameters {
		if isPasswordHash(parameter) {
			redacted[index] = redactedParameter
		} else {
			redacted[index] = parameter
		}
	}
	return sql, redacted
}

func isPasswordHash(parameter any) bool {
	var value string
	switch typed := parameter.(type) {
	case string:
		value = typed
	case *string:
		if typed == nil {
			return false
		}
		value = *typed
	default:
		return false
	}
	if len(value) != bcryptHashLength {
		return false
	}
	for _, prefix := range bcryptHashPrefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

// statementLogger 关闭 zap 自带的 caller（它总指向本文件），真实调用位置见 source 字段。
func (zapGormLogger *ZapGormLogger) statementLogger(ctx context.Context) *zap.Logger {
	return contextLogger(ctx).WithOptions(zap.WithCaller(false), zap.AddStacktrace(zapcore.FatalLevel))
}
