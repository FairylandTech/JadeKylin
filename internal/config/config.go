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

	"github.com/goccy/go-yaml"
	"jkylin.com/kylin/v2/pkg/consts"
)

type App struct {
	Name            string `yaml:"name"`
	Mode            string `yaml:"mode"` // debug | release | test
	Addr            string `yaml:"addr"`
	ShutdownTimeout int    `yaml:"shutdown_timeout"` // 秒
}

type Log struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	Dir        string `yaml:"dir"`
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
	Enabled         bool   `yaml:"enabled"`
	DSN             string `yaml:"dsn"`
	MaxConns        int32  `yaml:"max_conns"`
	MinConns        int32  `yaml:"min_conns"`
	ConnectTimeout  int    `yaml:"connect_timeout"`
	MaxConnLifetime int    `yaml:"max_conn_lifetime"`
	MaxConnIdleTime int    `yaml:"max_conn_idle_time"`
}

type Redis struct {
	Enabled        bool   `yaml:"enabled"`
	URL            string `yaml:"url"`
	PoolSize       int    `yaml:"pool_size"`
	MinIdleConns   int    `yaml:"min_idle_conns"`
	ConnectTimeout int    `yaml:"connect_timeout"` // 秒，包含启动 Ping
	ReadTimeout    int    `yaml:"read_timeout"`    // 秒
	WriteTimeout   int    `yaml:"write_timeout"`   // 秒
}

type Config struct {
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
		Redis: Redis{URL: "redis://localhost:6379/0", PoolSize: 10, ConnectTimeout: 5, ReadTimeout: 3, WriteTimeout: 3},
		Postgres: Postgres{
			MaxConns:        10,
			MinConns:        0,
			ConnectTimeout:  5,
			MaxConnLifetime: 3600,
			MaxConnIdleTime: 300,
		},
		App: App{
			Name:            "jade-kylin",
			Mode:            "debug",
			Addr:            ":8080",
			ShutdownTimeout: 10,
		},
		Log: Log{
			Level:      "debug",
			Format:     "console",
			Output:     "stdout",
			Dir:        "logs",
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

// Get 返回当前生效的配置。
func Get() *Config { return current }

// Load 按默认值、配置文件、环境变量的顺序加载配置。
// path 为空时依次尝试环境变量 KYLIN_CONFIG 和 configs/config.yaml；
// 文件不存在时使用默认配置。
func Load(path string) (*Config, error) {
	if path == "" {
		path = os.Getenv(consts.EnvConfigPath)
	}
	if path == "" {
		path = consts.DefaultConfigPath
	}

	cfg := Default() // 每次加载都从默认配置开始
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		// 使用默认配置
	case err != nil:
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	default:
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("config: parse %s: %w", path, err)
		}
	}

	if appRunMode, ok := os.LookupEnv(consts.GinMode); ok {
		cfg.App.Mode = appRunMode
	}

	if databaseDSN, ok := os.LookupEnv(consts.KylinDatabaseDsn); ok {
		cfg.Postgres.DSN = databaseDSN
	}

	if redisUrl, ok := os.LookupEnv(consts.KylinRedisURL); ok {
		cfg.Redis.URL = redisUrl
	}

	current = cfg
	return cfg, nil
}
