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

	"github.com/tanishqua/hermes/api"
	"github.com/tanishqua/hermes/config"
	"github.com/tanishqua/hermes/queue"
	"github.com/tanishqua/hermes/store"
	"github.com/tanishqua/hermes/worker"
)

func main() {
	// Structured logger
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Connect to Postgres
	slog.Info("connecting to postgres...")
	s, err := store.New(ctx, cfg.PostgresDSN)
	if err != nil {
		slog.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer s.Close()
	slog.Info("postgres connected")

	// Connect to Redis
	slog.Info("connecting to redis...")
	q, err := queue.New(cfg.RedisAddr)
	if err != nil {
		slog.Error("redis connect failed", "err", err)
		os.Exit(1)
	}
	defer q.Close()
	slog.Info("redis connected")

	// Start worker pool in a goroutine
	pool := worker.NewPool(cfg.WorkerConcurrency, q, s)
	go pool.Start()

	// HTTP server
	router := api.NewRouter(s, q)
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.APIPort),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("API server listening", "port", cfg.APIPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutdown signal received")

	pool.Stop()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("server shutdown error", "err", err)
	}
	slog.Info("shutdown complete")
}
