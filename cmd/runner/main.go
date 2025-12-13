package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yourorg/guidellm-runner/internal/config"
	"github.com/yourorg/guidellm-runner/internal/runner"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Setup logger
	var level slog.Level
	switch *logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Count targets
	totalTargets := 0
	for envName, env := range cfg.Environments {
		logger.Info("loaded environment", "name", envName, "targets", len(env.Targets))
		totalTargets += len(env.Targets)
	}
	logger.Info("configuration loaded",
		"environments", len(cfg.Environments),
		"total_targets", totalTargets,
		"prometheus_port", cfg.Prometheus.Port)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Info("received shutdown signal", "signal", sig)
		cancel()
	}()

	// Start Prometheus metrics server
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Prometheus.Port)
		logger.Info("starting prometheus metrics server", "addr", addr)
		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})
		if err := http.ListenAndServe(addr, nil); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server failed", "error", err)
		}
	}()

	// Start runner
	r := runner.New(cfg, logger)
	if err := r.Start(ctx); err != nil {
		logger.Error("runner failed", "error", err)
		os.Exit(1)
	}

	logger.Info("shutdown complete")
}
