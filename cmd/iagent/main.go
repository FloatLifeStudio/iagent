// iagent CMDB hardware asset collection client (one-shot).
// Triggered by systemd timer: collect -> assemble -> push -> exit.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"iagent/internal/collector"
	"iagent/internal/config"
	"iagent/internal/payload"
	"iagent/internal/push"
)

// Version is injected at build time: -ldflags "-X main.version=0.1.0"
var version = "dev"

// defaultConfigTemplate is written by --init, with all options documented.
// server_url is left empty so the first real run fails with a clear hint.
const defaultConfigTemplate = `# iagent config file. Priority: CLI flags > this file > built-in defaults.
# Only server_url is required; all other options fall back to defaults.

# CMDB API address (required)
# Push target, the root URL of the icmdb ingest API, e.g. http://192.168.201.18:8080.
# Empty or missing fails fast on startup (exit 1), nothing is pushed.
server_url: ""

# API token (optional, default "")
# Reserved for auth: fill in once icmdb token auth is ready.
# Sent as "Authorization: Bearer <token>"; omitted when empty, zero-change integration.
token: ""

# Push interval (optional, default 12h)
# Only used by Ansible to generate the systemd timer schedule; the process itself
# never reads it (one-shot process is driven by the timer, exits after each run).
# Accepts Go duration format: 30m / 1h / 6h / 12h / 24h.
# 12h recommended: aligns with the CMDB offline threshold of 3 days (~6 pushes/cycle).
interval: 12h

# HTTP timeout (optional, default 30s)
# Timeout for a single POST push. Timeout counts as failure: log, no retry,
# the next cycle full sync self-heals.
timeout: 30s
`

func main() {
	os.Exit(run())
}

// usage prints help (shown when invoked without any arguments).
func usage() {
	fmt.Print(`iagent CMDB hardware asset collection client (one-shot)

Usage:
  iagent                     show this help
  iagent --print             test mode: collect and print JSON to stdout (no push)
  iagent --init              generate default config file (default /etc/iagent/config.yml, never overwrites)
  iagent --config <path>     collect and push to CMDB (config file must contain server_url)

Flags:
`)
	flag.PrintDefaults()
	fmt.Print(`
Examples:
  iagent --print                              # collect once, print JSON directly
  iagent --init                               # generate config template, then fill in server_url
  iagent --config /etc/iagent/config.yml      # real collect and push (invoked by systemd timer)
`)
}

func run() int {
	configPath := flag.String("config", "", "config file path (default /etc/iagent/config.yml)")
	serverURL := flag.String("server", "", "CMDB API address, e.g. http://192.168.201.18:8080")
	token := flag.String("token", "", "API token (reserved)")
	initConfig := flag.Bool("init", false, "generate default config file (never overwrites)")
	printOnly := flag.Bool("print", false, "collect and print JSON to stdout, no push, no config file needed (test mode)")
	flag.Usage = usage
	flag.Parse()

	// Default (no arguments): show help
	if !*initConfig && !*printOnly && *configPath == "" && *serverURL == "" && *token == "" {
		flag.Usage()
		return 0
	}

	if *initConfig {
		if *configPath == "" {
			*configPath = "/etc/iagent/config.yml"
		}
		return writeDefaultConfig(*configPath)
	}

	// --print test mode: skip config file, collect then print
	if *printOnly {
		return printPayload()
	}

	if *configPath == "" {
		*configPath = "/etc/iagent/config.yml"
	}

	cfg, err := config.Load(*configPath, *serverURL, *token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] config: %v\n", err)
		return 1
	}

	p, err := collectAndBuild()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		return 1
	}

	result, err := push.NewClient(cfg.ServerURL, cfg.Token).Push(&p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] push: %v\n", err)
		return 1
	}
	fmt.Printf("[INFO] push ok: result=%s device_id=%d pending_change_id=%v\n",
		result.Result, result.DeviceID, result.PendingChangeID)
	return 0
}

// collectAndBuild collects and assembles the payload. hostname failure is the only hard failure.
func collectAndBuild() (payload.Payload, error) {
	// hostname failure is the only hard failure: abort, do not push
	osInfo, err := collector.CollectOS()
	if err != nil {
		return payload.Payload{}, fmt.Errorf("collect os: %w", err)
	}

	// Other categories are collected independently; single-field failures are already
	// mapped to null inside the collector
	mgmt, _ := collector.CollectMgmt()
	hw, _ := collector.CollectHardware()

	agent := payload.Agent{
		Version:   version,
		Source:    "icmdb",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	p, err := payload.Build(agent, osInfo, &mgmt, &hw)
	if err != nil {
		return payload.Payload{}, fmt.Errorf("build payload: %w", err)
	}
	return p, nil
}

// printPayload implements the --print test mode: collect then print JSON, no push, no config file.
func printPayload() int {
	p, err := collectAndBuild()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		return 1
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] marshal payload: %v\n", err)
		return 1
	}
	fmt.Println(string(b))
	return 0
}

// writeDefaultConfig generates the default config file (--init subcommand).
// Idempotent: refuses to overwrite an existing file, exits with a hint.
func writeDefaultConfig(path string) int {
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("[INFO] config already exists: %s (not overwritten; delete it first to regenerate)\n", path)
		return 0
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] mkdir %s: %v\n", dir, err)
		return 1
	}
	if err := os.WriteFile(path, []byte(defaultConfigTemplate), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] write config: %v\n", err)
		return 1
	}
	fmt.Printf("[INFO] config written: %s\n", path)
	fmt.Printf("[INFO] fill in server_url, then run iagent to start collecting\n")
	return 0
}
