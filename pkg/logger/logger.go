/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 23:30:00 America/Los_Angeles
 *****************************************************************************/

package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Options struct {
	Level      string // debug | info | warn | error | dpanic | panic | fatal
	Format     string // console | json
	Output     string // stdout | file | both
	Dir        string // 日志目录
	Filename   string // 日志文件名
	MaxSize    int    // 单个文件最大尺寸（MB）
	MaxBackups int    // 最多保留的旧文件个数
	MaxAge     int    // 旧文件最多保留天数
	Compress   bool   // 是否 gzip 压缩旧文件
	Caller     bool   // 是否输出调用位置
	Stacktrace string // 达到该级别时输出堆栈，默认 error
}

type ctxKey struct{}

var (
	mu     sync.RWMutex
	global *zap.Logger
	sugar  *zap.SugaredLogger
	level  = zap.NewAtomicLevel()
)

func defaultStr(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func defaultInt(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func replace(l *zap.Logger) {
	mu.Lock()
	defer mu.Unlock()
	global = l
	sugar = l.Sugar()
	zap.ReplaceGlobals(l)
}

func init() {
	l, _ := zap.NewDevelopment()
	replace(l)
}

func newEncoder(format string, color bool) zapcore.Encoder {
	cfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout(time.DateTime + ".000"),
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	if strings.ToLower(format) == "json" {
		cfg.EncodeTime = zapcore.ISO8601TimeEncoder
		return zapcore.NewJSONEncoder(cfg)
	}
	if color {
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	return zapcore.NewConsoleEncoder(cfg)
}

// SetLevel 运行时动态调整日志级别。
func SetLevel(l string) error {
	lvl, err := zapcore.ParseLevel(l)
	if err != nil {
		return err
	}
	level.SetLevel(lvl)
	return nil
}

func L() *zap.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return global
}

func S() *zap.SugaredLogger {
	mu.RLock()
	defer mu.RUnlock()
	return sugar
}

func Sync() error {
	err := L().Sync()
	if err != nil && (strings.Contains(err.Error(), "invalid argument") ||
		strings.Contains(err.Error(), "inappropriate ioctl")) {
		return nil
	}
	return err
}

func NewContext(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

func FromContext(ctx context.Context) *zap.Logger {
	if ctx != nil {
		if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok && l != nil {
			return l
		}
	}
	return L()
}

// WithContext 是 FromContext 的别名，便于链式调用：logger.WithContext(ctx).Info(...)
func WithContext(ctx context.Context) *zap.Logger {
	return FromContext(ctx)
}

func Debug(msg string, fields ...zap.Field) {
	L().WithOptions(zap.AddCallerSkip(1)).Debug(msg, fields...)
}
func Info(msg string, fields ...zap.Field) {
	L().WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
}
func Warn(msg string, fields ...zap.Field) {
	L().WithOptions(zap.AddCallerSkip(1)).Warn(msg, fields...)
}
func Error(msg string, fields ...zap.Field) {
	L().WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
}
func Fatal(msg string, fields ...zap.Field) {
	L().WithOptions(zap.AddCallerSkip(1)).Fatal(msg, fields...)
}

func Init(opts Options) error {
	lvl, err := zapcore.ParseLevel(defaultStr(opts.Level, "info"))
	if err != nil {
		return fmt.Errorf("logger: invalid level %q: %w", opts.Level, err)
	}
	level.SetLevel(lvl)

	stackLvl, err := zapcore.ParseLevel(defaultStr(opts.Stacktrace, "error"))
	if err != nil {
		return fmt.Errorf("logger: invalid stacktrace level %q: %w", opts.Stacktrace, err)
	}

	var cores []zapcore.Core
	output := strings.ToLower(defaultStr(opts.Output, "stdout"))

	if output == "stdout" || output == "both" {
		// 控制台：console 格式下启用彩色级别
		enc := newEncoder(opts.Format, true)
		cores = append(cores, zapcore.NewCore(enc, zapcore.Lock(os.Stdout), level))
	}
	if output == "file" || output == "both" {
		dir := defaultStr(opts.Dir, "logs")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("logger: create log dir: %w", err)
		}
		writer := &lumberjack.Logger{
			Filename:   filepath.Join(dir, defaultStr(opts.Filename, "app.log")),
			MaxSize:    defaultInt(opts.MaxSize, 100),
			MaxBackups: defaultInt(opts.MaxBackups, 7),
			MaxAge:     defaultInt(opts.MaxAge, 30),
			Compress:   opts.Compress,
			LocalTime:  true,
		}
		// 文件：不使用彩色
		enc := newEncoder(opts.Format, false)
		cores = append(cores, zapcore.NewCore(enc, zapcore.AddSync(writer), level))
	}
	if len(cores) == 0 {
		return fmt.Errorf("logger: invalid output %q, want stdout|file|both", opts.Output)
	}

	zopts := []zap.Option{zap.AddStacktrace(stackLvl)}
	if opts.Caller {
		zopts = append(zopts, zap.AddCaller())
	}

	replace(zap.New(zapcore.NewTee(cores...), zopts...))
	return nil
}
