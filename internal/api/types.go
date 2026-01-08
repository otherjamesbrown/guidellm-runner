package api

import (
	"time"

	"github.com/yourorg/guidellm-runner/internal/parser"
)

// AddTargetRequest is the request body for adding a new target
type AddTargetRequest struct {
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Model       string   `json:"model"`
	Environment string   `json:"environment,omitempty"` // defaults to "dynamic"
	APIKey      string   `json:"api_key,omitempty"`
	Profile     string   `json:"profile,omitempty"`
	Rate        *float64 `json:"rate,omitempty"`
	MaxSeconds  *int     `json:"max_seconds,omitempty"`
	RequestType string   `json:"request_type,omitempty"` // chat_completions or text_completions
}

// TargetStatus represents the current state of a target
type TargetStatus string

const (
	TargetStatusStopped  TargetStatus = "stopped"
	TargetStatusRunning  TargetStatus = "running"
	TargetStatusStarting TargetStatus = "starting"
)

// TargetResponse is the response for a single target
type TargetResponse struct {
	Name        string                 `json:"name"`
	Model       string                 `json:"model"`
	URL         string                 `json:"url"`
	Environment string                 `json:"environment"`
	Status      TargetStatus           `json:"status"`
	Profile     string                 `json:"profile,omitempty"`
	Rate        float64                `json:"rate,omitempty"`
	MaxSeconds  int                    `json:"max_seconds,omitempty"`
	RequestType string                 `json:"request_type,omitempty"`
	LastRunAt   *time.Time             `json:"last_run_at,omitempty"`
	LastResults *parser.ParsedResults  `json:"last_results,omitempty"`
}

// ListTargetsResponse is the response for listing all targets
type ListTargetsResponse struct {
	Targets []TargetResponse `json:"targets"`
}

// StatusResponse is the response for the runner status endpoint
type StatusResponse struct {
	Running       bool   `json:"running"`
	TargetsCount  int    `json:"targets_count"`
	ActiveCount   int    `json:"active_count"`
	StoppedCount  int    `json:"stopped_count"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	Version       string `json:"version,omitempty"`
}

// HealthResponse is the response for the health endpoint
type HealthResponse struct {
	Status string `json:"status"`
}

// ErrorResponse is the standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// TargetActionResponse is the response for start/stop actions
type TargetActionResponse struct {
	Name    string       `json:"name"`
	Status  TargetStatus `json:"status"`
	Message string       `json:"message,omitempty"`
}

// TriggerRunRequest is the request body for triggering a manual benchmark run
type TriggerRunRequest struct {
	RunID           string                 `json:"run_id"`
	ConfigOverrides map[string]interface{} `json:"config_overrides,omitempty"`
}

// TriggerRunResponse is the response for a triggered benchmark run
type TriggerRunResponse struct {
	Name    string                 `json:"name"`
	RunID   string                 `json:"run_id"`
	Status  string                 `json:"status"`
	Results *parser.ParsedResults  `json:"results,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// SchedulerState represents the current state of the scheduler
type SchedulerState string

const (
	SchedulerStateRunning SchedulerState = "running"
	SchedulerStatePaused  SchedulerState = "paused"
)

// SchedulerStatusResponse is the response for the scheduler status endpoint
type SchedulerStatusResponse struct {
	State            SchedulerState `json:"state"`
	PausedAt         *time.Time     `json:"paused_at,omitempty"`
	NextScheduledRun *time.Time     `json:"next_scheduled_run,omitempty"`
}

// SchedulerActionResponse is the response for scheduler pause/resume actions
type SchedulerActionResponse struct {
	State   SchedulerState `json:"state"`
	Message string         `json:"message"`
}
