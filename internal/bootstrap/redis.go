/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-13 11:54:00 America/Los_Angeles
 *****************************************************************************/

package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"jkylin.com/kylin/v2/internal/config"
)

// InitializeRedis 创建 Redis 客户端并验证连通性；未启用时返回 nil。
func InitializeRedis(ctx context.Context, configuration config.Redis) (*redis.Client, error) {
	if !configuration.Enabled {
		return nil, nil
	}
	if configuration.PoolSize <= 0 || configuration.MinIdleConnections < 0 || configuration.MinIdleConnections > configuration.PoolSize {
		return nil, errors.New("redis: require pool_size > 0 and 0 <= min_idle_conns <= pool_size")
	}
	if configuration.ConnectTimeout <= 0 || configuration.ReadTimeout <= 0 || configuration.WriteTimeout <= 0 {
		return nil, errors.New("redis: timeouts must be positive")
	}
	options, err := redis.ParseURL(configuration.URL)
	if err != nil {
		// URL 解析错误可能包含凭据，不向日志返回原始错误。
		return nil, errors.New("redis: invalid URL")
	}
	options.PoolSize = configuration.PoolSize
	options.MinIdleConns = configuration.MinIdleConnections
	options.DialTimeout = time.Duration(configuration.ConnectTimeout) * time.Second
	options.ReadTimeout = time.Duration(configuration.ReadTimeout) * time.Second
	options.WriteTimeout = time.Duration(configuration.WriteTimeout) * time.Second
	options.ContextTimeoutEnabled = true
	client := redis.NewClient(options)
	pingContext, cancel := context.WithTimeout(ctx, options.DialTimeout)
	defer cancel()
	if err := client.Ping(pingContext).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: ping: %w", err)
	}
	return client, nil
}
