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

// 未配置时使用的默认值。
const (
	defaultLevel           = "info"
	defaultStacktraceLevel = "error"
	defaultOutput          = "stdout"
	defaultDirectory       = "logs"
	defaultFilename        = "app.log"
	defaultMaxSize         = 100 // 日志文件最大 100 MB
	defaultMaxBackups      = 7   // 最多保留 7 个旧文件
	defaultMaxAge          = 30  // 最多保留 30 天
)

// 日志输出位置。
const (
	outputStdout = "stdout"
	outputFile   = "file"
	outputBoth   = "both"
)

type Options struct {
	Level      string // debug | info | warn | error | dpanic | panic | fatal
	Format     string // console | json
	Output     string // stdout | file | both
	Directory  string // 日志目录
	Filename   string // 日志文件名
	MaxSize    int    // 单个文件最大尺寸（MB）
	MaxBackups int    // 最多保留的旧文件个数
	MaxAge     int    // 旧文件最多保留天数
	Compress   bool   // 是否 gzip 压缩旧文件
	Caller     bool   // 是否输出调用位置
	Stacktrace string // 达到该级别时输出堆栈，默认 error
}

type loggerContextKey struct{}

type traceIDContextKey struct{}

var (
	mutex         sync.RWMutex           // 保护全局 logger 的读写锁
	globalLogger  *zap.Logger            // 全局 logger
	sugaredLogger *zap.SugaredLogger     // 全局 SugaredLogger
	atomicLevel   = zap.NewAtomicLevel() // 全局日志级别
)

// defaultString 返回 value 或默认值 fallback（value 为空字符串时返回 fallback）。
func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// defaultInteger 返回 value 或默认值 fallback（value 小于等于 0 时返回 fallback）。
func defaultInteger(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

// replace 替换全局 logger。
func replace(replacement *zap.Logger) {
	mutex.Lock()
	defer mutex.Unlock()
	globalLogger = replacement
	sugaredLogger = replacement.Sugar()
	zap.ReplaceGlobals(replacement)
}

// newEncoder 创建日志编码器。
func newEncoder(format string, color bool) zapcore.Encoder {
	encoderConfiguration := zapcore.EncoderConfig{
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
		encoderConfiguration.EncodeTime = zapcore.ISO8601TimeEncoder
		return zapcore.NewJSONEncoder(encoderConfiguration)
	}
	if color {
		encoderConfiguration.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	return zapcore.NewConsoleEncoder(encoderConfiguration)
}

// SetLevel 设置全局日志级别。
func SetLevel(levelName string) error {
	level, err := zapcore.ParseLevel(levelName)
	if err != nil {
		return err
	}
	atomicLevel.SetLevel(level)
	return nil
}

// Global 返回全局 logger。
func Global() *zap.Logger {
	mutex.RLock()
	defer mutex.RUnlock()
	return globalLogger
}

// Sugared 返回全局 SugaredLogger。
func Sugared() *zap.SugaredLogger {
	mutex.RLock()
	defer mutex.RUnlock()
	return sugaredLogger
}

// Sync 刷新缓冲区，确保日志写入磁盘。
func Sync() error {
	err := Global().Sync()
	if err != nil && (strings.Contains(err.Error(), "invalid argument") ||
		strings.Contains(err.Error(), "inappropriate ioctl")) {
		return nil
	}
	return err
}

// NewContext 返回附带 logger 的新 context。
func NewContext(ctx context.Context, contextLogger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, contextLogger)
}

// WithTraceID 返回附带 trace id 的新 context。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDContextKey{}, traceID)
}

// TraceID 从 context 中取出 trace id；不存在时返回空字符串。
func TraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	traceID, _ := ctx.Value(traceIDContextKey{}).(string)
	return traceID
}

// FromContext 从 context 中获取 logger；若不存在则返回全局 logger。
func FromContext(ctx context.Context) *zap.Logger {
	if ctx != nil {
		if contextLogger, ok := ctx.Value(loggerContextKey{}).(*zap.Logger); ok && contextLogger != nil {
			return contextLogger
		}
	}
	return Global()
}

func Debug(message string, fields ...zap.Field) {
	Global().WithOptions(zap.AddCallerSkip(1)).Debug(message, fields...)
}

func Info(message string, fields ...zap.Field) {
	Global().WithOptions(zap.AddCallerSkip(1)).Info(message, fields...)
}

func Warn(message string, fields ...zap.Field) {
	Global().WithOptions(zap.AddCallerSkip(1)).Warn(message, fields...)
}

func Error(message string, fields ...zap.Field) {
	Global().WithOptions(zap.AddCallerSkip(1)).Error(message, fields...)
}

func Fatal(message string, fields ...zap.Field) {
	Global().WithOptions(zap.AddCallerSkip(1)).Fatal(message, fields...)
}

func init() {
	developmentLogger, _ := zap.NewDevelopment()
	replace(developmentLogger)
}

// Initialize 按配置创建并替换全局 logger。
func Initialize(options Options) error {
	level, err := zapcore.ParseLevel(defaultString(options.Level, defaultLevel)) // 设置默认日志级别为 info
	if err != nil {
		return fmt.Errorf("logger: invalid level %q: %w", options.Level, err)
	}
	atomicLevel.SetLevel(level)

	stacktraceLevel, err := zapcore.ParseLevel(defaultString(options.Stacktrace, defaultStacktraceLevel)) // 设置默认堆栈输出级别为 error
	if err != nil {
		return fmt.Errorf("logger: invalid stacktrace level %q: %w", options.Stacktrace, err)
	}

	var cores []zapcore.Core
	output := strings.ToLower(defaultString(options.Output, defaultOutput)) // 设置默认输出为 stdout

	if output == outputStdout || output == outputBoth {
		// 控制台：console 格式下启用彩色级别
		encoder := newEncoder(options.Format, true)
		cores = append(cores, zapcore.NewCore(encoder, zapcore.Lock(os.Stdout), atomicLevel))
	}
	if output == outputFile || output == outputBoth {
		directory := defaultString(options.Directory, defaultDirectory)
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return fmt.Errorf("logger: create log dir: %w", err)
		}
		writer := &lumberjack.Logger{
			Filename:   filepath.Join(directory, defaultString(options.Filename, defaultFilename)),
			MaxSize:    defaultInteger(options.MaxSize, defaultMaxSize),
			MaxBackups: defaultInteger(options.MaxBackups, defaultMaxBackups),
			MaxAge:     defaultInteger(options.MaxAge, defaultMaxAge),
			Compress:   options.Compress,
			LocalTime:  true,
		}
		// 文件：不使用彩色
		encoder := newEncoder(options.Format, false)
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(writer), atomicLevel))
	}
	if len(cores) == 0 {
		return fmt.Errorf("logger: invalid output %q, want stdout|file|both", options.Output)
	}

	zapOptions := []zap.Option{zap.AddStacktrace(stacktraceLevel)} // 设置堆栈输出级别
	if options.Caller {
		// 设置调用位置输出
		zapOptions = append(zapOptions, zap.AddCaller())
	}

	replace(zap.New(zapcore.NewTee(cores...), zapOptions...)) // 替换全局 logger
	return nil
}
