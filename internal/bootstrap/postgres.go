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
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"jkylin.com/kylin/v2/internal/config"
)

// InitializePostgres 创建连接池并返回 GORM 实例和关闭函数；未启用时全部返回 nil。
func InitializePostgres(ctx context.Context, configuration config.Postgres) (*gorm.DB, func(), error) {
	if !configuration.Enabled {
		return nil, nil, nil
	}
	if strings.TrimSpace(configuration.DSN) == "" {
		return nil, nil, errors.New("postgres: DSN is required when enabled")
	}
	if configuration.MaxConnections <= 0 || configuration.MinConnections < 0 || configuration.MinConnections > configuration.MaxConnections {
		return nil, nil, errors.New("postgres: require max_conns > 0 and 0 <= min_conns <= max_conns")
	}
	if configuration.ConnectTimeout <= 0 || configuration.MaxConnectionLifetime <= 0 || configuration.MaxConnectionIdleTime <= 0 {
		return nil, nil, errors.New("postgres: timeouts and connection lifetimes must be positive")
	}
	poolConfiguration, err := pgxpool.ParseConfig(configuration.DSN)
	if err != nil {
		// pgx 的解析错误可能包含完整 DSN，避免将凭据写入日志。
		return nil, nil, errors.New("postgres: invalid DSN")
	}
	poolConfiguration.MaxConns = configuration.MaxConnections
	poolConfiguration.MinConns = configuration.MinConnections
	poolConfiguration.ConnConfig.ConnectTimeout = time.Duration(configuration.ConnectTimeout) * time.Second
	poolConfiguration.MaxConnLifetime = time.Duration(configuration.MaxConnectionLifetime) * time.Second
	poolConfiguration.MaxConnIdleTime = time.Duration(configuration.MaxConnectionIdleTime) * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, poolConfiguration)
	if err != nil {
		return nil, nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	sqlDatabase := stdlib.OpenDBFromPool(pool)
	closePostgres := func() {
		_ = sqlDatabase.Close()
		pool.Close()
	}
	pingContext, cancel := context.WithTimeout(ctx, poolConfiguration.ConnConfig.ConnectTimeout)
	defer cancel()
	if err := sqlDatabase.PingContext(pingContext); err != nil {
		closePostgres()
		return nil, nil, fmt.Errorf("postgres: ping: %w", err)
	}
	database, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDatabase}), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               NewZapGormLogger(slowQueryThreshold),
	})
	if err != nil {
		closePostgres()
		return nil, nil, fmt.Errorf("postgres: initialize GORM: %w", err)
	}
	return database, closePostgres, nil
}
