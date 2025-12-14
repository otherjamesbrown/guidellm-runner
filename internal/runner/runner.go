package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/yourorg/guidellm-runner/internal/config"
	"github.com/yourorg/guidellm-runner/internal/metrics"
	"github.com/yourorg/guidellm-runner/internal/parser"
)

// Runner manages GuideLLM benchmark runs across all configured targets
type Runner struct {
	cfg    *config.Config
	logger *slog.Logger
	wg     sync.WaitGroup
}

// New creates a new Runner
func New(cfg *config.Config, logger *slog.Logger) *Runner {
	return &Runner{
		cfg:    cfg,
		logger: logger,
	}
}

// Start begins running benchmarks for all environments and targets
func (r *Runner) Start(ctx context.Context) error {
	r.logger.Info("starting guidellm-runner",
		"environments", len(r.cfg.Environments),
		"interval", r.cfg.GetInterval())

	// Start a goroutine for each environment/target combination
	for envName, env := range r.cfg.Environments {
		for _, target := range env.Targets {
			r.wg.Add(1)
			go r.runTargetLoop(ctx, envName, target)
		}
	}

	// Wait for context cancellation
	<-ctx.Done()
	r.logger.Info("shutting down, waiting for benchmark runs to complete")
	r.wg.Wait()
	return nil
}

// runTargetLoop continuously runs benchmarks for a single target
func (r *Runner) runTargetLoop(ctx context.Context, envName string, target config.Target) {
	defer r.wg.Done()

	labels := metrics.Labels(envName, target.Name, target.Model)
	metrics.RunnerUp.With(labels).Set(1)
	defer metrics.RunnerUp.With(labels).Set(0)

	logger := r.logger.With(
		"environment", envName,
		"target", target.Name,
		"model", target.Model,
	)

	logger.Info("starting benchmark loop",
		"url", target.URL,
		"profile", target.GetProfile(r.cfg.Defaults),
		"rate", target.GetRate(r.cfg.Defaults))

	ticker := time.NewTicker(r.cfg.GetInterval())
	defer ticker.Stop()

	// Run immediately, then on interval
	r.runBenchmark(ctx, envName, target, logger)

	for {
		select {
		case <-ctx.Done():
			logger.Info("stopping benchmark loop")
			return
		case <-ticker.C:
			r.runBenchmark(ctx, envName, target, logger)
		}
	}
}

// runBenchmark executes a single GuideLLM benchmark run
func (r *Runner) runBenchmark(ctx context.Context, envName string, target config.Target, logger *slog.Logger) {
	labels := metrics.Labels(envName, target.Name, target.Model)
	metrics.BenchmarkRunsTotal.With(labels).Inc()

	// Create temp directory for output
	tmpDir, err := os.MkdirTemp("", "guidellm-*")
	if err != nil {
		logger.Error("failed to create temp directory", "error", err)
		metrics.BenchmarkRunsFailed.With(labels).Inc()
		return
	}
	defer os.RemoveAll(tmpDir)

	outputFile := filepath.Join(tmpDir, "benchmarks.json")

	// Build GuideLLM command
	args := r.buildArgs(target, tmpDir)
	logger.Debug("running guidellm", "args", args)

	cmd := exec.CommandContext(ctx, "guidellm", args...)

	// Set API key if configured
	if target.APIKey != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("OPENAI_API_KEY=%s", target.APIKey))
	}

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("guidellm failed",
			"error", err,
			"output", string(output))
		metrics.BenchmarkRunsFailed.With(labels).Inc()
		return
	}

	logger.Debug("guidellm completed", "output_length", len(output))

	// Parse results
	results, err := parser.ParseFile(outputFile)
	if err != nil {
		logger.Error("failed to parse results", "error", err)
		metrics.BenchmarkRunsFailed.With(labels).Inc()
		return
	}

	// Update Prometheus metrics
	r.updateMetrics(labels, results)
	metrics.LastBenchmarkTimestamp.With(labels).SetToCurrentTime()

	// Log at appropriate level based on results
	if results.TotalRequests == 0 {
		// Zero requests indicates a silent failure - likely validation or connection issue
		logger.Error("benchmark completed with ZERO requests - possible validation failure",
			"requests", results.TotalRequests,
			"successful", results.SuccessfulRequests,
			"failed", results.FailedRequests,
			"url", target.URL,
			"model", target.Model,
			"hint", "Check if the target URL is reachable and authentication is configured correctly")
		metrics.BenchmarkRunsFailed.With(labels).Inc()
	} else if results.FailedRequests > 0 && results.SuccessfulRequests == 0 {
		// All requests failed
		logger.Error("benchmark completed with all requests failed",
			"requests", results.TotalRequests,
			"successful", results.SuccessfulRequests,
			"failed", results.FailedRequests,
			"tokens_per_sec", results.OutputTokensPerSec)
	} else {
		logger.Info("benchmark completed",
			"requests", results.TotalRequests,
			"successful", results.SuccessfulRequests,
			"failed", results.FailedRequests,
			"tokens_per_sec", results.OutputTokensPerSec)
	}
}

// buildArgs constructs the GuideLLM CLI arguments
func (r *Runner) buildArgs(target config.Target, outputDir string) []string {
	args := []string{
		"benchmark",
		"--target", target.URL,
		"--model", target.Model,
		"--profile", target.GetProfile(r.cfg.Defaults),
		"--rate", fmt.Sprintf("%d", target.GetRate(r.cfg.Defaults)),
		"--max-seconds", fmt.Sprintf("%d", target.GetMaxSeconds(r.cfg.Defaults)),
		"--data", r.cfg.Defaults.DataSpec,
		"--output-dir", outputDir,
		"--outputs", "json",
		"--backend-kwargs", `{"validate_backend": false}`,
	}

	return args
}

// updateMetrics updates Prometheus metrics from parsed results
func (r *Runner) updateMetrics(labels map[string]string, results *parser.ParsedResults) {
	// Request counters
	metrics.RequestsTotal.With(labels).Add(float64(results.TotalRequests))
	metrics.RequestsSuccessful.With(labels).Add(float64(results.SuccessfulRequests))
	metrics.RequestsFailed.With(labels).Add(float64(results.FailedRequests))

	// Token counters
	metrics.PromptTokensTotal.With(labels).Add(float64(results.PromptTokens))
	metrics.OutputTokensTotal.With(labels).Add(float64(results.OutputTokens))

	// Throughput gauges
	metrics.OutputTokensPerSecond.With(labels).Set(results.OutputTokensPerSec)
	metrics.RequestsPerSecond.With(labels).Set(results.RequestsPerSec)

	// Latency histograms
	for _, v := range results.TTFTValues {
		metrics.TimeToFirstToken.With(labels).Observe(v)
	}
	for _, v := range results.ITLValues {
		metrics.InterTokenLatency.With(labels).Observe(v)
	}
	for _, v := range results.E2EValues {
		metrics.EndToEndLatency.With(labels).Observe(v)
	}
}
