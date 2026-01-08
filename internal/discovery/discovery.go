package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/yourorg/guidellm-runner/internal/config"
)

// ModelInfo represents a model from the /v1/models endpoint
type ModelInfo struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Created   int64  `json:"created"`
	OwnedBy   string `json:"owned_by"`
	ModelType string `json:"model_type"`
}

// ModelsResponse represents the /v1/models API response
type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

// Client handles model discovery from API endpoints
type Client struct {
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient creates a new discovery client
func NewClient(logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// DiscoverModels fetches available models from the /v1/models endpoint
func (c *Client) DiscoverModels(ctx context.Context, endpoint, apiKey string) ([]ModelInfo, error) {
	c.logger.Info("discovering models", "endpoint", endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	c.logger.Info("discovered models", "count", len(modelsResp.Data))
	return modelsResp.Data, nil
}

// FilterTextModels filters models to only include text generation models
func FilterTextModels(models []ModelInfo) []ModelInfo {
	filtered := make([]ModelInfo, 0, len(models))
	for _, model := range models {
		if model.ModelType == "text" {
			filtered = append(filtered, model)
		}
	}
	return filtered
}

// GenerateTargets converts discovered models into benchmark targets
func GenerateTargets(models []ModelInfo, baseURL, apiKey string, envName string) []config.Target {
	targets := make([]config.Target, 0, len(models))

	for _, model := range models {
		// Normalize name for target (replace slashes with hyphens)
		targetName := NormalizeModelName(model.ID)

		targets = append(targets, config.Target{
			Name:   targetName,
			URL:    baseURL,
			Model:  model.ID,
			APIKey: apiKey,
		})
	}

	return targets
}

// NormalizeModelName converts model IDs to valid target names
// e.g., "unsloth/gpt-oss-20b" -> "unsloth-gpt-oss-20b"
func NormalizeModelName(modelID string) string {
	// Replace slashes with hyphens
	normalized := strings.ReplaceAll(modelID, "/", "-")
	// Ensure it doesn't start or end with hyphen
	normalized = strings.Trim(normalized, "-")
	return normalized
}
