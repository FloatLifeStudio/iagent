// Package config 配置加载:命令行参数 > 配置文件 > 默认值。
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ErrNoServerURL 缺少必填的 server_url。
var ErrNoServerURL = errors.New("server_url is required")

type Config struct {
	ServerURL string        `yaml:"server_url"`
	Token     string        `yaml:"token"`    // 预留,icmdb 鉴权就绪后接入
	Interval  time.Duration `yaml:"interval"` // 推送周期,用于生成 systemd timer
	Timeout   time.Duration `yaml:"timeout"`  // HTTP 超时
}

// Load 加载配置。path 为空跳过文件;命令行参数覆盖配置文件;默认值兜底。
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
