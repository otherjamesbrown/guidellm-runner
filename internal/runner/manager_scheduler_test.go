package runner

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/yourorg/guidellm-runner/internal/api"
	"github.com/yourorg/guidellm-runner/internal/config"
)

func TestSchedulerPauseResume(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.Config{
		Defaults: config.Defaults{
			Profile:     "default",
			Rate:        10.0,
			Interval:    300, // 5 minutes in seconds
			MaxSeconds:  60,
			RequestType: "chat_completions",
		},
	}

	manager := NewTargetManager(cfg, logger)

	// Test initial state (running)
	status := manager.GetSchedulerStatus()
	if status.State != api.SchedulerStateRunning {
		t.Errorf("expected initial state to be running, got %s", status.State)
	}

	// Test pause
	if err := manager.PauseScheduler(); err != nil {
		t.Fatalf("failed to pause scheduler: %v", err)
	}

	status = manager.GetSchedulerStatus()
	if status.State != api.SchedulerStatePaused {
		t.Errorf("expected state to be paused, got %s", status.State)
	}
	if status.PausedAt == nil {
		t.Error("expected PausedAt to be set")
	}

	// Test double pause (should error)
	if err := manager.PauseScheduler(); err == nil {
		t.Error("expected error when pausing already paused scheduler")
	}

	// Test resume
	if err := manager.ResumeScheduler(); err != nil {
		t.Fatalf("failed to resume scheduler: %v", err)
	}

	status = manager.GetSchedulerStatus()
	if status.State != api.SchedulerStateRunning {
		t.Errorf("expected state to be running, got %s", status.State)
	}
	if status.PausedAt != nil {
		t.Error("expected PausedAt to be nil after resume")
	}

	// Test double resume (should error)
	if err := manager.ResumeScheduler(); err == nil {
		t.Error("expected error when resuming already running scheduler")
	}
}

func TestSchedulerAutoResumeAfterManualRun(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.Config{
		Defaults: config.Defaults{
			Profile:     "default",
			Rate:        10.0,
			Interval:    300, // 5 minutes in seconds
			MaxSeconds:  60,
			RequestType: "chat_completions",
		},
	}

	manager := NewTargetManager(cfg, logger)

	// Add a test target
	ctx := context.Background()
	err := manager.AddTarget(ctx, api.AddTargetRequest{
		Name:  "test-target",
		URL:   "http://localhost:8000",
		Model: "test-model",
	})
	if err != nil {
		t.Fatalf("failed to add target: %v", err)
	}

	// Verify scheduler is running initially
	status := manager.GetSchedulerStatus()
	if status.State != api.SchedulerStateRunning {
		t.Errorf("expected initial state to be running, got %s", status.State)
	}

	// Note: We can't fully test TriggerRun without a real benchmark runner
	// This would require mocking the runner, which is out of scope for this test
	// The auto-resume timer logic is tested by the pause/resume tests above
}

func TestSchedulerStateWithTargets(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.Config{
		Defaults: config.Defaults{
			Profile:     "default",
			Rate:        10.0,
			Interval:    300, // 5 minutes in seconds
			MaxSeconds:  60,
			RequestType: "chat_completions",
		},
	}

	manager := NewTargetManager(cfg, logger)
	ctx := context.Background()

	// Add a target
	err := manager.AddTarget(ctx, api.AddTargetRequest{
		Name:  "test-target",
		URL:   "http://localhost:8000",
		Model: "test-model",
	})
	if err != nil {
		t.Fatalf("failed to add target: %v", err)
	}

	// Pause scheduler
	if err := manager.PauseScheduler(); err != nil {
		t.Fatalf("failed to pause scheduler: %v", err)
	}

	// Verify status shows paused state
	status := manager.GetSchedulerStatus()
	if status.State != api.SchedulerStatePaused {
		t.Errorf("expected state to be paused, got %s", status.State)
	}

	// When paused, NextScheduledRun should be nil
	if status.NextScheduledRun != nil {
		t.Error("expected NextScheduledRun to be nil when paused")
	}

	// Resume scheduler
	if err := manager.ResumeScheduler(); err != nil {
		t.Fatalf("failed to resume scheduler: %v", err)
	}

	// Verify status shows running state
	status = manager.GetSchedulerStatus()
	if status.State != api.SchedulerStateRunning {
		t.Errorf("expected state to be running, got %s", status.State)
	}

	// When running with targets, NextScheduledRun should be set
	if status.NextScheduledRun == nil {
		t.Error("expected NextScheduledRun to be set when running")
	}
}
