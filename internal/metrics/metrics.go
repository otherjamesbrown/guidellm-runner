package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Labels used for all metrics
	labels = []string{"environment", "target", "model"}

	// Request metrics
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "guidellm_requests_total",
			Help: "Total number of requests made to the LLM",
		},
		labels,
	)

	RequestsSuccessful = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "guidellm_requests_successful_total",
			Help: "Total number of successful requests",
		},
		labels,
	)

	RequestsFailed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "guidellm_requests_failed_total",
			Help: "Total number of failed requests",
		},
		labels,
	)

	// Latency metrics
	TimeToFirstToken = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "guidellm_ttft_seconds",
			Help:    "Time to first token in seconds",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		labels,
	)

	InterTokenLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "guidellm_itl_seconds",
			Help:    "Inter-token latency in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
		labels,
	)

	EndToEndLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "guidellm_e2e_latency_seconds",
			Help:    "End-to-end request latency in seconds",
			Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 25, 50, 100},
		},
		labels,
	)

	// Throughput metrics
	OutputTokensPerSecond = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "guidellm_output_tokens_per_second",
			Help: "Output tokens generated per second",
		},
		labels,
	)

	RequestsPerSecond = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "guidellm_requests_per_second",
			Help: "Requests completed per second",
		},
		labels,
	)

	// Token metrics
	PromptTokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "guidellm_prompt_tokens_total",
			Help: "Total prompt tokens sent",
		},
		labels,
	)

	OutputTokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "guidellm_output_tokens_total",
			Help: "Total output tokens received",
		},
		labels,
	)

	// Benchmark run metrics
	BenchmarkRunsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "guidellm_benchmark_runs_total",
			Help: "Total number of benchmark runs",
		},
		labels,
	)

	BenchmarkRunsFailed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "guidellm_benchmark_runs_failed_total",
			Help: "Total number of failed benchmark runs",
		},
		labels,
	)

	LastBenchmarkTimestamp = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "guidellm_last_benchmark_timestamp",
			Help: "Unix timestamp of last successful benchmark",
		},
		labels,
	)

	// Runner status
	RunnerUp = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "guidellm_runner_up",
			Help: "Whether the runner is active for this target (1 = up, 0 = down)",
		},
		labels,
	)
)

// Labels returns a prometheus.Labels map for the given parameters
func Labels(environment, target, model string) prometheus.Labels {
	return prometheus.Labels{
		"environment": environment,
		"target":      target,
		"model":       model,
	}
}
