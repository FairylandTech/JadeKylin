package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"jkylin.com/kylin/v2/internal/config"
)

func InitRedis(ctx context.Context, cfg config.Redis) (*redis.Client, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if cfg.PoolSize <= 0 || cfg.MinIdleConns < 0 || cfg.MinIdleConns > cfg.PoolSize {
		return nil, errors.New("redis: require pool_size > 0 and 0 <= min_idle_conns <= pool_size")
	}
	if cfg.ConnectTimeout <= 0 || cfg.ReadTimeout <= 0 || cfg.WriteTimeout <= 0 {
		return nil, errors.New("redis: timeouts must be positive")
	}
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		// URL 解析错误可能包含凭据，不向日志返回原始错误。
		return nil, errors.New("redis: invalid URL")
	}
	opts.PoolSize = cfg.PoolSize
	opts.MinIdleConns = cfg.MinIdleConns
	opts.DialTimeout = time.Duration(cfg.ConnectTimeout) * time.Second
	opts.ReadTimeout = time.Duration(cfg.ReadTimeout) * time.Second
	opts.WriteTimeout = time.Duration(cfg.WriteTimeout) * time.Second
	opts.ContextTimeoutEnabled = true
	client := redis.NewClient(opts)
	pingCtx, cancel := context.WithTimeout(ctx, opts.DialTimeout)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: ping: %w", err)
	}
	return client, nil
}
