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

// runBenchmark executes a single GuideLLM benchmark run (backwards compatible)
func (r *Runner) runBenchmark(ctx context.Context, envName string, target config.Target, logger *slog.Logger) {
	r.runBenchmarkWithResults(ctx, envName, target, logger)
}

// runBenchmarkWithResults executes a single GuideLLM benchmark run and returns results
func (r *Runner) runBenchmarkWithResults(ctx context.Context, envName string, target config.Target, logger *slog.Logger) *parser.ParsedResults {
	labels := metrics.Labels(envName, target.Name, target.Model)
	metrics.BenchmarkRunsTotal.With(labels).Inc()

	// Create temp directory for output
	tmpDir, err := os.MkdirTemp("", "guidellm-*")
	if err != nil {
		logger.Error("failed to create temp directory", "error", err)
		metrics.BenchmarkRunsFailed.With(labels).Inc()
		return nil
	}
	defer os.RemoveAll(tmpDir)

	outputFile := filepath.Join(tmpDir, "benchmarks.json")

	// Get API key - prefer target config, fall back to environment
	apiKey := target.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	// Build GuideLLM command with API key injected into headers
	// Note: guidellm does NOT read OPENAI_API_KEY from environment, so we
	// must inject it via --request-formatter-kwargs
	args := r.buildArgs(target, tmpDir, apiKey)
	logger.Debug("running guidellm", "args", args)

	cmd := exec.CommandContext(ctx, "guidellm", args...)

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("guidellm failed",
			"error", err,
			"output", string(output))
		metrics.BenchmarkRunsFailed.With(labels).Inc()
		return nil
	}

	logger.Debug("guidellm completed", "output_length", len(output))

	// Parse results
	results, err := parser.ParseFile(outputFile)
	if err != nil {
		logger.Error("failed to parse results", "error", err)
		metrics.BenchmarkRunsFailed.With(labels).Inc()
		return nil
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

	return results
}

// buildArgs constructs the GuideLLM CLI arguments
func (r *Runner) buildArgs(target config.Target, outputDir string, apiKey string) []string {
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
		"--request-type", target.GetRequestType(r.cfg.Defaults),
		// Use gpt2 processor to avoid needing model-specific tokenizers
		// (many models like mistral need sentencepiece which isn't installed)
		"--processor", "gpt2",
	}

	// Build request-formatter-kwargs with:
	// - stream: false (streaming causes 502 errors with vLLM)
	// - Authorization header (guidellm doesn't read OPENAI_API_KEY env var)
	if apiKey != "" {
		formatterKwargs := fmt.Sprintf(`{"stream": false, "extras": {"headers": {"Authorization": "Bearer %s"}}}`, apiKey)
		args = append(args, "--request-formatter-kwargs", formatterKwargs)
	} else {
		args = append(args, "--request-formatter-kwargs", `{"stream": false}`)
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

	// Latency histograms - observe individual values if available
	for _, v := range results.TTFTValues {
		metrics.TimeToFirstToken.With(labels).Observe(v)
	}
	for _, v := range results.ITLValues {
		metrics.InterTokenLatency.With(labels).Observe(v)
	}
	for _, v := range results.E2EValues {
		metrics.EndToEndLatency.With(labels).Observe(v)
	}

	// If no individual values but we have distribution stats, observe synthetic samples
	// based on the distribution percentiles to populate the histogram
	if len(results.TTFTValues) == 0 && results.TTFTStats != nil && results.TTFTStats.Count > 0 {
		observeFromDistribution(metrics.TimeToFirstToken.With(labels), results.TTFTStats, true)
	}
	if len(results.ITLValues) == 0 && results.ITLStats != nil && results.ITLStats.Count > 0 {
		observeFromDistribution(metrics.InterTokenLatency.With(labels), results.ITLStats, true)
	}
	if len(results.E2EValues) == 0 && results.E2EStats != nil && results.E2EStats.Count > 0 {
		// E2E latency is already in seconds, no conversion needed
		observeFromDistribution(metrics.EndToEndLatency.With(labels), results.E2EStats, false)
	}
}

// observeFromDistribution generates synthetic observations from distribution statistics
// to populate Prometheus histograms when individual request data is not available.
// If msToSec is true, converts millisecond values to seconds.
func observeFromDistribution(observer interface{ Observe(float64) }, stats *parser.DistributionSummary, msToSec bool) {
	if stats == nil || stats.Count == 0 {
		return
	}

	// Conversion factor: 1/1000 for ms->s, 1 for already in seconds
	conv := 1.0
	if msToSec {
		conv = 0.001
	}

	// Generate observations weighted by their percentile ranges to approximate the distribution
	// This provides a reasonable approximation for histogram bucket population
	percentilePoints := []struct {
		value  float64
		weight int // how many observations to emit for this percentile range
	}{
		{stats.Percentiles.P01, 1},  // 0-1%: 1% of data
		{stats.Percentiles.P05, 4},  // 1-5%: 4% of data
		{stats.Percentiles.P10, 5},  // 5-10%: 5% of data
		{stats.Percentiles.P25, 15}, // 10-25%: 15% of data
		{stats.Percentiles.P50, 25}, // 25-50%: 25% of data
		{stats.Percentiles.P75, 25}, // 50-75%: 25% of data
		{stats.Percentiles.P90, 15}, // 75-90%: 15% of data
		{stats.Percentiles.P95, 5},  // 90-95%: 5% of data
		{stats.Percentiles.P99, 4},  // 95-99%: 4% of data
		{stats.Percentiles.P999, 1}, // 99-99.9%: ~1% of data
	}

	// Scale weights to match actual count
	totalWeight := 0
	for _, p := range percentilePoints {
		totalWeight += p.weight
	}

	// Observe each percentile value weighted proportionally
	for _, p := range percentilePoints {
		if p.value <= 0 {
			continue
		}
		// Calculate how many observations for this percentile
		numObs := (p.weight * stats.Count) / totalWeight
		if numObs < 1 {
			numObs = 1
		}
		valueInSeconds := p.value * conv
		for i := 0; i < numObs; i++ {
			observer.Observe(valueInSeconds)
		}
	}
}
