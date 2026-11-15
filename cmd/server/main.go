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
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"jkylin.com/kylin/v2/internal/bootstrap"
	"jkylin.com/kylin/v2/internal/config"
	"jkylin.com/kylin/v2/internal/middleware"
	"jkylin.com/kylin/v2/internal/module/role"
	"jkylin.com/kylin/v2/internal/module/user"
	"jkylin.com/kylin/v2/internal/router"
	"jkylin.com/kylin/v2/pkg/consts"
	"jkylin.com/kylin/v2/pkg/logger"
)

const (
	// readHeaderTimeout 是读取请求头的超时时间。
	readHeaderTimeout = 10 * time.Second
	// defaultListenAddress 是未配置监听地址时使用的地址。
	defaultListenAddress = ":http"
)

func run(configPath string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Load config
	configuration, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	// Logger
	if err := bootstrap.InitializeLogger(configuration.Log); err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}

	// Database
	database, closeDatabase, err := bootstrap.InitializePostgres(ctx, configuration.Postgres)
	if err != nil {
		if ctx.Err() != nil && errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("initialize PostgreSQL: %w", err)
	}
	if closeDatabase != nil {
		defer closeDatabase()
		logger.Info("PostgreSQL connected")
	}

	// Redis
	redisClient, err := bootstrap.InitializeRedis(ctx, configuration.Redis)
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

	logger.Sugared().Infof("Application runner mode: %s", configuration.App.Mode)
	gin.SetMode(configuration.App.Mode)
	engine := gin.New()
	// gin.Context.Value 回退到 Request.Context()，使 logger.FromContext(ginCtx) 同样可用
	engine.ContextWithFallback = true

	middleware.Initialize(engine, configuration)
	dependencies := router.Dependencies{
		Database:      database,
		Redis:         redisClient,
		Auth:          configuration.Auth,
		SkipAuth:      configuration.App.SkipAuth(),
		LocalUsername: os.Getenv(consts.KylinAdminUsername),
	}
	if dependencies.SkipAuth {
		logger.Warn("Authentication is SKIPPED: requests without Authorization act as the admin user",
			zap.String("run_mode", configuration.App.RunMode))
	}
	router.Initialize(engine, dependencies)
	if !dependencies.Enabled() {
		logger.Warn("Login and management APIs disabled: requires PostgreSQL, Redis and KYLIN_JWT_SECRET")
	} else {
		if err := ensureAdmin(ctx, database); err != nil {
			return fmt.Errorf("ensure initial admin: %w", err)
		}
		// 启动时把接口放行权限同步到 Redis，覆盖 migration 或手工修改数据库带来的变更。
		if err := router.NewServices(dependencies).ApiPermissions.SyncCache(ctx); err != nil {
			return fmt.Errorf("sync api permissions to Redis: %w", err)
		}
		logger.Info("API permissions synced to Redis")
	}

	server := &http.Server{
		Addr:              configuration.App.Address,
		Handler:           engine,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	address := server.Addr
	if address == "" {
		address = defaultListenAddress
	}
	if ctx.Err() != nil {
		return nil
	}
	var listenConfiguration net.ListenConfig
	listener, err := listenConfiguration.Listen(ctx, "tcp", address)
	if err != nil {
		if ctx.Err() != nil && errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("listen on %q: %w", address, err)
	}
	defer func() {
		_ = server.Close()
	}()
	logger.Info(fmt.Sprintf("Application HTTP server listening: %s", listener.Addr().String()),
		zap.String("name", configuration.App.Name),
		zap.String("addr", listener.Addr().String()),
		zap.String("mode", configuration.App.Mode),
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
	shutdownContext, cancel := context.WithTimeout(context.Background(), time.Duration(configuration.App.ShutdownTimeout)*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("graceful HTTP shutdown: %w", err)
	}
	if err := <-serveDone; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	logger.Info("HTTP server stopped")
	return nil
}

// ensureAdmin 按 KYLIN_ADMIN_USERNAME / KYLIN_ADMIN_PASSWORD 创建初始管理员；用户已存在时不修改。
func ensureAdmin(ctx context.Context, database *gorm.DB) error {
	username, password := os.Getenv(consts.KylinAdminUsername), os.Getenv(consts.KylinAdminPassword)
	if strings.TrimSpace(username) == "" || password == "" {
		return nil
	}
	userService := user.NewService(user.NewRepository(database), role.NewRepository(database))
	created, err := userService.EnsureAdmin(ctx, username, password)
	if err != nil {
		return err
	}
	if created {
		logger.Info("Initial admin user created", zap.String("username", username))
	}
	return nil
}

// @title           JadeKylin API
// @version         1.0
// @description     JadeKylin 服务接口文档
// @host            localhost:8080
// @BasePath        /api/v1
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
