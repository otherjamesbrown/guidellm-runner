package parser

import (
	"encoding/json"
	"fmt"
	"os"
)

// BenchmarkReport represents the top-level GuideLLM JSON output
type BenchmarkReport struct {
	Benchmarks []Benchmark `json:"benchmarks"`
}

// Benchmark represents a single benchmark run
type Benchmark struct {
	Type_          string             `json:"type_"`
	Profile        string             `json:"profile"`
	Rate           float64            `json:"rate"`
	Config         *BenchmarkConfig   `json:"config,omitempty"`
	Requests       RequestsData       `json:"requests"`
	Stats          *Stats             `json:"stats,omitempty"`
	Summary        *Summary           `json:"summary,omitempty"`
	SchedulerState *SchedulerState    `json:"scheduler_state,omitempty"`
	Metrics        *GenerativeMetrics `json:"metrics,omitempty"`
	StartTime      float64            `json:"start_time"`
	EndTime        float64            `json:"end_time"`
	Duration       float64            `json:"duration"`
	Completed      int                `json:"completed_requests"`
	Errored        int                `json:"errored_requests"`
}

// BenchmarkConfig contains benchmark configuration
type BenchmarkConfig struct {
	Profile string  `json:"profile"`
	Rate    float64 `json:"rate"`
}

// SchedulerState contains the request counts from guidellm's scheduler
type SchedulerState struct {
	CreatedRequests    int `json:"created_requests"`
	SuccessfulRequests int `json:"successful_requests"`
	ErroredRequests    int `json:"errored_requests"`
	CancelledRequests  int `json:"cancelled_requests"`
}

// RequestsData contains arrays of requests by status
type RequestsData struct {
	Successful []RequestStats `json:"successful"`
	Errored    []RequestStats `json:"errored"`
	Incomplete []RequestStats `json:"incomplete"`
}

// Percentiles contains standard percentile values
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

// DistributionSummary contains statistical summary of a distribution
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

// StatusDistributionSummary contains distribution summaries by request status
type StatusDistributionSummary struct {
	Successful *DistributionSummary `json:"successful"`
	Incomplete *DistributionSummary `json:"incomplete"`
	Errored    *DistributionSummary `json:"errored"`
	Total      *DistributionSummary `json:"total"`
}

// StatusBreakdownInt contains integer counts by status
type StatusBreakdownInt struct {
	Successful int `json:"successful"`
	Incomplete int `json:"incomplete"`
	Errored    int `json:"errored"`
	Total      int `json:"total"`
}

// GenerativeMetrics contains the v0.5.0 GuideLLM metrics format
type GenerativeMetrics struct {
	// Request stats
	RequestTotals      *StatusBreakdownInt        `json:"request_totals,omitempty"`
	RequestsPerSecond  *StatusDistributionSummary `json:"requests_per_second,omitempty"`
	RequestConcurrency *StatusDistributionSummary `json:"request_concurrency,omitempty"`
	RequestLatency     *StatusDistributionSummary `json:"request_latency,omitempty"`

	// Token stats
	PromptTokenCount  *StatusDistributionSummary `json:"prompt_token_count,omitempty"`
	OutputTokenCount  *StatusDistributionSummary `json:"output_token_count,omitempty"`
	TotalTokenCount   *StatusDistributionSummary `json:"total_token_count,omitempty"`
	TokensPerSecond   *StatusDistributionSummary `json:"tokens_per_second,omitempty"`
	OutputTokensPS    *StatusDistributionSummary `json:"output_tokens_per_second,omitempty"`
	PromptTokensPS    *StatusDistributionSummary `json:"prompt_tokens_per_second,omitempty"`

	// Latency metrics (in milliseconds)
	TimeToFirstTokenMs   *StatusDistributionSummary `json:"time_to_first_token_ms,omitempty"`
	TimePerOutputTokenMs *StatusDistributionSummary `json:"time_per_output_token_ms,omitempty"`
	InterTokenLatencyMs  *StatusDistributionSummary `json:"inter_token_latency_ms,omitempty"`

	// Legacy format support
	RequestThroughput *LegacyThroughputMetrics      `json:"request_throughput,omitempty"`
	TokenThroughput   *LegacyTokenThroughputMetrics `json:"token_throughput,omitempty"`
}

// LegacyThroughputMetrics contains legacy request throughput stats
type LegacyThroughputMetrics struct {
	Mean float64 `json:"mean"`
}

// LegacyTokenThroughputMetrics contains legacy token throughput stats
type LegacyTokenThroughputMetrics struct {
	OutputPerSecond *LegacyMetricStats `json:"output_per_second,omitempty"`
}

// LegacyMetricStats contains legacy statistical measurements
type LegacyMetricStats struct {
	Mean float64 `json:"mean"`
}

// RequestStats represents a single request's data (v0.5.0 format)
type RequestStats struct {
	Type_                string  `json:"type_"`
	RequestID            string  `json:"request_id"`
	RequestType          string  `json:"request_type"`
	RequestStartTime     float64 `json:"request_start_time"`
	RequestEndTime       float64 `json:"request_end_time"`
	RequestLatency       float64 `json:"request_latency"`
	PromptTokens         int     `json:"prompt_tokens"`
	InputTokens          int     `json:"input_tokens"`
	OutputTokens         int     `json:"output_tokens"`
	TotalTokens          int     `json:"total_tokens"`
	TimeToFirstTokenMs   float64 `json:"time_to_first_token_ms"`
	TimePerOutputTokenMs float64 `json:"time_per_output_token_ms"`
	InterTokenLatencyMs  float64 `json:"inter_token_latency_ms"`
	TokensPerSecond      float64 `json:"tokens_per_second"`
	OutputTokensPS       float64 `json:"output_tokens_per_second"`
}

// Stats contains aggregated statistics (legacy format)
type Stats struct {
	TTFT LatencyStats `json:"ttft"`
	ITL  LatencyStats `json:"itl"`
	E2E  LatencyStats `json:"e2e"`
}

// LatencyStats contains latency distribution statistics (legacy format)
type LatencyStats struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
	P50    float64 `json:"p50"`
	P90    float64 `json:"p90"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
	StdDev float64 `json:"std_dev"`
}

// Summary contains overall benchmark summary (legacy format)
type Summary struct {
	TotalRequests      int     `json:"total_requests"`
	SuccessfulRequests int     `json:"successful_requests"`
	FailedRequests     int     `json:"failed_requests"`
	TotalPromptTokens  int     `json:"total_prompt_tokens"`
	TotalOutputTokens  int     `json:"total_output_tokens"`
	OutputTokensPerSec float64 `json:"output_tokens_per_second"`
	RequestsPerSec     float64 `json:"requests_per_second"`
	Duration           float64 `json:"duration_seconds"`
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

	// Individual latencies for histogram recording (in seconds)
	TTFTValues []float64
	ITLValues  []float64
	E2EValues  []float64

	// Distribution statistics for metrics (from aggregated data)
	TTFTStats *DistributionSummary
	ITLStats  *DistributionSummary
	E2EStats  *DistributionSummary
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
		// Try parsing as a single benchmark (not wrapped in array)
		var singleBenchmark Benchmark
		if err2 := json.Unmarshal(data, &singleBenchmark); err2 != nil {
			return nil, fmt.Errorf("parsing JSON: %w (also tried single: %w)", err, err2)
		}
		report.Benchmarks = []Benchmark{singleBenchmark}
	}

	results := &ParsedResults{
		TTFTValues: make([]float64, 0),
		ITLValues:  make([]float64, 0),
		E2EValues:  make([]float64, 0),
	}

	for _, benchmark := range report.Benchmarks {
		// Extract request counts from scheduler_state (v0.5.0 format)
		if benchmark.SchedulerState != nil {
			results.TotalRequests += benchmark.SchedulerState.CreatedRequests
			results.SuccessfulRequests += benchmark.SchedulerState.SuccessfulRequests
			results.FailedRequests += benchmark.SchedulerState.ErroredRequests
		} else if benchmark.Summary != nil {
			// Fall back to summary if available (legacy format)
			results.TotalRequests += benchmark.Summary.TotalRequests
			results.SuccessfulRequests += benchmark.Summary.SuccessfulRequests
			results.FailedRequests += benchmark.Summary.FailedRequests
			results.PromptTokens += benchmark.Summary.TotalPromptTokens
			results.OutputTokens += benchmark.Summary.TotalOutputTokens
			results.OutputTokensPerSec = benchmark.Summary.OutputTokensPerSec
			results.RequestsPerSec = benchmark.Summary.RequestsPerSec
		} else {
			// Fall back to counting requests directly (oldest format)
			results.TotalRequests += benchmark.Completed + benchmark.Errored
			results.SuccessfulRequests += benchmark.Completed
			results.FailedRequests += benchmark.Errored
		}

		// Extract metrics from v0.5.0 format
		if benchmark.Metrics != nil {
			// Extract throughput from v0.5.0 format
			if benchmark.Metrics.RequestsPerSecond != nil && benchmark.Metrics.RequestsPerSecond.Total != nil {
				results.RequestsPerSec = benchmark.Metrics.RequestsPerSecond.Total.Mean
			}
			if benchmark.Metrics.OutputTokensPS != nil && benchmark.Metrics.OutputTokensPS.Total != nil {
				results.OutputTokensPerSec = benchmark.Metrics.OutputTokensPS.Total.Mean
			}

			// Extract token counts from v0.5.0 format
			if benchmark.Metrics.PromptTokenCount != nil && benchmark.Metrics.PromptTokenCount.Successful != nil {
				results.PromptTokens += int(benchmark.Metrics.PromptTokenCount.Successful.TotalSum)
			}
			if benchmark.Metrics.OutputTokenCount != nil && benchmark.Metrics.OutputTokenCount.Successful != nil {
				results.OutputTokens += int(benchmark.Metrics.OutputTokenCount.Successful.TotalSum)
			}

			// Extract latency distributions (values are in milliseconds, convert to seconds)
			if benchmark.Metrics.TimeToFirstTokenMs != nil && benchmark.Metrics.TimeToFirstTokenMs.Successful != nil {
				results.TTFTStats = benchmark.Metrics.TimeToFirstTokenMs.Successful
			}
			if benchmark.Metrics.InterTokenLatencyMs != nil && benchmark.Metrics.InterTokenLatencyMs.Successful != nil {
				results.ITLStats = benchmark.Metrics.InterTokenLatencyMs.Successful
			}
			if benchmark.Metrics.RequestLatency != nil && benchmark.Metrics.RequestLatency.Successful != nil {
				results.E2EStats = benchmark.Metrics.RequestLatency.Successful
			}

			// Legacy format support
			if benchmark.Metrics.RequestThroughput != nil && results.RequestsPerSec == 0 {
				results.RequestsPerSec = benchmark.Metrics.RequestThroughput.Mean
			}
			if benchmark.Metrics.TokenThroughput != nil && benchmark.Metrics.TokenThroughput.OutputPerSecond != nil && results.OutputTokensPerSec == 0 {
				results.OutputTokensPerSec = benchmark.Metrics.TokenThroughput.OutputPerSecond.Mean
			}
		}

		// Extract individual request latencies from successful requests (v0.5.0 format)
		for _, req := range benchmark.Requests.Successful {
			// TimeToFirstTokenMs is in milliseconds, convert to seconds
			if req.TimeToFirstTokenMs > 0 {
				results.TTFTValues = append(results.TTFTValues, req.TimeToFirstTokenMs/1000.0)
			}
			// InterTokenLatencyMs is in milliseconds, convert to seconds
			if req.InterTokenLatencyMs > 0 {
				results.ITLValues = append(results.ITLValues, req.InterTokenLatencyMs/1000.0)
			}
			// RequestLatency is already in seconds
			if req.RequestLatency > 0 {
				results.E2EValues = append(results.E2EValues, req.RequestLatency)
			}
			results.PromptTokens += req.PromptTokens
			results.OutputTokens += req.OutputTokens
		}
	}

	return results, nil
}
