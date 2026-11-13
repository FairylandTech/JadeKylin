package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"jkylin.com/kylin/v2/internal/config"
)

func InitPostgres(ctx context.Context, cfg config.Postgres) (*pgxpool.Pool, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, errors.New("postgres: DSN is required when enabled")
	}
	if cfg.MaxConns <= 0 || cfg.MinConns < 0 || cfg.MinConns > cfg.MaxConns {
		return nil, errors.New("postgres: require max_conns > 0 and 0 <= min_conns <= max_conns")
	}
	if cfg.ConnectTimeout <= 0 || cfg.MaxConnLifetime <= 0 || cfg.MaxConnIdleTime <= 0 {
		return nil, errors.New("postgres: timeouts and connection lifetimes must be positive")
	}
	pc, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		// pgx 的解析错误可能包含完整 DSN，避免将凭据写入日志。
		return nil, errors.New("postgres: invalid DSN")
	}
	pc.MaxConns = cfg.MaxConns
	pc.MinConns = cfg.MinConns
	pc.ConnConfig.ConnectTimeout = time.Duration(cfg.ConnectTimeout) * time.Second
	pc.MaxConnLifetime = time.Duration(cfg.MaxConnLifetime) * time.Second
	pc.MaxConnIdleTime = time.Duration(cfg.MaxConnIdleTime) * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, pc.ConnConfig.ConnectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return pool, nil
}
