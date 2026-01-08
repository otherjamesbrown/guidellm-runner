package discovery

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_DiscoverModels(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Quiet during tests
	}))

	t.Run("successful discovery", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/models", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Accept"))

			resp := ModelsResponse{
				Object: "list",
				Data: []ModelInfo{
					{ID: "model-1", Object: "model", ModelType: "text"},
					{ID: "model-2", Object: "model", ModelType: "vision-language"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient(logger)
		models, err := client.DiscoverModels(context.Background(), server.URL+"/v1/models", "")

		require.NoError(t, err)
		assert.Len(t, models, 2)
		assert.Equal(t, "model-1", models[0].ID)
		assert.Equal(t, "text", models[0].ModelType)
	})

	t.Run("with API key", func(t *testing.T) {
		expectedKey := "test-api-key"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer "+expectedKey, r.Header.Get("Authorization"))

			resp := ModelsResponse{Object: "list", Data: []ModelInfo{}}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient(logger)
		_, err := client.DiscoverModels(context.Background(), server.URL+"/v1/models", expectedKey)

		require.NoError(t, err)
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
		}))
		defer server.Close()

		client := NewClient(logger)
		_, err := client.DiscoverModels(context.Background(), server.URL+"/v1/models", "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status 500")
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		client := NewClient(logger)
		_, err := client.DiscoverModels(context.Background(), server.URL+"/v1/models", "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "decoding response")
	})
}

func TestFilterTextModels(t *testing.T) {
	models := []ModelInfo{
		{ID: "text-1", ModelType: "text"},
		{ID: "vlm-1", ModelType: "vision-language"},
		{ID: "text-2", ModelType: "text"},
		{ID: "other", ModelType: "embedding"},
	}

	filtered := FilterTextModels(models)

	assert.Len(t, filtered, 2)
	assert.Equal(t, "text-1", filtered[0].ID)
	assert.Equal(t, "text-2", filtered[1].ID)
}

func TestGenerateTargets(t *testing.T) {
	models := []ModelInfo{
		{ID: "unsloth/gpt-oss-20b", ModelType: "text"},
		{ID: "llama-3-1-8b-instruct", ModelType: "text"},
	}

	baseURL := "https://api.example.com/v1/chat/completions"
	apiKey := "test-key"
	envName := "test"

	targets := GenerateTargets(models, baseURL, apiKey, envName)

	require.Len(t, targets, 2)

	// First target - normalized name
	assert.Equal(t, "unsloth-gpt-oss-20b", targets[0].Name)
	assert.Equal(t, "unsloth/gpt-oss-20b", targets[0].Model)
	assert.Equal(t, baseURL, targets[0].URL)
	assert.Equal(t, apiKey, targets[0].APIKey)

	// Second target - already normalized
	assert.Equal(t, "llama-3-1-8b-instruct", targets[1].Name)
	assert.Equal(t, "llama-3-1-8b-instruct", targets[1].Model)
}

func TestNormalizeModelName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"unsloth/gpt-oss-20b", "unsloth-gpt-oss-20b"},
		{"openai/gpt-4", "openai-gpt-4"},
		{"llama-3-1-8b", "llama-3-1-8b"},
		{"meta-llama/Llama-2-7b-hf", "meta-llama-Llama-2-7b-hf"},
		{"model/with/multiple/slashes", "model-with-multiple-slashes"},
		{"-leading-hyphen", "leading-hyphen"},
		{"trailing-hyphen-", "trailing-hyphen"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeModelName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
