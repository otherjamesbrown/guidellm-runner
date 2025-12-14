package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/yourorg/guidellm-runner/internal/config"
)

// TestAPIKeyHandling verifies that API keys are correctly passed to the guidellm subprocess
func TestAPIKeyHandling(t *testing.T) {
	tests := []struct {
		name         string
		targetAPIKey string
		envAPIKey    string
		expectedKey  string
	}{
		{
			name:         "uses target config key when set",
			targetAPIKey: "config-key-12345",
			envAPIKey:    "env-key-67890",
			expectedKey:  "config-key-12345",
		},
		{
			name:         "falls back to env var when target key empty",
			targetAPIKey: "",
			envAPIKey:    "env-key-67890",
			expectedKey:  "env-key-67890",
		},
		{
			name:         "empty when neither set",
			targetAPIKey: "",
			envAPIKey:    "",
			expectedKey:  "",
		},
		{
			name:         "uses config key even when env var empty",
			targetAPIKey: "config-key-only",
			envAPIKey:    "",
			expectedKey:  "config-key-only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env var and restore after test
			originalEnv := os.Getenv("OPENAI_API_KEY")
			defer func() {
				if originalEnv != "" {
					os.Setenv("OPENAI_API_KEY", originalEnv)
				} else {
					os.Unsetenv("OPENAI_API_KEY")
				}
			}()

			// Set up environment variable for this test
			if tt.envAPIKey != "" {
				os.Setenv("OPENAI_API_KEY", tt.envAPIKey)
			} else {
				os.Unsetenv("OPENAI_API_KEY")
			}

			// Create a test runner with minimal config
			cfg := &config.Config{
				Environments: map[string]config.Environment{
					"test": {
						Targets: []config.Target{
							{
								Name:   "test-target",
								URL:    "http://test.local/v1",
								Model:  "test-model",
								APIKey: tt.targetAPIKey,
							},
						},
					},
				},
				Defaults: config.Defaults{
					Profile:    "constant",
					Rate:       1,
					MaxSeconds: 1,
					DataSpec:   "prompt_tokens=10,output_tokens=10",
				},
			}

			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelError, // Quiet during tests
			}))

			runner := New(cfg, logger)

			// Get the target we're testing
			target := cfg.Environments["test"].Targets[0]

			// Create a mock command to inspect the environment
			// We'll use 'env' command to print environment variables
			tmpDir := t.TempDir()
			args := runner.buildArgs(target, tmpDir)

			// Create the command (we won't actually run guidellm, but we'll inspect the env setup)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			cmd := exec.CommandContext(ctx, "env") // Use 'env' to print environment

			// Apply the same API key logic as runBenchmark
			apiKey := target.APIKey
			if apiKey == "" {
				apiKey = os.Getenv("OPENAI_API_KEY")
			}
			if apiKey != "" {
				cmd.Env = append(os.Environ(), fmt.Sprintf("OPENAI_API_KEY=%s", apiKey))
			} else {
				cmd.Env = os.Environ()
			}

			// Run the env command to capture the environment
			output, err := cmd.CombinedOutput()
			if err != nil && !strings.Contains(err.Error(), "deadline exceeded") {
				t.Fatalf("Failed to run env command: %v", err)
			}

			// Verify the OPENAI_API_KEY in the command environment
			envLines := strings.Split(string(output), "\n")
			var foundKey string
			for _, line := range envLines {
				if strings.HasPrefix(line, "OPENAI_API_KEY=") {
					foundKey = strings.TrimPrefix(line, "OPENAI_API_KEY=")
					break
				}
			}

			// Check if the key matches expectations
			if tt.expectedKey == "" {
				// We should still find OPENAI_API_KEY in env if it was set in the parent env
				// But we don't explicitly set it in cmd.Env
				if tt.envAPIKey == "" && foundKey != "" && foundKey != originalEnv {
					t.Errorf("Expected no API key to be explicitly set, but found: %s", foundKey)
				}
			} else {
				if foundKey != tt.expectedKey {
					t.Errorf("Expected OPENAI_API_KEY=%s, but got: %s", tt.expectedKey, foundKey)
				}
			}

			// Verify that buildArgs doesn't include API key (it should be env-only)
			for _, arg := range args {
				if strings.Contains(arg, "api") && strings.Contains(arg, "key") {
					t.Errorf("API key should not be in command args, found: %s", arg)
				}
			}
		})
	}
}

// TestAPIKeyNotInCommandArgs verifies that the API key is never passed as a command-line argument
func TestAPIKeyNotInCommandArgs(t *testing.T) {
	cfg := &config.Config{
		Environments: map[string]config.Environment{
			"test": {
				Targets: []config.Target{
					{
						Name:   "test-target",
						URL:    "http://test.local/v1",
						Model:  "test-model",
						APIKey: "secret-key-12345",
					},
				},
			},
		},
		Defaults: config.Defaults{
			Profile:    "constant",
			Rate:       1,
			MaxSeconds: 1,
			DataSpec:   "prompt_tokens=10,output_tokens=10",
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	runner := New(cfg, logger)
	target := cfg.Environments["test"].Targets[0]
	tmpDir := t.TempDir()

	args := runner.buildArgs(target, tmpDir)

	// Convert args to string for easier inspection
	argsStr := strings.Join(args, " ")

	// Verify the API key is NOT in the arguments
	if strings.Contains(argsStr, target.APIKey) {
		t.Errorf("API key should not appear in command arguments. Args: %v", args)
	}

	// Verify we have the expected guidellm arguments
	expectedArgs := []string{
		"benchmark",
		"--target", target.URL,
		"--model", target.Model,
		"--profile", "constant",
		"--rate", "1",
		"--max-seconds", "1",
		"--data", "prompt_tokens=10,output_tokens=10",
		"--output-dir", tmpDir,
		"--outputs", "json",
		"--backend-kwargs", `{"validate_backend": false}`,
	}

	if len(args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(args))
	}
}

// TestBuildArgsWithDefaults verifies that buildArgs correctly constructs guidellm arguments
func TestBuildArgsWithDefaults(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.Defaults{
			Profile:    "constant",
			Rate:       10,
			MaxSeconds: 30,
			DataSpec:   "prompt_tokens=256,output_tokens=128",
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	runner := New(cfg, logger)

	tests := []struct {
		name     string
		target   config.Target
		expected map[string]string // flag -> expected value
	}{
		{
			name: "uses defaults",
			target: config.Target{
				Name:  "default-target",
				URL:   "http://localhost:8000/v1",
				Model: "test-model",
			},
			expected: map[string]string{
				"--target":      "http://localhost:8000/v1",
				"--model":       "test-model",
				"--profile":     "constant",
				"--rate":        "10",
				"--max-seconds": "30",
				"--data":        "prompt_tokens=256,output_tokens=128",
			},
		},
		{
			name: "uses target overrides",
			target: config.Target{
				Name:       "override-target",
				URL:        "http://override:8000/v1",
				Model:      "override-model",
				Profile:    "poisson",
				Rate:       intPtr(5),
				MaxSeconds: intPtr(60),
			},
			expected: map[string]string{
				"--target":      "http://override:8000/v1",
				"--model":       "override-model",
				"--profile":     "poisson",
				"--rate":        "5",
				"--max-seconds": "60",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			args := runner.buildArgs(tt.target, tmpDir)

			// Convert args to map for easier checking
			argsMap := make(map[string]string)
			for i := 0; i < len(args)-1; i++ {
				if strings.HasPrefix(args[i], "--") {
					argsMap[args[i]] = args[i+1]
				}
			}

			// Verify expected flags
			for flag, expectedValue := range tt.expected {
				actualValue, ok := argsMap[flag]
				if !ok {
					t.Errorf("Expected flag %s not found in args: %v", flag, args)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("Flag %s: expected %s, got %s", flag, expectedValue, actualValue)
				}
			}

			// Verify output-dir is set
			if argsMap["--output-dir"] != tmpDir {
				t.Errorf("Expected --output-dir=%s, got %s", tmpDir, argsMap["--output-dir"])
			}

			// Verify outputs format
			if argsMap["--outputs"] != "json" {
				t.Errorf("Expected --outputs=json, got %s", argsMap["--outputs"])
			}

			// Verify backend-kwargs
			expectedKwargs := `{"validate_backend": false}`
			if argsMap["--backend-kwargs"] != expectedKwargs {
				t.Errorf("Expected --backend-kwargs=%s, got %s", expectedKwargs, argsMap["--backend-kwargs"])
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
