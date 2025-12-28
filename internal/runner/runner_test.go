package runner

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

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

			// Apply the same API key logic as runBenchmark
			apiKey := target.APIKey
			if apiKey == "" {
				apiKey = os.Getenv("OPENAI_API_KEY")
			}

			args := runner.buildArgs(target, tmpDir, apiKey)

			// Verify that API key is correctly included in request-formatter-kwargs
			// The implementation now passes API key via Authorization header in request-formatter-kwargs
			argsStr := strings.Join(args, " ")

			if tt.expectedKey != "" {
				// When API key is set, it should appear in request-formatter-kwargs
				expectedHeader := fmt.Sprintf(`"Authorization": "Bearer %s"`, tt.expectedKey)
				if !strings.Contains(argsStr, expectedHeader) {
					t.Errorf("Expected API key in request-formatter-kwargs with header %s, but not found in args: %v", expectedHeader, args)
				}
			} else {
				// When no API key, request-formatter-kwargs should just have stream: false
				if strings.Contains(argsStr, "Authorization") {
					t.Errorf("Expected no Authorization header when API key is empty, but found in args: %v", args)
				}
			}
		})
	}
}

// TestAPIKeyInAuthHeader verifies that the API key is passed via Authorization header in request-formatter-kwargs
func TestAPIKeyInAuthHeader(t *testing.T) {
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

	args := runner.buildArgs(target, tmpDir, target.APIKey)

	// Convert args to string for easier inspection
	argsStr := strings.Join(args, " ")

	// Verify the API key IS in the arguments via Authorization header
	expectedHeader := fmt.Sprintf(`"Authorization": "Bearer %s"`, target.APIKey)
	if !strings.Contains(argsStr, expectedHeader) {
		t.Errorf("API key should appear in Authorization header. Expected %s in args: %v", expectedHeader, args)
	}

	// Verify we have the expected guidellm arguments
	expectedFlags := []string{
		"benchmark",
		"--target",
		"--model",
		"--profile",
		"--rate",
		"--max-seconds",
		"--data",
		"--output-dir",
		"--outputs",
		"--backend-kwargs",
		"--request-type",
		"--processor",
		"--request-formatter-kwargs",
	}

	for _, flag := range expectedFlags {
		if !strings.Contains(argsStr, flag) {
			t.Errorf("Expected flag %s in args: %v", flag, args)
		}
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
			args := runner.buildArgs(tt.target, tmpDir, "") // empty apiKey for these tests

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

// TestRequestTypeConfiguration verifies that request type is correctly configured
func TestRequestTypeConfiguration(t *testing.T) {
	tests := []struct {
		name                string
		defaultRequestType  string
		targetRequestType   string
		expectedRequestType string
	}{
		{
			name:                "uses default text_completions when not specified",
			defaultRequestType:  "",
			targetRequestType:   "",
			expectedRequestType: "text_completions",
		},
		{
			name:                "uses target override when specified",
			defaultRequestType:  "chat_completions",
			targetRequestType:   "text_completions",
			expectedRequestType: "text_completions",
		},
		{
			name:                "uses default when target not specified",
			defaultRequestType:  "text_completions",
			targetRequestType:   "",
			expectedRequestType: "text_completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Defaults: config.Defaults{
					Profile:     "constant",
					Rate:        1,
					MaxSeconds:  1,
					DataSpec:    "prompt_tokens=10,output_tokens=10",
					RequestType: tt.defaultRequestType,
				},
			}

			// Apply default if not set (mimics Load behavior)
			if cfg.Defaults.RequestType == "" {
				cfg.Defaults.RequestType = "text_completions"
			}

			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelError,
			}))

			runner := New(cfg, logger)

			target := config.Target{
				Name:        "test-target",
				URL:         "http://test.local/v1/chat/completions",
				Model:       "test-model",
				RequestType: tt.targetRequestType,
			}

			tmpDir := t.TempDir()
			args := runner.buildArgs(target, tmpDir, "")

			// Find the request-type value in args
			var actualRequestType string
			for i, arg := range args {
				if arg == "--request-type" && i+1 < len(args) {
					actualRequestType = args[i+1]
					break
				}
			}

			if actualRequestType != tt.expectedRequestType {
				t.Errorf("Expected request type %s, got %s", tt.expectedRequestType, actualRequestType)
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
