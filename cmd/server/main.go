/*******************************************************************************
 * Copyright (c) 2026 Beau Dean. All rights reserved.
 ******************************************************************************/

/******************************************************************************
 * @software: GoLand
 * @author: Beau Dean (Hean)
 * @contact: https://github.com/FairylandTech
 * @organization: https://github.com/FairylandFuture
 * @datetime: 2026-11-11 03:53:12 America/Los_Angeles
 *****************************************************************************/

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"jkylin.com/kylin/v2/internal/bootstrap"
	"jkylin.com/kylin/v2/internal/config"
	"jkylin.com/kylin/v2/internal/middleware"
	"jkylin.com/kylin/v2/internal/router"
	"jkylin.com/kylin/v2/pkg/consts"
	"jkylin.com/kylin/v2/pkg/logger"
)

func main() {
	_ = godotenv.Load()

	configPath := flag.String("config", "", "config file path (default: $KYLIN_CONFIG or configs/config.yaml)")
	flag.Parse()

	err := run(*configPath)
	if err != nil {
		logger.Error("Application failed", zap.Error(err))
	}
	_ = logger.Sync()
	if err != nil {
		os.Exit(1)
	}
}

func run(configPath string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	// Logger
	if err := bootstrap.InitLogger(cfg.Log); err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}

	// Database
	postgresDB, closePostgres, err := bootstrap.InitPostgres(ctx, cfg.Postgres)
	if err != nil {
		if ctx.Err() != nil && errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("initialize PostgreSQL: %w", err)
	}
	if closePostgres != nil {
		defer closePostgres()
		logger.Info("PostgreSQL connected")
	}

	// Redis
	redisClient, err := bootstrap.InitRedis(ctx, cfg.Redis)
	if err != nil {
		if ctx.Err() != nil && errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("initialize Redis: %w", err)
	}
	if redisClient != nil {
		defer func() {
			if err := redisClient.Close(); err != nil {
				logger.Error("Redis close failed", zap.Error(err))
			}
		}()
		logger.Info("Redis connected")
	}

	logger.S().Infof("Application runner mode: %s", cfg.App.Mode)
	gin.SetMode(cfg.App.Mode)
	engine := gin.New()
	// gin.Context.Value 回退到 Request.Context()，使 logger.FromContext(ginCtx) 同样可用
	engine.ContextWithFallback = true

	middleware.Init(engine, cfg)
	router.Init(engine, postgresDB, os.Getenv(consts.KylinUserAdminToken))

	server := &http.Server{
		Addr:              cfg.App.Addr,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	addr := server.Addr
	if addr == "" {
		addr = ":http"
	}
	if ctx.Err() != nil {
		return nil
	}
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", addr)
	if err != nil {
		if ctx.Err() != nil && errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("listen on %q: %w", addr, err)
	}
	defer func() { _ = server.Close() }()
	logger.Info(fmt.Sprintf("Application HTTP server listening: %s", listener.Addr().String()),
		zap.String("name", cfg.App.Name),
		zap.String("addr", listener.Addr().String()),
		zap.String("mode", cfg.App.Mode),
	)

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
	}()

	select {
	case err := <-serveDone:
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		stop()
	}

	logger.Info("Shutting down HTTP server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.App.ShutdownTimeout)*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful HTTP shutdown: %w", err)
	}
	if err := <-serveDone; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	logger.Info("HTTP server stopped")
	return nil
}
