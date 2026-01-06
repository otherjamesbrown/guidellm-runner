package parser

import (
	"testing"
)

func TestParse(t *testing.T) {
	// Sample GuideLLM v0.5.0 output based on actual structure
	sampleJSON := `{
		"metadata": {
			"version": 1,
			"guidellm_version": "0.5.0"
		},
		"args": {
			"target": "http://localhost:8000/v1",
			"model": "test-model"
		},
		"benchmarks": [
			{
				"type_": "benchmark",
				"config": {
					"id_": "test-id",
					"run_id": "test-run"
				},
				"scheduler_state": {
					"created_requests": 100,
					"successful_requests": 95,
					"errored_requests": 5,
					"cancelled_requests": 0,
					"processed_requests": 100
				},
				"metrics": {
					"request_totals": {
						"successful": 95,
						"errored": 5,
						"incomplete": 0,
						"total": 100
					},
					"requests_per_second": {
						"successful": {
							"mean": 10.5,
							"median": 10.0,
							"mode": 10.0,
							"variance": 1.0,
							"std_dev": 1.0,
							"min": 8.0,
							"max": 13.0,
							"count": 95,
							"total_sum": 997.5,
							"percentiles": {
								"p001": 8.0,
								"p01": 8.5,
								"p05": 9.0,
								"p10": 9.5,
								"p25": 10.0,
								"p50": 10.5,
								"p75": 11.0,
								"p90": 11.5,
								"p95": 12.0,
								"p99": 12.5,
								"p999": 13.0
							}
						},
						"errored": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"incomplete": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"total": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}}
					},
					"request_latency": {
						"successful": {
							"mean": 0.5,
							"median": 0.45,
							"mode": 0.4,
							"variance": 0.01,
							"std_dev": 0.1,
							"min": 0.3,
							"max": 0.8,
							"count": 95,
							"total_sum": 47.5,
							"percentiles": {
								"p001": 0.3,
								"p01": 0.32,
								"p05": 0.35,
								"p10": 0.38,
								"p25": 0.42,
								"p50": 0.45,
								"p75": 0.55,
								"p90": 0.65,
								"p95": 0.70,
								"p99": 0.75,
								"p999": 0.80
							}
						},
						"errored": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"incomplete": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"total": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}}
					},
					"prompt_token_count": {
						"successful": {
							"mean": 50,
							"median": 50,
							"mode": 50,
							"variance": 0,
							"std_dev": 0,
							"min": 50,
							"max": 50,
							"count": 95,
							"total_sum": 4750,
							"percentiles": {"p001": 50, "p01": 50, "p05": 50, "p10": 50, "p25": 50, "p50": 50, "p75": 50, "p90": 50, "p95": 50, "p99": 50, "p999": 50}
						},
						"errored": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"incomplete": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"total": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}}
					},
					"output_token_count": {
						"successful": {
							"mean": 20,
							"median": 20,
							"mode": 20,
							"variance": 0,
							"std_dev": 0,
							"min": 20,
							"max": 20,
							"count": 95,
							"total_sum": 1900,
							"percentiles": {"p001": 20, "p01": 20, "p05": 20, "p10": 20, "p25": 20, "p50": 20, "p75": 20, "p90": 20, "p95": 20, "p99": 20, "p999": 20}
						},
						"errored": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"incomplete": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"total": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}}
					},
					"total_token_count": {
						"successful": {"mean": 70, "median": 70, "mode": 70, "variance": 0, "std_dev": 0, "min": 70, "max": 70, "count": 95, "total_sum": 6650, "percentiles": {"p001": 70, "p01": 70, "p05": 70, "p10": 70, "p25": 70, "p50": 70, "p75": 70, "p90": 70, "p95": 70, "p99": 70, "p999": 70}},
						"errored": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"incomplete": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"total": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}}
					},
					"time_to_first_token_ms": {
						"successful": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"errored": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"incomplete": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"total": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}}
					},
					"inter_token_latency_ms": {
						"successful": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"errored": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"incomplete": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"total": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}}
					},
					"output_tokens_per_second": {
						"successful": {
							"mean": 40.0,
							"median": 38.0,
							"mode": 35.0,
							"variance": 25.0,
							"std_dev": 5.0,
							"min": 30.0,
							"max": 55.0,
							"count": 95,
							"total_sum": 3800.0,
							"percentiles": {"p001": 30, "p01": 31, "p05": 32, "p10": 33, "p25": 35, "p50": 38, "p75": 45, "p90": 50, "p95": 52, "p99": 54, "p999": 55}
						},
						"errored": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"incomplete": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"total": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}}
					},
					"tokens_per_second": {
						"successful": {"mean": 140.0, "median": 138.0, "mode": 135.0, "variance": 25.0, "std_dev": 5.0, "min": 130.0, "max": 155.0, "count": 95, "total_sum": 13300.0, "percentiles": {"p001": 130, "p01": 131, "p05": 132, "p10": 133, "p25": 135, "p50": 138, "p75": 145, "p90": 150, "p95": 152, "p99": 154, "p999": 155}},
						"errored": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"incomplete": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}},
						"total": {"mean": 0, "median": 0, "mode": 0, "variance": 0, "std_dev": 0, "min": 0, "max": 0, "count": 0, "total_sum": 0, "percentiles": {"p001": 0, "p01": 0, "p05": 0, "p10": 0, "p25": 0, "p50": 0, "p75": 0, "p90": 0, "p95": 0, "p99": 0, "p999": 0}}
					}
				}
			}
		]
	}`

	results, err := Parse([]byte(sampleJSON))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify request counts from scheduler_state
	if results.TotalRequests != 100 {
		t.Errorf("TotalRequests = %d, want 100", results.TotalRequests)
	}
	if results.SuccessfulRequests != 95 {
		t.Errorf("SuccessfulRequests = %d, want 95", results.SuccessfulRequests)
	}
	if results.FailedRequests != 5 {
		t.Errorf("FailedRequests = %d, want 5", results.FailedRequests)
	}

	// Verify token counts
	if results.PromptTokens != 4750 {
		t.Errorf("PromptTokens = %d, want 4750", results.PromptTokens)
	}
	if results.OutputTokens != 1900 {
		t.Errorf("OutputTokens = %d, want 1900", results.OutputTokens)
	}

	// Verify throughput
	if results.OutputTokensPerSec != 40.0 {
		t.Errorf("OutputTokensPerSec = %f, want 40.0", results.OutputTokensPerSec)
	}
	if results.RequestsPerSec != 10.5 {
		t.Errorf("RequestsPerSec = %f, want 10.5", results.RequestsPerSec)
	}

	// Verify E2E latency values were generated (100 samples from percentiles)
	if len(results.E2EValues) != 100 {
		t.Errorf("E2EValues length = %d, want 100", len(results.E2EValues))
	}

	// Verify E2EStats were captured
	if results.E2EStats == nil {
		t.Error("E2EStats is nil, want non-nil")
	} else if results.E2EStats.Mean != 0.5 {
		t.Errorf("E2EStats.Mean = %f, want 0.5", results.E2EStats.Mean)
	}

	// TTFT and ITL should be empty (no streaming data)
	if len(results.TTFTValues) != 0 {
		t.Errorf("TTFTValues length = %d, want 0 (no streaming)", len(results.TTFTValues))
	}
	if len(results.ITLValues) != 0 {
		t.Errorf("ITLValues length = %d, want 0 (no streaming)", len(results.ITLValues))
	}
}

func TestGenerateValuesFromDistribution(t *testing.T) {
	stats := &DistributionSummary{
		Count: 100,
		Percentiles: Percentiles{
			P01:  0.32,
			P05:  0.35,
			P10:  0.38,
			P25:  0.42,
			P50:  0.45,
			P75:  0.55,
			P90:  0.65,
			P95:  0.70,
			P99:  0.75,
			P999: 0.80,
		},
	}

	values := generateValuesFromDistribution(stats)

	// Should generate 100 samples
	if len(values) != 100 {
		t.Errorf("Generated %d values, want 100", len(values))
	}

	// Check that values are in expected range
	for i, v := range values {
		if v < 0.32 || v > 0.80 {
			t.Errorf("Value[%d] = %f, out of expected range [0.32, 0.80]", i, v)
		}
	}
}

func TestGenerateValuesFromDistribution_NilStats(t *testing.T) {
	values := generateValuesFromDistribution(nil)
	if values != nil {
		t.Error("Expected nil for nil stats")
	}
}

func TestGenerateValuesFromDistribution_ZeroCount(t *testing.T) {
	stats := &DistributionSummary{Count: 0}
	values := generateValuesFromDistribution(stats)
	if values != nil {
		t.Error("Expected nil for zero count")
	}
}
