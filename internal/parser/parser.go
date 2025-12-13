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
	Profile     string       `json:"profile"`
	Rate        float64      `json:"rate"`
	Requests    []Request    `json:"requests"`
	Stats       *Stats       `json:"stats,omitempty"`
	Summary     *Summary     `json:"summary,omitempty"`
	StartTime   float64      `json:"start_time"`
	EndTime     float64      `json:"end_time"`
	Completed   int          `json:"completed_requests"`
	Errored     int          `json:"errored_requests"`
}

// Request represents a single request's data
type Request struct {
	ID              string  `json:"id,omitempty"`
	StartTime       float64 `json:"start_time"`
	EndTime         float64 `json:"end_time"`
	TTFT            float64 `json:"ttft"`             // Time to first token (seconds)
	ITL             float64 `json:"itl"`              // Inter-token latency (seconds)
	E2ELatency      float64 `json:"e2e_latency"`      // End-to-end latency (seconds)
	PromptTokens    int     `json:"prompt_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	TotalTokens     int     `json:"total_tokens"`
	Success         bool    `json:"success"`
	Error           string  `json:"error,omitempty"`
}

// Stats contains aggregated statistics
type Stats struct {
	TTFT    LatencyStats `json:"ttft"`
	ITL     LatencyStats `json:"itl"`
	E2E     LatencyStats `json:"e2e"`
}

// LatencyStats contains latency distribution statistics
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

// Summary contains overall benchmark summary
type Summary struct {
	TotalRequests       int     `json:"total_requests"`
	SuccessfulRequests  int     `json:"successful_requests"`
	FailedRequests      int     `json:"failed_requests"`
	TotalPromptTokens   int     `json:"total_prompt_tokens"`
	TotalOutputTokens   int     `json:"total_output_tokens"`
	OutputTokensPerSec  float64 `json:"output_tokens_per_second"`
	RequestsPerSec      float64 `json:"requests_per_second"`
	Duration            float64 `json:"duration_seconds"`
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
	TTFTValues    []float64
	ITLValues     []float64
	E2EValues     []float64
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
		// Extract from summary if available
		if benchmark.Summary != nil {
			results.TotalRequests += benchmark.Summary.TotalRequests
			results.SuccessfulRequests += benchmark.Summary.SuccessfulRequests
			results.FailedRequests += benchmark.Summary.FailedRequests
			results.PromptTokens += benchmark.Summary.TotalPromptTokens
			results.OutputTokens += benchmark.Summary.TotalOutputTokens
			results.OutputTokensPerSec = benchmark.Summary.OutputTokensPerSec
			results.RequestsPerSec = benchmark.Summary.RequestsPerSec
		} else {
			// Fall back to counting requests directly
			results.TotalRequests += benchmark.Completed + benchmark.Errored
			results.SuccessfulRequests += benchmark.Completed
			results.FailedRequests += benchmark.Errored
		}

		// Extract individual request latencies for histograms
		for _, req := range benchmark.Requests {
			if req.Success {
				if req.TTFT > 0 {
					results.TTFTValues = append(results.TTFTValues, req.TTFT)
				}
				if req.ITL > 0 {
					results.ITLValues = append(results.ITLValues, req.ITL)
				}
				if req.E2ELatency > 0 {
					results.E2EValues = append(results.E2EValues, req.E2ELatency)
				}
				results.PromptTokens += req.PromptTokens
				results.OutputTokens += req.OutputTokens
			}
		}
	}

	return results, nil
}
