// Package config loads configuration: CLI flags > config file > defaults
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ErrNoServerURL means the required server_url is missing
var ErrNoServerURL = errors.New("server_url is required")

type Config struct {
	ServerURL string        `yaml:"server_url"`
	Token     string        `yaml:"token"`    // reserved, wired in once icmdb auth is ready
	Interval  time.Duration `yaml:"interval"` // push interval, used to generate the systemd timer
	Timeout   time.Duration `yaml:"timeout"`  // HTTP timeout
}

// Load loads configuration. Empty path skips the file; CLI flags override the
// file; defaults as fallback
func Load(path, serverURL, token string) (Config, error) {
	cfg := Config{Interval: 12 * time.Hour, Timeout: 30 * time.Second}
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return Config{}, fmt.Errorf("read config %q: %w", path, err)
		}
		if err := yaml.Unmarshal(b, &cfg); err != nil {
			return Config{}, fmt.Errorf("parse config %q: %w", path, err)
		}
	}
	if serverURL != "" {
		cfg.ServerURL = serverURL
	}
	if token != "" {
		cfg.Token = token
	}
	if cfg.ServerURL == "" {
		return Config{}, ErrNoServerURL
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 12 * time.Hour
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	return cfg, nil
}
