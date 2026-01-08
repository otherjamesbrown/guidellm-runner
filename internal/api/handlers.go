package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/yourorg/guidellm-runner/internal/parser"
)

// TargetManager interface for the handlers to use
// This matches the interface in runner/manager.go
type TargetManager interface {
	AddTarget(ctx context.Context, req AddTargetRequest) error
	RemoveTarget(name string) error
	StartTarget(ctx context.Context, name string) error
	StopTarget(name string) error
	TriggerRun(ctx context.Context, name string, runID string) (*parser.ParsedResults, error)
	ListTargets() []TargetResponse
	GetTarget(name string) (*TargetResponse, bool)
	GetStatus() StatusResponse
	GetLatestResults(name string) (*parser.ParsedResults, bool)
	PauseScheduler() error
	ResumeScheduler() error
	GetSchedulerStatus() SchedulerStatusResponse
}

// Handlers contains the HTTP handlers for the API
type Handlers struct {
	manager TargetManager
	logger  *slog.Logger
}

// NewHandlers creates a new Handlers instance
func NewHandlers(manager TargetManager, logger *slog.Logger) *Handlers {
	return &Handlers{
		manager: manager,
		logger:  logger,
	}
}

// ListTargets handles GET /api/targets
func (h *Handlers) ListTargets(w http.ResponseWriter, r *http.Request) {
	targets := h.manager.ListTargets()
	h.respondJSON(w, http.StatusOK, ListTargetsResponse{Targets: targets})
}

// AddTarget handles POST /api/targets
func (h *Handlers) AddTarget(w http.ResponseWriter, r *http.Request) {
	var req AddTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := h.manager.AddTarget(r.Context(), req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	target, ok := h.manager.GetTarget(req.Name)
	if !ok {
		h.respondError(w, http.StatusInternalServerError, "target added but not found", "")
		return
	}

	h.respondJSON(w, http.StatusCreated, target)
}

// GetTarget handles GET /api/targets/{name}
func (h *Handlers) GetTarget(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		h.respondError(w, http.StatusBadRequest, "target name is required", "")
		return
	}

	target, ok := h.manager.GetTarget(name)
	if !ok {
		h.respondError(w, http.StatusNotFound, "target not found", "")
		return
	}

	h.respondJSON(w, http.StatusOK, target)
}

// RemoveTarget handles DELETE /api/targets/{name}
func (h *Handlers) RemoveTarget(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		h.respondError(w, http.StatusBadRequest, "target name is required", "")
		return
	}

	if err := h.manager.RemoveTarget(name); err != nil {
		h.respondError(w, http.StatusNotFound, err.Error(), "")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{
		"message": "target removed",
		"name":    name,
	})
}

// StartTarget handles POST /api/targets/{name}/start
func (h *Handlers) StartTarget(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		h.respondError(w, http.StatusBadRequest, "target name is required", "")
		return
	}

	if err := h.manager.StartTarget(r.Context(), name); err != nil {
		// Check if it's a not found error
		if _, ok := h.manager.GetTarget(name); !ok {
			h.respondError(w, http.StatusNotFound, err.Error(), "")
			return
		}
		h.respondError(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	h.respondJSON(w, http.StatusOK, TargetActionResponse{
		Name:    name,
		Status:  TargetStatusRunning,
		Message: "target started",
	})
}

// StopTarget handles POST /api/targets/{name}/stop
func (h *Handlers) StopTarget(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		h.respondError(w, http.StatusBadRequest, "target name is required", "")
		return
	}

	if err := h.manager.StopTarget(name); err != nil {
		// Check if it's a not found error
		if _, ok := h.manager.GetTarget(name); !ok {
			h.respondError(w, http.StatusNotFound, err.Error(), "")
			return
		}
		h.respondError(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	h.respondJSON(w, http.StatusOK, TargetActionResponse{
		Name:    name,
		Status:  TargetStatusStopped,
		Message: "target stopped",
	})
}

// GetTargetResults handles GET /api/targets/{name}/results
func (h *Handlers) GetTargetResults(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		h.respondError(w, http.StatusBadRequest, "target name is required", "")
		return
	}

	// Check if target exists
	if _, ok := h.manager.GetTarget(name); !ok {
		h.respondError(w, http.StatusNotFound, "target not found", "")
		return
	}

	results, ok := h.manager.GetLatestResults(name)
	if !ok {
		h.respondJSON(w, http.StatusOK, map[string]interface{}{
			"name":    name,
			"results": nil,
			"message": "no results available yet",
		})
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"name":    name,
		"results": results,
	})
}

// TriggerRun handles POST /api/targets/{name}/trigger
func (h *Handlers) TriggerRun(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		h.respondError(w, http.StatusBadRequest, "target name is required", "")
		return
	}

	// Check if target exists
	if _, ok := h.manager.GetTarget(name); !ok {
		h.respondError(w, http.StatusNotFound, "target not found", "")
		return
	}

	var req TriggerRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	h.logger.Info("trigger run requested", "target", name, "run_id", req.RunID)

	// Run the benchmark synchronously (this may take a while)
	results, err := h.manager.TriggerRun(r.Context(), name, req.RunID)
	if err != nil {
		h.logger.Error("trigger run failed", "target", name, "error", err)
		h.respondJSON(w, http.StatusOK, TriggerRunResponse{
			Name:   name,
			RunID:  req.RunID,
			Status: "failed",
			Error:  err.Error(),
		})
		return
	}

	h.respondJSON(w, http.StatusOK, TriggerRunResponse{
		Name:    name,
		RunID:   req.RunID,
		Status:  "completed",
		Results: results,
	})
}

// GetStatus handles GET /api/status
func (h *Handlers) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.manager.GetStatus()
	h.respondJSON(w, http.StatusOK, status)
}

// HealthCheck handles GET /api/health
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

// PauseBenchmark handles POST /api/v1/benchmark/pause
func (h *Handlers) PauseBenchmark(w http.ResponseWriter, r *http.Request) {
	if err := h.manager.PauseScheduler(); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	h.respondJSON(w, http.StatusOK, SchedulerActionResponse{
		State:   SchedulerStatePaused,
		Message: "scheduler paused",
	})
}

// ResumeBenchmark handles POST /api/v1/benchmark/resume
func (h *Handlers) ResumeBenchmark(w http.ResponseWriter, r *http.Request) {
	if err := h.manager.ResumeScheduler(); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	h.respondJSON(w, http.StatusOK, SchedulerActionResponse{
		State:   SchedulerStateRunning,
		Message: "scheduler resumed",
	})
}

// GetBenchmarkStatus handles GET /api/v1/benchmark/status
func (h *Handlers) GetBenchmarkStatus(w http.ResponseWriter, r *http.Request) {
	status := h.manager.GetSchedulerStatus()
	h.respondJSON(w, http.StatusOK, status)
}

// TriggerManualRun handles POST /api/v1/benchmark/run
// Triggers immediate manual runs for all active targets
func (h *Handlers) TriggerManualRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RunID  string `json:"run_id"`
		Target string `json:"target,omitempty"` // Optional: run specific target only
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// If target is specified, run that target only
	if req.Target != "" {
		results, err := h.manager.TriggerRun(r.Context(), req.Target, req.RunID)
		if err != nil {
			if _, ok := h.manager.GetTarget(req.Target); !ok {
				h.respondError(w, http.StatusNotFound, "target not found", "")
				return
			}
			h.respondJSON(w, http.StatusOK, TriggerRunResponse{
				Name:   req.Target,
				RunID:  req.RunID,
				Status: "failed",
				Error:  err.Error(),
			})
			return
		}

		h.respondJSON(w, http.StatusOK, TriggerRunResponse{
			Name:    req.Target,
			RunID:   req.RunID,
			Status:  "completed",
			Results: results,
		})
		return
	}

	// Otherwise, trigger all running targets
	targets := h.manager.ListTargets()
	runningTargets := 0
	for _, t := range targets {
		if t.Status == TargetStatusRunning {
			runningTargets++
		}
	}

	if runningTargets == 0 {
		h.respondError(w, http.StatusBadRequest, "no running targets to trigger", "")
		return
	}

	// Trigger all running targets asynchronously
	h.respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"message":         "manual run triggered for all running targets",
		"run_id":          req.RunID,
		"triggered_count": runningTargets,
	})
}

// respondJSON writes a JSON response
func (h *Handlers) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

// respondError writes an error response
func (h *Handlers) respondError(w http.ResponseWriter, status int, error string, message string) {
	w.WriteHeader(status)
	resp := ErrorResponse{Error: error}
	if message != "" {
		resp.Message = message
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode error response", "error", err)
	}
}
