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
	ListTargets() []TargetResponse
	GetTarget(name string) (*TargetResponse, bool)
	GetStatus() StatusResponse
	GetLatestResults(name string) (*parser.ParsedResults, bool)
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

// GetStatus handles GET /api/status
func (h *Handlers) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.manager.GetStatus()
	h.respondJSON(w, http.StatusOK, status)
}

// HealthCheck handles GET /api/health
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
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
