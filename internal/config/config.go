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
)

// EnvConfigPath 指定配置文件路径的环境变量。
const EnvConfigPath = "KYLIN_CONFIG"

// DefaultConfigPath 默认配置文件路径。
const DefaultConfigPath = "configs/config.yaml"

type Config struct {
	App       App       `yaml:"app"`
	Log       Log       `yaml:"log"`
	RateLimit RateLimit `yaml:"rate_limit"`
	CORS      CORS      `yaml:"cors"`
}

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

// Default 返回默认配置。
func Default() *Config {
	return &Config{
		App: App{Name: "jade-kylin", Mode: "debug", Addr: ":8080", ShutdownTimeout: 10},
		Log: Log{
			Level: "info", Format: "console", Output: "stdout",
			Dir: "logs", Filename: "app.log",
			MaxSize: 100, MaxBackups: 7, MaxAge: 30, Caller: true,
		},
		RateLimit: RateLimit{Enabled: true, RPS: 20, Burst: 40},
		CORS:      CORS{Enabled: true, AllowOrigins: []string{"*"}},
	}
}

var current = Default()

// Get 返回当前生效的配置。
func Get() *Config { return current }

// Load 加载配置：默认值 <- 配置文件。
// path 为空时依次尝试环境变量 KYLIN_CONFIG 和 configs/config.yaml；
// 文件不存在时使用默认配置。
func Load(path string) (*Config, error) {
	if path == "" {
		path = os.Getenv(EnvConfigPath)
	}
	if path == "" {
		path = DefaultConfigPath
	}

	cfg := Default()
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

	current = cfg
	return cfg, nil
}
