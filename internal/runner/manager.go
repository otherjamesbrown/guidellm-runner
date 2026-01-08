package runner

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/yourorg/guidellm-runner/internal/api"
	"github.com/yourorg/guidellm-runner/internal/config"
	"github.com/yourorg/guidellm-runner/internal/metrics"
	"github.com/yourorg/guidellm-runner/internal/parser"
)

// TargetManager manages runtime target lifecycle
type TargetManager interface {
	// AddTarget adds a new target at runtime
	AddTarget(ctx context.Context, req api.AddTargetRequest) error

	// RemoveTarget removes a target by name
	RemoveTarget(name string) error

	// StartTarget starts benchmarking for a target
	StartTarget(ctx context.Context, name string) error

	// StopTarget stops benchmarking for a target
	StopTarget(name string) error

	// TriggerRun triggers an immediate benchmark run for a target
	TriggerRun(ctx context.Context, name string, runID string) (*parser.ParsedResults, error)

	// ListTargets returns all registered targets
	ListTargets() []api.TargetResponse

	// GetTarget returns a single target by name
	GetTarget(name string) (*api.TargetResponse, bool)

	// GetStatus returns the overall runner status
	GetStatus() api.StatusResponse

	// GetLatestResults returns the latest benchmark results for a target
	GetLatestResults(name string) (*parser.ParsedResults, bool)

	// PauseScheduler pauses scheduled benchmark runs
	PauseScheduler() error

	// ResumeScheduler resumes scheduled benchmark runs
	ResumeScheduler() error

	// GetSchedulerStatus returns the current scheduler state
	GetSchedulerStatus() api.SchedulerStatusResponse
}

// managedTarget holds runtime state for a target
type managedTarget struct {
	target      config.Target
	environment string
	status      api.TargetStatus
	cancel      context.CancelFunc
	lastRunAt   *time.Time
	lastResults *parser.ParsedResults
}

// DefaultTargetManager is the default implementation of TargetManager
type DefaultTargetManager struct {
	mu                sync.RWMutex
	targets           map[string]*managedTarget
	cfg               *config.Config
	logger            *slog.Logger
	runner            *Runner
	startTime         time.Time
	wg                sync.WaitGroup
	schedulerPaused   bool
	schedulerPausedAt *time.Time
	autoResumeTimer   *time.Timer
}

// NewTargetManager creates a new DefaultTargetManager
func NewTargetManager(cfg *config.Config, logger *slog.Logger) *DefaultTargetManager {
	// Initialize metric to 0 (running)
	metrics.SchedulerPaused.Set(0)

	return &DefaultTargetManager{
		targets:   make(map[string]*managedTarget),
		cfg:       cfg,
		logger:    logger,
		startTime: time.Now(),
	}
}

// SetRunner sets the runner reference for running benchmarks
func (m *DefaultTargetManager) SetRunner(r *Runner) {
	m.runner = r
}

// AddTarget adds a new target at runtime
func (m *DefaultTargetManager) AddTarget(ctx context.Context, req api.AddTargetRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate
	if _, exists := m.targets[req.Name]; exists {
		return fmt.Errorf("target %q already exists", req.Name)
	}

	// Validate required fields
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.URL == "" {
		return fmt.Errorf("url is required")
	}
	if req.Model == "" {
		return fmt.Errorf("model is required")
	}

	// Create config.Target from request
	target := config.Target{
		Name:        req.Name,
		URL:         req.URL,
		Model:       req.Model,
		APIKey:      req.APIKey,
		Profile:     req.Profile,
		Rate:        req.Rate,
		MaxSeconds:  req.MaxSeconds,
		RequestType: req.RequestType,
	}

	// Default environment to "dynamic" for runtime-added targets
	env := req.Environment
	if env == "" {
		env = "dynamic"
	}

	m.targets[req.Name] = &managedTarget{
		target:      target,
		environment: env,
		status:      api.TargetStatusStopped,
	}

	m.logger.Info("target added",
		"name", req.Name,
		"url", req.URL,
		"model", req.Model,
		"environment", env)

	return nil
}

// RemoveTarget removes a target by name
func (m *DefaultTargetManager) RemoveTarget(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mt, exists := m.targets[name]
	if !exists {
		return fmt.Errorf("target %q not found", name)
	}

	// Stop if running
	if mt.status == api.TargetStatusRunning && mt.cancel != nil {
		mt.cancel()
	}

	delete(m.targets, name)
	m.logger.Info("target removed", "name", name)
	return nil
}

// StartTarget starts benchmarking for a target
func (m *DefaultTargetManager) StartTarget(ctx context.Context, name string) error {
	m.mu.Lock()
	mt, exists := m.targets[name]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("target %q not found", name)
	}

	if mt.status == api.TargetStatusRunning {
		m.mu.Unlock()
		return fmt.Errorf("target %q is already running", name)
	}

	// Create cancellable context for this target
	// Use Background() instead of the HTTP request context to avoid
	// cancellation when the API request completes
	targetCtx, cancel := context.WithCancel(context.Background())
	mt.cancel = cancel
	mt.status = api.TargetStatusRunning
	m.mu.Unlock()

	// Start the benchmark loop in a goroutine
	m.wg.Add(1)
	go m.runTargetLoop(targetCtx, name)

	m.logger.Info("target started", "name", name)
	return nil
}

// StopTarget stops benchmarking for a target
func (m *DefaultTargetManager) StopTarget(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mt, exists := m.targets[name]
	if !exists {
		return fmt.Errorf("target %q not found", name)
	}

	if mt.status != api.TargetStatusRunning {
		return fmt.Errorf("target %q is not running", name)
	}

	if mt.cancel != nil {
		mt.cancel()
		mt.cancel = nil
	}
	mt.status = api.TargetStatusStopped

	m.logger.Info("target stopped", "name", name)
	return nil
}

// ListTargets returns all registered targets
func (m *DefaultTargetManager) ListTargets() []api.TargetResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	targets := make([]api.TargetResponse, 0, len(m.targets))
	for _, mt := range m.targets {
		targets = append(targets, m.toTargetResponse(mt))
	}
	return targets
}

// GetTarget returns a single target by name
func (m *DefaultTargetManager) GetTarget(name string) (*api.TargetResponse, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mt, exists := m.targets[name]
	if !exists {
		return nil, false
	}

	resp := m.toTargetResponse(mt)
	return &resp, true
}

// GetStatus returns the overall runner status
func (m *DefaultTargetManager) GetStatus() api.StatusResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	activeCount := 0
	stoppedCount := 0
	for _, mt := range m.targets {
		if mt.status == api.TargetStatusRunning {
			activeCount++
		} else {
			stoppedCount++
		}
	}

	return api.StatusResponse{
		Running:       true,
		TargetsCount:  len(m.targets),
		ActiveCount:   activeCount,
		StoppedCount:  stoppedCount,
		UptimeSeconds: int64(time.Since(m.startTime).Seconds()),
	}
}

// GetLatestResults returns the latest benchmark results for a target
func (m *DefaultTargetManager) GetLatestResults(name string) (*parser.ParsedResults, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mt, exists := m.targets[name]
	if !exists {
		return nil, false
	}

	return mt.lastResults, mt.lastResults != nil
}

// TriggerRun triggers an immediate benchmark run for a target
// This runs synchronously and returns the results when complete
// After a manual run, scheduled runs are auto-paused for 60 minutes
func (m *DefaultTargetManager) TriggerRun(ctx context.Context, name string, runID string) (*parser.ParsedResults, error) {
	m.mu.RLock()
	mt, exists := m.targets[name]
	if !exists {
		m.mu.RUnlock()
		return nil, fmt.Errorf("target %q not found", name)
	}
	target := mt.target
	envName := mt.environment
	m.mu.RUnlock()

	if m.runner == nil {
		return nil, fmt.Errorf("runner not initialized")
	}

	logger := m.logger.With(
		"environment", envName,
		"target", name,
		"model", target.Model,
		"run_id", runID,
		"trigger", "manual",
	)

	logger.Info("triggering manual benchmark run")

	// Pause scheduler before manual run
	m.mu.Lock()
	wasAlreadyPaused := m.schedulerPaused
	if !wasAlreadyPaused {
		m.schedulerPaused = true
		now := time.Now()
		m.schedulerPausedAt = &now
		metrics.SchedulerPaused.Set(1)
		logger.Info("scheduler paused for manual run")
	}
	m.mu.Unlock()

	// Run the benchmark synchronously
	results := m.runner.runBenchmarkWithResults(ctx, envName, target, logger)

	// Update last run time and results
	m.mu.Lock()
	if mt, exists := m.targets[name]; exists {
		now := time.Now()
		mt.lastRunAt = &now
		mt.lastResults = results
	}

	// Set up auto-resume timer (60 minutes) if scheduler was not already paused
	if !wasAlreadyPaused {
		// Cancel existing timer if any
		if m.autoResumeTimer != nil {
			m.autoResumeTimer.Stop()
		}

		m.autoResumeTimer = time.AfterFunc(60*time.Minute, func() {
			m.mu.Lock()
			defer m.mu.Unlock()

			if m.schedulerPaused {
				m.schedulerPaused = false
				m.schedulerPausedAt = nil
				m.autoResumeTimer = nil
				metrics.SchedulerPaused.Set(0)
				m.logger.Info("scheduler auto-resumed after manual run delay")
			}
		})

		logger.Info("scheduler will auto-resume in 60 minutes")
	}
	m.mu.Unlock()

	if results == nil {
		return nil, fmt.Errorf("benchmark produced no results")
	}

	logger.Info("manual benchmark run completed",
		"requests", results.TotalRequests,
		"successful", results.SuccessfulRequests,
		"failed", results.FailedRequests)

	return results, nil
}

// LoadFromConfig loads targets from configuration (for backwards compatibility)
func (m *DefaultTargetManager) LoadFromConfig() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for envName, env := range m.cfg.Environments {
		for _, target := range env.Targets {
			m.targets[target.Name] = &managedTarget{
				target:      target,
				environment: envName,
				status:      api.TargetStatusStopped,
			}
		}
	}

	m.logger.Info("loaded targets from config", "count", len(m.targets))
}

// StartAllConfigured starts all targets loaded from configuration
func (m *DefaultTargetManager) StartAllConfigured(ctx context.Context) {
	m.mu.RLock()
	names := make([]string, 0, len(m.targets))
	for name := range m.targets {
		names = append(names, name)
	}
	m.mu.RUnlock()

	for _, name := range names {
		if err := m.StartTarget(ctx, name); err != nil {
			m.logger.Error("failed to start target", "name", name, "error", err)
		}
	}
}

// Wait waits for all running targets to complete
func (m *DefaultTargetManager) Wait() {
	m.wg.Wait()
}

// StopAll stops all running targets
func (m *DefaultTargetManager) StopAll() {
	m.mu.Lock()
	for name, mt := range m.targets {
		if mt.status == api.TargetStatusRunning && mt.cancel != nil {
			mt.cancel()
			mt.cancel = nil
			mt.status = api.TargetStatusStopped
			m.logger.Info("target stopped", "name", name)
		}
	}
	m.mu.Unlock()
}

// runTargetLoop runs the benchmark loop for a single target
func (m *DefaultTargetManager) runTargetLoop(ctx context.Context, name string) {
	defer m.wg.Done()

	m.mu.RLock()
	mt, exists := m.targets[name]
	if !exists {
		m.mu.RUnlock()
		return
	}
	target := mt.target
	envName := mt.environment
	m.mu.RUnlock()

	logger := m.logger.With(
		"environment", envName,
		"target", name,
		"model", target.Model,
	)

	logger.Info("starting benchmark loop",
		"url", target.URL,
		"profile", target.GetProfile(m.cfg.Defaults),
		"rate", target.GetRate(m.cfg.Defaults))

	ticker := time.NewTicker(m.cfg.GetInterval())
	defer ticker.Stop()

	// Run immediately, then on interval
	m.runBenchmarkWithCallback(ctx, envName, target, logger, name)

	for {
		select {
		case <-ctx.Done():
			logger.Info("stopping benchmark loop")
			m.mu.Lock()
			if mt, exists := m.targets[name]; exists {
				mt.status = api.TargetStatusStopped
			}
			m.mu.Unlock()
			return
		case <-ticker.C:
			// Check if scheduler is paused
			m.mu.RLock()
			paused := m.schedulerPaused
			m.mu.RUnlock()

			if !paused {
				m.runBenchmarkWithCallback(ctx, envName, target, logger, name)
			} else {
				logger.Debug("skipping scheduled run (scheduler paused)")
			}
		}
	}
}

// runBenchmarkWithCallback runs a benchmark and updates the target's last results
func (m *DefaultTargetManager) runBenchmarkWithCallback(ctx context.Context, envName string, target config.Target, logger *slog.Logger, name string) {
	if m.runner == nil {
		logger.Error("runner not set, cannot run benchmark")
		return
	}

	// Run the benchmark and get results
	results := m.runner.runBenchmarkWithResults(ctx, envName, target, logger)

	// Update last run time and results
	m.mu.Lock()
	if mt, exists := m.targets[name]; exists {
		now := time.Now()
		mt.lastRunAt = &now
		mt.lastResults = results
	}
	m.mu.Unlock()
}

// toTargetResponse converts a managedTarget to an API response
func (m *DefaultTargetManager) toTargetResponse(mt *managedTarget) api.TargetResponse {
	return api.TargetResponse{
		Name:        mt.target.Name,
		Model:       mt.target.Model,
		URL:         mt.target.URL,
		Environment: mt.environment,
		Status:      mt.status,
		Profile:     mt.target.GetProfile(m.cfg.Defaults),
		Rate:        mt.target.GetRate(m.cfg.Defaults),
		MaxSeconds:  mt.target.GetMaxSeconds(m.cfg.Defaults),
		RequestType: mt.target.GetRequestType(m.cfg.Defaults),
		LastRunAt:   mt.lastRunAt,
		LastResults: mt.lastResults,
	}
}

// PauseScheduler pauses all scheduled benchmark runs
func (m *DefaultTargetManager) PauseScheduler() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.schedulerPaused {
		return fmt.Errorf("scheduler is already paused")
	}

	// Cancel auto-resume timer if it exists
	if m.autoResumeTimer != nil {
		m.autoResumeTimer.Stop()
		m.autoResumeTimer = nil
	}

	m.schedulerPaused = true
	now := time.Now()
	m.schedulerPausedAt = &now

	// Update metrics
	metrics.SchedulerPaused.Set(1)

	m.logger.Info("scheduler paused")
	return nil
}

// ResumeScheduler resumes all scheduled benchmark runs
func (m *DefaultTargetManager) ResumeScheduler() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.schedulerPaused {
		return fmt.Errorf("scheduler is not paused")
	}

	// Cancel auto-resume timer if it exists
	if m.autoResumeTimer != nil {
		m.autoResumeTimer.Stop()
		m.autoResumeTimer = nil
	}

	m.schedulerPaused = false
	m.schedulerPausedAt = nil

	// Update metrics
	metrics.SchedulerPaused.Set(0)

	m.logger.Info("scheduler resumed")
	return nil
}

// GetSchedulerStatus returns the current scheduler state
func (m *DefaultTargetManager) GetSchedulerStatus() api.SchedulerStatusResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var nextScheduledRun *time.Time
	if !m.schedulerPaused {
		// Calculate next scheduled run based on interval
		for _, mt := range m.targets {
			if mt.status == api.TargetStatusRunning && mt.lastRunAt != nil {
				next := mt.lastRunAt.Add(m.cfg.GetInterval())
				if nextScheduledRun == nil || next.Before(*nextScheduledRun) {
					nextScheduledRun = &next
				}
			}
		}

		// If no last run, next run is now
		if nextScheduledRun == nil {
			now := time.Now()
			nextScheduledRun = &now
		}
	}

	return api.SchedulerStatusResponse{
		State:            m.getSchedulerState(),
		PausedAt:         m.schedulerPausedAt,
		NextScheduledRun: nextScheduledRun,
	}
}

// getSchedulerState returns the current scheduler state
func (m *DefaultTargetManager) getSchedulerState() api.SchedulerState {
	if m.schedulerPaused {
		return api.SchedulerStatePaused
	}
	return api.SchedulerStateRunning
}
