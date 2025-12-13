package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration structure
type Config struct {
	Environments map[string]Environment `yaml:"environments"`
	Defaults     Defaults               `yaml:"defaults"`
	Prometheus   PrometheusConfig       `yaml:"prometheus"`
}

// Environment represents a deployment environment (e.g., develop, staging)
type Environment struct {
	Targets []Target `yaml:"targets"`
}

// Target represents an LLM endpoint to benchmark
type Target struct {
	Name      string `yaml:"name"`
	URL       string `yaml:"url"`
	Model     string `yaml:"model"`
	APIKey    string `yaml:"api_key,omitempty"`

	// Per-target overrides (optional)
	Profile    string `yaml:"profile,omitempty"`
	Rate       *int   `yaml:"rate,omitempty"`
	MaxSeconds *int   `yaml:"max_seconds,omitempty"`
}

// Defaults contains default benchmark settings
type Defaults struct {
	Profile    string `yaml:"profile"`
	Rate       int    `yaml:"rate"`
	Interval   int    `yaml:"interval"`    // seconds between benchmark runs
	MaxSeconds int    `yaml:"max_seconds"` // duration per run
	MaxTokens  int    `yaml:"max_tokens"`
	DataSpec   string `yaml:"data_spec"`   // e.g., "prompt_tokens=256,output_tokens=128"
}

// PrometheusConfig contains Prometheus exporter settings
type PrometheusConfig struct {
	Port int `yaml:"port"`
}

// Load reads and parses the config file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Apply defaults
	if cfg.Defaults.Profile == "" {
		cfg.Defaults.Profile = "constant"
	}
	if cfg.Defaults.Rate == 0 {
		cfg.Defaults.Rate = 1
	}
	if cfg.Defaults.Interval == 0 {
		cfg.Defaults.Interval = 60
	}
	if cfg.Defaults.MaxSeconds == 0 {
		cfg.Defaults.MaxSeconds = 30
	}
	if cfg.Defaults.MaxTokens == 0 {
		cfg.Defaults.MaxTokens = 100
	}
	if cfg.Defaults.DataSpec == "" {
		cfg.Defaults.DataSpec = "prompt_tokens=256,output_tokens=128"
	}
	if cfg.Prometheus.Port == 0 {
		cfg.Prometheus.Port = 9090
	}

	return &cfg, nil
}

// GetInterval returns the interval duration
func (c *Config) GetInterval() time.Duration {
	return time.Duration(c.Defaults.Interval) * time.Second
}

// GetRate returns the effective rate for a target
func (t *Target) GetRate(defaults Defaults) int {
	if t.Rate != nil {
		return *t.Rate
	}
	return defaults.Rate
}

// GetMaxSeconds returns the effective max_seconds for a target
func (t *Target) GetMaxSeconds(defaults Defaults) int {
	if t.MaxSeconds != nil {
		return *t.MaxSeconds
	}
	return defaults.MaxSeconds
}

// GetProfile returns the effective profile for a target
func (t *Target) GetProfile(defaults Defaults) string {
	if t.Profile != "" {
		return t.Profile
	}
	return defaults.Profile
}
