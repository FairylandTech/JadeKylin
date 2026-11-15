/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 04:20:05 America/Los_Angeles
 *****************************************************************************/

package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
	"jkylin.com/kylin/v2/pkg/consts"
)

const (
	// minimumJWTSecretBytes 是 JWT 密钥的最小长度。
	minimumJWTSecretBytes = 32

	modeDebug   = "debug"
	modeRelease = "release"

	runModeLocal       = "local"
	runModeDevelopment = "dev"
)

type App struct {
	Name            string `yaml:"name"`
	Mode            string `yaml:"mode"`     // debug | release | test
	RunMode         string `yaml:"run_mode"` // 运行环境；local | dev 时跳过登录认证，以初始管理员身份访问
	Address         string `yaml:"addr"`
	ShutdownTimeout int    `yaml:"shutdown_timeout"` // 秒
}

// SkipAuth 返回是否跳过登录认证（仅 local / dev 运行环境）。
func (app App) SkipAuth() bool {
	switch strings.ToLower(strings.TrimSpace(app.RunMode)) {
	case runModeLocal, runModeDevelopment:
		return true
	}
	return false
}

type Log struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	Directory  string `yaml:"dir"`
	Filename   string `yaml:"filename"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
	Caller     bool   `yaml:"caller"`
	// 访问日志中跳过的路径（如健康检查）
	SkipPaths []string `yaml:"skip_paths"`
}

type RateLimit struct {
	Enabled bool    `yaml:"enabled"`
	RPS     float64 `yaml:"rps"`   // 每个客户端 IP 每秒允许的请求数
	Burst   int     `yaml:"burst"` // 突发容量
}

type CORS struct {
	Enabled      bool     `yaml:"enabled"`
	AllowOrigins []string `yaml:"allow_origins"`
}

type Postgres struct {
	Enabled               bool   `yaml:"enabled"`
	DSN                   string `yaml:"dsn"`
	MaxConnections        int32  `yaml:"max_conns"`
	MinConnections        int32  `yaml:"min_conns"`
	ConnectTimeout        int    `yaml:"connect_timeout"`
	MaxConnectionLifetime int    `yaml:"max_conn_lifetime"`
	MaxConnectionIdleTime int    `yaml:"max_conn_idle_time"`
}

type Redis struct {
	Enabled            bool   `yaml:"enabled"`
	URL                string `yaml:"url"`
	PoolSize           int    `yaml:"pool_size"`
	MinIdleConnections int    `yaml:"min_idle_conns"`
	ConnectTimeout     int    `yaml:"connect_timeout"` // 秒，包含启动 Ping
	ReadTimeout        int    `yaml:"read_timeout"`    // 秒
	WriteTimeout       int    `yaml:"write_timeout"`   // 秒
}

type Auth struct {
	JWTSecret   string `yaml:"jwt_secret"`   // 推荐通过 KYLIN_JWT_SECRET 设置，至少 32 字节
	Issuer      string `yaml:"issuer"`       // JWT iss
	TokenTTL    int    `yaml:"token_ttl"`    // 秒，JWT 有效期
	IdleTimeout int    `yaml:"idle_timeout"` // 秒，无请求超过该时长会话失效；有请求即续期
	MaxSessions int    `yaml:"max_sessions"` // 每个用户同时在线的设备数，0 表示不限制；超出时踢掉最早登录的设备
	CaptchaTTL  int    `yaml:"captcha_ttl"`  // 秒，验证码有效期
}

func (auth Auth) Validate() error {
	switch {
	case auth.JWTSecret != "" && len(auth.JWTSecret) < minimumJWTSecretBytes:
		return errors.New("auth: jwt_secret must be at least 32 bytes")
	case auth.TokenTTL <= 0 || auth.IdleTimeout <= 0 || auth.CaptchaTTL <= 0:
		return errors.New("auth: token_ttl, idle_timeout and captcha_ttl must be positive")
	case auth.IdleTimeout >= auth.TokenTTL:
		// 续签条件是 "剩余有效期 < idle_timeout"，idle_timeout 必须小于 token_ttl 才不会每次请求都续签。
		return errors.New("auth: idle_timeout must be less than token_ttl")
	case auth.MaxSessions < 0:
		return errors.New("auth: max_sessions must not be negative")
	}
	return nil
}

type Config struct {
	Auth      Auth      `yaml:"auth"`
	Redis     Redis     `yaml:"redis"`
	App       App       `yaml:"app"`
	Log       Log       `yaml:"log"`
	RateLimit RateLimit `yaml:"rate_limit"`
	CORS      CORS      `yaml:"cors"`
	Postgres  Postgres  `yaml:"postgres"`
}

// Default 返回默认配置。
func Default() *Config {
	return &Config{
		Auth: Auth{
			Issuer:      "jade-kylin",
			TokenTTL:    900,
			IdleTimeout: 600,
			MaxSessions: 1,
			CaptchaTTL:  120,
		},
		Redis: Redis{
			URL:            "redis://localhost:6379/0",
			PoolSize:       10,
			ConnectTimeout: 5,
			ReadTimeout:    3,
			WriteTimeout:   3,
		},
		Postgres: Postgres{
			MaxConnections:        10,
			MinConnections:        0,
			ConnectTimeout:        5,
			MaxConnectionLifetime: 3600,
			MaxConnectionIdleTime: 300,
		},
		App: App{
			Name:            "jade-kylin",
			Mode:            modeDebug,
			Address:         ":8080",
			ShutdownTimeout: 10,
		},
		Log: Log{
			Level:      "debug",
			Format:     "console",
			Output:     "stdout",
			Directory:  "logs",
			Filename:   "app.log",
			MaxSize:    100,
			MaxBackups: 7,
			MaxAge:     30,
			Caller:     true,
		},
		RateLimit: RateLimit{
			Enabled: true,
			RPS:     20,
			Burst:   40,
		},
		CORS: CORS{
			Enabled:      true,
			AllowOrigins: []string{"*"},
		},
	}
}

var current = Default()

// Get 返回最近一次 Load 成功加载的配置；未加载时为默认配置。
func Get() *Config {
	return current
}

// Load 读取配置文件并应用环境变量覆盖。path 为空时依次使用 $KYLIN_CONFIG、configs/config.yaml。
func Load(path string) (*Config, error) {
	if path == "" {
		path = os.Getenv(consts.EnvironmentConfigPath)
	}
	if path == "" {
		path = consts.DefaultConfigPath
	}

	configuration := Default() // 每次加载都从默认配置开始
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		// 使用默认配置
	case err != nil:
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	default:
		if err := yaml.Unmarshal(data, configuration); err != nil {
			return nil, fmt.Errorf("config: parse %s: %w", path, err)
		}
	}

	if ginMode, ok := os.LookupEnv(consts.GinMode); ok {
		configuration.App.Mode = ginMode
	}

	if databaseDSN, ok := os.LookupEnv(consts.KylinDatabaseDsn); ok {
		configuration.Postgres.DSN = databaseDSN
	}

	if redisURL, ok := os.LookupEnv(consts.KylinRedisURL); ok {
		configuration.Redis.URL = redisURL
	}

	if runMode, ok := os.LookupEnv(consts.RunMode); ok {
		configuration.App.RunMode = runMode
	}
	if configuration.App.SkipAuth() && configuration.App.Mode == modeRelease {
		// 防止把本地调试配置带到生产环境。
		return nil, fmt.Errorf("config: run_mode %q skips authentication and must not be used with app.mode release", configuration.App.RunMode)
	}

	if jwtSecret, ok := os.LookupEnv(consts.KylinJWTSecret); ok {
		configuration.Auth.JWTSecret = jwtSecret
	}
	if err := configuration.Auth.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	// 如果是 debug 模式，强制日志级别为 debug，避免在开发环境中漏掉日志。
	if configuration.App.Mode == modeDebug {
		configuration.Log.Level = "debug"
	}

	current = configuration
	return configuration, nil
}
