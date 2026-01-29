package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/onelineai/hana-news-api/docs" // swagger docs

	"github.com/onelineai/hana-news-api/internal/config"
	"github.com/onelineai/hana-news-api/internal/db"
	"github.com/onelineai/hana-news-api/internal/handler"
	"github.com/onelineai/hana-news-api/internal/repository"
	"github.com/onelineai/hana-news-api/internal/scheduler"
	"github.com/onelineai/hana-news-api/internal/service"
)

// @title           Hana Securities News API
// @version         1.0
// @description     API server for Hana Securities translated news service.
// @description     Provides access to Japanese (Minkabu) and Chinese (Wind) news translated to Korean.

// @contact.name   OLA B2B Team
// @contact.email  support@onelineai.com

// @license.name  Proprietary
// @license.url   https://onelineai.com

// @host      localhost:8080
// @BasePath  /

// @schemes http https

func main() {
	// Setup logger
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to databases
	logger.Info("connecting to databases")
	database, err := db.New(ctx, cfg)
	if err != nil {
		logger.Error("failed to connect to databases", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	logger.Info("database connections established")

	// Initialize repositories
	silverRepo := repository.NewSilverRepository(database.Silver)
	goldRepo := repository.NewGoldRepository(database.Gold)

	// Initialize services
	batchService := service.NewBatchService(silverRepo, goldRepo, logger)
	newsService := service.NewNewsService(goldRepo)

	// Initialize scheduler
	sched, err := scheduler.New(batchService, cfg.Batch.Interval, logger)
	if err != nil {
		logger.Error("failed to create scheduler", "error", err)
		os.Exit(1)
	}

	// Start scheduler
	if err := sched.Start(ctx); err != nil {
		logger.Error("failed to start scheduler", "error", err)
		os.Exit(1)
	}

	// Initialize HTTP handler
	h := handler.New(newsService, database, logger)

	// Setup HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      h.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("starting HTTP server", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			cancel()
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutdown signal received")

	// Cancel context to signal all goroutines
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop scheduler (waits for running jobs to complete)
	if err := sched.Stop(); err != nil {
		logger.Error("scheduler shutdown error", "error", err)
	}

	// Shutdown HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	logger.Info("shutdown complete")
}
