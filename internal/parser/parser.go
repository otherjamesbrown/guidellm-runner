package parser

import (
	"encoding/json"
	"fmt"
	"os"
)

// GuideLLM v0.5.0 JSON output structures

// BenchmarkReport represents the top-level GuideLLM v0.5.0 JSON output
type BenchmarkReport struct {
	Metadata   Metadata    `json:"metadata"`
	Args       Args        `json:"args"`
	Benchmarks []Benchmark `json:"benchmarks"`
}

// Metadata contains version info
type Metadata struct {
	Version         int    `json:"version"`
	GuideLLMVersion string `json:"guidellm_version"`
}

// Args contains the benchmark arguments
type Args struct {
	Target string `json:"target"`
	Model  string `json:"model"`
}

// Benchmark represents a single benchmark run
type Benchmark struct {
	Type           string          `json:"type_"`
	Config         BenchmarkConfig `json:"config"`
	SchedulerState SchedulerState  `json:"scheduler_state"`
	Metrics        BenchmarkMetrics `json:"metrics"`
}

// BenchmarkConfig contains benchmark configuration
type BenchmarkConfig struct {
	ID    string `json:"id_"`
	RunID string `json:"run_id"`
}

// SchedulerState contains the request counts from guidellm's scheduler
type SchedulerState struct {
	CreatedRequests    int `json:"created_requests"`
	SuccessfulRequests int `json:"successful_requests"`
	ErroredRequests    int `json:"errored_requests"`
	CancelledRequests  int `json:"cancelled_requests"`
	ProcessedRequests  int `json:"processed_requests"`
}

// BenchmarkMetrics contains all benchmark metrics with status distributions
type BenchmarkMetrics struct {
	RequestTotals           StatusCounts       `json:"request_totals"`
	RequestsPerSecond       StatusDistribution `json:"requests_per_second"`
	RequestLatency          StatusDistribution `json:"request_latency"`
	PromptTokenCount        StatusDistribution `json:"prompt_token_count"`
	OutputTokenCount        StatusDistribution `json:"output_token_count"`
	TotalTokenCount         StatusDistribution `json:"total_token_count"`
	TimeToFirstTokenMS      StatusDistribution `json:"time_to_first_token_ms"`
	InterTokenLatencyMS     StatusDistribution `json:"inter_token_latency_ms"`
	OutputTokensPerSecond   StatusDistribution `json:"output_tokens_per_second"`
	TokensPerSecond         StatusDistribution `json:"tokens_per_second"`
}

// StatusCounts contains request counts by status
type StatusCounts struct {
	Successful int `json:"successful"`
	Errored    int `json:"errored"`
	Incomplete int `json:"incomplete"`
	Total      int `json:"total"`
}

// StatusDistribution contains distributions for each status type
type StatusDistribution struct {
	Successful  DistributionSummary `json:"successful"`
	Errored     DistributionSummary `json:"errored"`
	Incomplete  DistributionSummary `json:"incomplete"`
	Total       DistributionSummary `json:"total"`
}

// DistributionSummary contains statistical summary of a metric
type DistributionSummary struct {
	Mean        float64     `json:"mean"`
	Median      float64     `json:"median"`
	Mode        float64     `json:"mode"`
	Variance    float64     `json:"variance"`
	StdDev      float64     `json:"std_dev"`
	Min         float64     `json:"min"`
	Max         float64     `json:"max"`
	Count       int         `json:"count"`
	TotalSum    float64     `json:"total_sum"`
	Percentiles Percentiles `json:"percentiles"`
}

// Percentiles contains percentile values
type Percentiles struct {
	P001 float64 `json:"p001"`
	P01  float64 `json:"p01"`
	P05  float64 `json:"p05"`
	P10  float64 `json:"p10"`
	P25  float64 `json:"p25"`
	P50  float64 `json:"p50"`
	P75  float64 `json:"p75"`
	P90  float64 `json:"p90"`
	P95  float64 `json:"p95"`
	P99  float64 `json:"p99"`
	P999 float64 `json:"p999"`
}

// ParsedResults contains the extracted metrics ready for Prometheus
type ParsedResults struct {
	TotalRequests      int
	SuccessfulRequests int
	FailedRequests     int
	PromptTokens       int
	OutputTokens       int
	OutputTokensPerSec float64
	RequestsPerSec     float64

	// Individual latencies for histogram recording
	// Note: TTFT and ITL require streaming to be enabled
	TTFTValues []float64
	ITLValues  []float64
	E2EValues  []float64

	// Distribution stats (for fallback when individual values unavailable)
	E2EStats *DistributionSummary
}

// ParseFile reads and parses a GuideLLM JSON output file
func ParseFile(path string) (*ParsedResults, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading output file: %w", err)
	}

	return Parse(data)
}

// Parse parses GuideLLM JSON output bytes
func Parse(data []byte) (*ParsedResults, error) {
	var report BenchmarkReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	results := &ParsedResults{
		TTFTValues: make([]float64, 0),
		ITLValues:  make([]float64, 0),
		E2EValues:  make([]float64, 0),
	}

	for _, benchmark := range report.Benchmarks {
		// Extract request counts from scheduler_state
		results.TotalRequests += benchmark.SchedulerState.CreatedRequests
		results.SuccessfulRequests += benchmark.SchedulerState.SuccessfulRequests
		results.FailedRequests += benchmark.SchedulerState.ErroredRequests

		// Extract token counts from metrics
		if benchmark.Metrics.PromptTokenCount.Successful.Count > 0 {
			results.PromptTokens += int(benchmark.Metrics.PromptTokenCount.Successful.TotalSum)
		}
		if benchmark.Metrics.OutputTokenCount.Successful.Count > 0 {
			results.OutputTokens += int(benchmark.Metrics.OutputTokenCount.Successful.TotalSum)
		}

		// Extract throughput from metrics (use successful distribution mean)
		if benchmark.Metrics.OutputTokensPerSecond.Successful.Count > 0 {
			results.OutputTokensPerSec = benchmark.Metrics.OutputTokensPerSecond.Successful.Mean
		}
		if benchmark.Metrics.RequestsPerSecond.Successful.Count > 0 {
			results.RequestsPerSec = benchmark.Metrics.RequestsPerSecond.Successful.Mean
		}

		// Extract E2E latency (request_latency is in seconds)
		if benchmark.Metrics.RequestLatency.Successful.Count > 0 {
			stats := benchmark.Metrics.RequestLatency.Successful
			results.E2EStats = &stats

			// Generate individual E2E values from percentiles for histogram recording
			// We use the distribution to create representative samples
			results.E2EValues = generateValuesFromDistribution(&stats)
		}

		// Extract TTFT if available (requires streaming)
		// Note: time_to_first_token_ms is in milliseconds, convert to seconds
		if benchmark.Metrics.TimeToFirstTokenMS.Successful.Count > 0 &&
			benchmark.Metrics.TimeToFirstTokenMS.Successful.Mean > 0 {
			stats := benchmark.Metrics.TimeToFirstTokenMS.Successful
			for _, v := range generateValuesFromDistribution(&stats) {
				results.TTFTValues = append(results.TTFTValues, v/1000.0) // ms to seconds
			}
		}

		// Extract ITL if available (requires streaming)
		// Note: inter_token_latency_ms is in milliseconds, convert to seconds
		if benchmark.Metrics.InterTokenLatencyMS.Successful.Count > 0 &&
			benchmark.Metrics.InterTokenLatencyMS.Successful.Mean > 0 {
			stats := benchmark.Metrics.InterTokenLatencyMS.Successful
			for _, v := range generateValuesFromDistribution(&stats) {
				results.ITLValues = append(results.ITLValues, v/1000.0) // ms to seconds
			}
		}
	}

	return results, nil
}

// generateValuesFromDistribution creates representative values from a distribution summary
// for recording in Prometheus histograms. This approximates the distribution using percentiles.
func generateValuesFromDistribution(stats *DistributionSummary) []float64 {
	if stats == nil || stats.Count == 0 {
		return nil
	}

	// Generate synthetic observations based on percentiles
	// The number of observations approximates the percentile distribution
	values := make([]float64, 0, 100)

	// Add observations weighted by percentile ranges
	// p01 represents the bottom 1%, p05 represents 1-5%, etc.
	addWeightedValues := func(value float64, weight int) {
		for i := 0; i < weight; i++ {
			values = append(values, value)
		}
	}

	p := stats.Percentiles

	// Weight observations by percentile ranges (out of 100 samples)
	addWeightedValues(p.P01, 1)   // 0-1%
	addWeightedValues(p.P05, 4)   // 1-5%
	addWeightedValues(p.P10, 5)   // 5-10%
	addWeightedValues(p.P25, 15)  // 10-25%
	addWeightedValues(p.P50, 25)  // 25-50%
	addWeightedValues(p.P75, 25)  // 50-75%
	addWeightedValues(p.P90, 15)  // 75-90%
	addWeightedValues(p.P95, 5)   // 90-95%
	addWeightedValues(p.P99, 4)   // 95-99%
	addWeightedValues(p.P999, 1)  // 99-99.9%

	return values
}
