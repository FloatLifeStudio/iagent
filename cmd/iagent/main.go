// iagent CMDB 硬件资产采集客户端(one-shot)
// systemd timer 触发:采集 → 组装 → 推送 → 退出
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"iagent/internal/collector"
	"iagent/internal/config"
	"iagent/internal/payload"
	"iagent/internal/push"
)

// 编译期注入版本号:-ldflags "-X main.version=0.1.0"
var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	configPath := flag.String("config", "/etc/iagent/config.yml", "配置文件路径")
	serverURL := flag.String("server", "", "CMDB API 地址,如 http://192.168.201.18:8080")
	token := flag.String("token", "", "API token(预留)")
	flag.Parse()

	cfg, err := config.Load(*configPath, *serverURL, *token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] config: %v\n", err)
		return 1
	}

	// hostname 失败是唯一硬失败:中止,不推送
	osInfo, err := collector.CollectOS()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] collect os: %v\n", err)
		return 1
	}

	// 其余类别独立采集,单字段失败已在 collector 内置 null
	mgmt, _ := collector.CollectMgmt()
	hw, _ := collector.CollectHardware()

	agent := payload.Agent{
		Version:   version,
		Source:    "icmdb",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	p, err := payload.Build(agent, osInfo, &mgmt, &hw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] build payload: %v\n", err)
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
