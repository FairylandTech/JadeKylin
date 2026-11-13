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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"jkylin.com/kylin/v2/internal/bootstrap"
	"jkylin.com/kylin/v2/internal/config"
	"jkylin.com/kylin/v2/internal/middleware"
	"jkylin.com/kylin/v2/internal/router"
	"jkylin.com/kylin/v2/pkg/logger"
)

func main() {
	configPath := flag.String("config", "", "config file path (default: $KYLIN_CONFIG or configs/config.yaml)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := bootstrap.InitLogger(cfg.Log); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	gin.SetMode(cfg.App.Mode)
	engine := gin.New()
	// gin.Context.Value 回退到 Request.Context()，使 logger.FromContext(ginCtx) 同样可用
	engine.ContextWithFallback = true

	middleware.Init(engine, cfg)
	router.Init(engine)

	srv := &http.Server{
		Addr:              cfg.App.Addr,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("server starting", zap.String("name", cfg.App.Name), zap.String("addr", cfg.App.Addr), zap.String("mode", cfg.App.Mode))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server listen failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("shutting down server", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.App.ShutdownTimeout)*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
		return
	}
	logger.Info("server exited")
}
