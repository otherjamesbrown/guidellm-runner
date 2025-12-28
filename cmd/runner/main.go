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
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yourorg/guidellm-runner/internal/api"
	"github.com/yourorg/guidellm-runner/internal/config"
	"github.com/yourorg/guidellm-runner/internal/runner"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	apiPort := flag.Int("api-port", 8080, "Port for the runtime control API")
	autoStart := flag.Bool("auto-start", true, "Automatically start configured targets on startup")
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
		"prometheus_port", cfg.Prometheus.Port,
		"api_port", *apiPort)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create target manager
	manager := runner.NewTargetManager(cfg, logger)

	// Create runner with manager reference
	r := runner.New(cfg, logger)
	manager.SetRunner(r)

	// Load targets from config
	manager.LoadFromConfig()

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

	// Start API server
	apiServer := api.NewServer(api.ServerConfig{
		Port:   *apiPort,
		Logger: logger,
	}, manager)

	go func() {
		if err := apiServer.Start(); err != nil {
			logger.Error("API server failed", "error", err)
		}
	}()

	// Auto-start configured targets if enabled
	if *autoStart && totalTargets > 0 {
		logger.Info("auto-starting configured targets", "count", totalTargets)
		manager.StartAllConfigured(ctx)
	}

	// Wait for shutdown signal
	sig := <-sigChan
	logger.Info("received shutdown signal", "signal", sig)
	cancel()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop API server
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("API server shutdown failed", "error", err)
	}

	// Stop all targets
	manager.StopAll()

	// Wait for all benchmark runs to complete
	logger.Info("waiting for benchmark runs to complete")
	manager.Wait()

	logger.Info("shutdown complete")
}
