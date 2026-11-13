package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"jkylin.com/kylin/v2/internal/config"
)

func InitPostgres(ctx context.Context, cfg config.Postgres) (*gorm.DB, func(), error) {
	if !cfg.Enabled {
		return nil, nil, nil
	}
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, nil, errors.New("postgres: DSN is required when enabled")
	}
	if cfg.MaxConns <= 0 || cfg.MinConns < 0 || cfg.MinConns > cfg.MaxConns {
		return nil, nil, errors.New("postgres: require max_conns > 0 and 0 <= min_conns <= max_conns")
	}
	if cfg.ConnectTimeout <= 0 || cfg.MaxConnLifetime <= 0 || cfg.MaxConnIdleTime <= 0 {
		return nil, nil, errors.New("postgres: timeouts and connection lifetimes must be positive")
	}
	pc, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		// pgx 的解析错误可能包含完整 DSN，避免将凭据写入日志。
		return nil, nil, errors.New("postgres: invalid DSN")
	}
	pc.MaxConns = cfg.MaxConns
	pc.MinConns = cfg.MinConns
	pc.ConnConfig.ConnectTimeout = time.Duration(cfg.ConnectTimeout) * time.Second
	pc.MaxConnLifetime = time.Duration(cfg.MaxConnLifetime) * time.Second
	pc.MaxConnIdleTime = time.Duration(cfg.MaxConnIdleTime) * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	sqlDB := stdlib.OpenDBFromPool(pool)
	closePostgres := func() {
		_ = sqlDB.Close()
		pool.Close()
	}
	pingCtx, cancel := context.WithTimeout(ctx, pc.ConnConfig.ConnectTimeout)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		closePostgres()
		return nil, nil, fmt.Errorf("postgres: ping: %w", err)
	}
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		DisableAutomaticPing: true,
		Logger: gormlogger.New(log.New(os.Stderr, "", log.LstdFlags), gormlogger.Config{
			SlowThreshold:        200 * time.Millisecond,
			LogLevel:             gormlogger.Warn,
			ParameterizedQueries: true,
		}),
	})
	if err != nil {
		closePostgres()
		return nil, nil, fmt.Errorf("postgres: initialize GORM: %w", err)
	}
	return db, closePostgres, nil
}
