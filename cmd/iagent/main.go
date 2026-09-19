// iagent CMDB 硬件资产采集客户端(one-shot)
// systemd timer 触发:采集 → 组装 → 推送 → 退出
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

// 编译期注入版本号:-ldflags "-X main.version=0.1.0"
var version = "dev"

// defaultConfigTemplate 生成的默认配置模板,含全部选项与注释。
// server_url 留空:首次正式运行会报错提示填写。
const defaultConfigTemplate = `# iagent 配置文件。优先级:命令行参数 > 本文件 > 内置默认值。
# 四个选项中只有 server_url 必填,其余可省略(省略走默认值)。

# CMDB API 地址(必填)
# iagent 推送目标,即 icmdb 的 ingest 接口根地址,如 http://192.168.201.18:8080。
# 留空或缺失会启动即失败(exit 1),不会推送。
server_url: ""

# API token(可选,默认 "")
# 鉴权预留位:icmdb 的 token 鉴权就绪后在此填入,
# 请求头为 Authorization: Bearer <token>;为空则不带该 header,零改动接入。
token: ""

# 推送周期(可选,默认 12h)
# 仅用于 Ansible 生成 systemd timer 的触发时刻,进程自身不读此值
# (one-shot 进程由 timer 驱动,跑完即退)。
# 支持 Go duration 格式:30m / 1h / 6h / 12h / 24h。
# 建议 12h:与 CMDB 下线阈值 3 天咬合(≈ 6 次推送/阈值周期)。
interval: 12h

# HTTP 超时(可选,默认 30s)
# 单次 POST 推送的超时时间。超时按失败处理:记日志、不重试,
# 下个周期全量同步自然自愈。
timeout: 30s
`

func main() {
	os.Exit(run())
}

func run() int {
	configPath := flag.String("config", "/etc/iagent/config.yml", "配置文件路径")
	serverURL := flag.String("server", "", "CMDB API 地址,如 http://192.168.201.18:8080")
	token := flag.String("token", "", "API token(预留)")
	initConfig := flag.Bool("init", false, "生成默认配置文件(已存在则不覆盖)")
	printOnly := flag.Bool("print", false, "只采集并打印 JSON 到 stdout,不推送、不需要配置文件(测试用)")
	flag.Parse()

	if *initConfig {
		return writeDefaultConfig(*configPath)
	}

	// --print 测试模式:不读配置文件,采集后直接打印
	if *printOnly {
		return printPayload()
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

// collectAndBuild 采集并组装推送体。hostname 失败是唯一硬失败。
func collectAndBuild() (payload.Payload, error) {
	// hostname 失败是唯一硬失败:中止,不推送
	osInfo, err := collector.CollectOS()
	if err != nil {
		return payload.Payload{}, fmt.Errorf("collect os: %w", err)
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
		return payload.Payload{}, fmt.Errorf("build payload: %w", err)
	}
	return p, nil
}

// printPayload --print 测试模式:采集后直接打印 JSON,不推送、不需要配置文件。
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

// writeDefaultConfig 生成默认配置文件(--init 子命令)。
// 幂等:文件已存在则不覆盖,提示后退出。
func writeDefaultConfig(path string) int {
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("[INFO] config already exists: %s (不覆盖;如需重新生成请先删除)\n", path)
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
	fmt.Printf("[INFO] 填好 server_url 后运行 iagent 正式采集\n")
	return 0
}
