# iagent 详细设计

> 代码级设计。上游文档:[DEVELOPMENT_PLAN.md](./DEVELOPMENT_PLAN.md)(需求)、[ARCHITECTURE.md](./ARCHITECTURE.md)(架构)、[PAYLOAD_EXAMPLE.md](./PAYLOAD_EXAMPLE.md)(推送体)。
> 本文档是编码的直接依据,代码与文档不一致时以本文档为准并回改文档。

---

## 一、运行模型

**one-shot 进程 + systemd timer**:timer 每 12h 触发一次 service,进程执行 `采集 → 归一化 → 组装 → 推送 → 退出`。

- 不做常驻守护:无内部 ticker、无监督复杂度,`systemctl status iagent` 直接看上次运行结果与退出码
- 推送失败不重试、无本地队列——下个周期全量同步自然自愈
- 唯一硬失败:hostname 采集失败 → 记日志、退出码 1、不推送

```
systemd timer(12h)→ iagent(一次运行)
    ├─ 1. config        加载配置
    ├─ 2. collector     各类别独立采集(单字段失败置 null)
    ├─ 3. normalize     容量归一化
    ├─ 4. payload       组装推送体(强制 null 语义)
    ├─ 5. push          POST /api/v1/devices,解析 result
    └─ exit 0(成功)/ exit 1(hostname 失败)
```

## 二、包结构与关键类型

```
internal/
├── config/      # 配置加载
├── collector/   # 采集(os/mgmt/hardware 七类)
├── normalize/   # 容量归一化
├── payload/     # 推送体结构 + 组装
└── push/        # HTTP 客户端
```

### 2.1 config

```go
type Config struct {
    ServerURL string        // CMDB 地址,如 http://192.168.201.18:8080
    Token     string        // 预留,icmdb 鉴权就绪后零改动接入
    Interval  time.Duration // 默认 12h,仅用于生成 systemd timer,进程自身不用
    Timeout   time.Duration // HTTP 超时,默认 30s
}
```

- 来源优先级:命令行参数 > 配置文件(`/etc/iagent/config.yml`)
- 配置文件示例随仓库分发,真实配置(含 token/IP)在 `.gitignore` 中排除

### 2.2 collector —— 采集模块

**接口与错误语义**(对应失败语义"单字段失败置 null"):

```go
// Collector 采集一个顶层组。返回的 struct 中:
//   - 采集成功的字段有值
//   - 采集失败的字段为 nil(JSON null),不返回 error
// 仅 hostname 失败时返回 error(整个流程中止)。
type Collector[T any] interface {
    Collect() (T, error)
}
```

- 每个类别一个文件,内部采集函数统一签名 `func() (*string, error)` 或 `func() ([]X, error)`,error → nil 映射收敛在 collector 内
- **无外部依赖假设**:优先用 gopsutil 与 /sys、/proc 直读;dmidecode/nvidia-smi 以 `exec.Command` 调用(root 环境已确认)
- **占位值归一化(`normStr`)**:`Unknown` / `None` / `N/A` / `NA` / `[N/A]` / `NULL` / `Not Specified` 等占位输出视为未采集 → nil(采纳 dg-agent 的 `_normalize` 思路)

| 文件 | 采集内容 | 失败处理 |
|---|---|---|
| `os.go` | hostname(`hostname -f`)/ type / version / kernel / virt(`systemd-detect-virt`,兜底 DMI sys_vendor) | **hostname 失败 → error 中止**;其余失败 → null |
| `mgmt.go` | `ipmitool lan print` 解析 MAC/IP/Subnet Mask | ipmitool 不存在或失败 → 字段全 null;BMC IP 未分配输出 `0.0.0.0` → null(含 prefix_length 联动) |
| `nics.go` | gopsutil net(排除 lo),CIDR 解析 IP/prefix | 列表失败 → null;单网卡内字段失败 → null |
| `memory.go` | `dmidecode -t memory`(跳过 `No Module` 空槽) | 失败 → null |
| `cpu.go` | /proc/cpuinfo + `dmidecode -t processor`(跳过未插槽) | 失败 → null |
| `disk.go` | `lsblk -bJ -o NAME,SERIAL,TYPE,MODEL,SIZE,ROTA` JSON 输出(过滤 loop/ram 等 type != disk);type 按 ROTA 推断(1=HDD,0=SSD) | 失败 → null;`manufacturer` 从 model 解析不可靠,按设计置 null |
| `psu.go` | `dmidecode -t 39` System Power Supply;`Status: Not Present` 空槽跳过;容量兼容 `Maximum`/`Max Power Capacity`(dmidecode 3.3 vs 新版) | 失败 → null |
| `gpu.go` | `nvidia-smi --query-gpu=uuid,gpu_name,serial,memory.total,driver_version,pci.bus_id --format=csv,noheader,nounits` | nvidia-smi 不存在或失败 → gpu 整体 null(机器无 GPU 也 null) |

**GPU 解析要点**:`memory.total` 以 `nounits` 拿到 MiB 数值,经 normalize 转为 `size` + `size_unit`;`pci.bus_id` 仅作参考字段;SN 查询字段名是 **`serial`**(不是 `serial_number`,驱动会拒绝该查询,已验证);消费级卡(如 4090D)SN 输出 `[N/A]` → null。

### 2.3 normalize —— 容量归一化

```go
// NormalizeCapacity 输入任意单位字节数(如 MiB),输出 size + size_unit:
//   81920( MiB)→ "80", "GB";  8192(GB)→ "8", "TB"
// 规则:GB/TB 中数值 ≥ 1 的最大单位;结果必须为整数(设备容量均为 2ⁿ);
// 输入无法精确整数化时向上取整并记日志(理论不出现)。
func NormalizeCapacity(bytes int64) (int64, string)
```

- 单测覆盖:1024MB→1GB、16384MB→16GB、81920MB→80GB、8TB→8TB、非 2ⁿ 边界

### 2.4 payload —— 推送体

```go
type Payload struct {
    Agent    Agent     `json:"agent"`
    OS       OS        `json:"os"`
    Mgmt     *Mgmt     `json:"mgmt"`
    Hardware *Hardware `json:"hardware"`
}

type Agent struct {
    Version   string `json:"version"`   // 编译期注入 -ldflags "-X main.version=..."
    Source    string `json:"source"`    // 固定 "icmdb"
    Timestamp string `json:"timestamp"` // RFC3339,统一 UTC(time.Now().UTC())
}
```

- **null 语义强制**:所有"可失败"字段用指针类型(`*string` / `*int`),nil 即 JSON null;`os.hostname` 为 string(非指针,必填)
- 列表字段(nics/memory/cpus/disks/psus/gpu)失败 → nil slice → JSON null(依赖服务端 nics/mgmt 改 Optional)
- 组装时校验:hostname 非空,否则拒绝生成 payload

### 2.5 push —— HTTP 客户端

```go
func (c *Client) Push(p *payload.Payload) (Result, error)

type Result struct {
    Result           string `json:"result"`            // created / unchanged / diff_created
    DeviceID         int    `json:"device_id"`
    PendingChangeID  *int   `json:"pending_change_id"`
}
```

- POST `{ServerURL}/api/v1/devices`,`Content-Type: application/json`
- token 预留:`Authorization: Bearer <token>`,Token 为空则不带该 header
- 处理:2xx → 解析 result 记日志;4xx/5xx → 记日志返回 error,**不重试**(下周期自愈)
- 预留 token 后,4xx 中 401/403 单独记日志提示鉴权失败

## 三、systemd 单元设计

```ini
# iagent.service — one-shot
[Unit]
Description=iagent CMDB collector
Wants=network-online.target
After=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/iagent --config /etc/iagent/config.yml
User=root
```

```ini
# iagent.timer
[Unit]
Description=Run iagent every 12h

[Timer]
OnCalendar=*-*-* 03,15:00:00     # 错开整点,可配
Persistent=true                   # 错过的触发开机后补跑

[Install]
WantedBy=timers.target
```

- `Persistent=true`:机器关机错过的周期开机后补跑一次,减少"疑似下线"误判
- 触发时刻默认 03:00/15:00(避开整点与工作高峰),由 Ansible 模板按配置生成

## 四、Ansible 角色设计

```
ansible/roles/iagent/
├── tasks/main.yml       # 1. 分发二进制(版本化命名)2. 渲染配置 3. 安装 systemd 单元 4. enable timer
├── templates/
│   ├── config.yml.j2    # server/token/interval
│   ├── iagent.service.j2
│   └── iagent.timer.j2  # OnCalendar 由 interval 生成
└── handlers/main.yml    # 配置变更 → daemon-reload
```

- **幂等**:二进制按版本号比对,相同跳过;配置 changed → handler reload
- **升级**:替换二进制 + 重启 timer,`agent.version` 随 `-ldflags` 版本号上报
- **回滚**:保留上一版本二进制,`iagent_version` 变量切回

## 五、错误处理与日志

| 场景 | 行为 | 退出码 |
|---|---|---|
| hostname 采集失败 | 记日志,不推送 | 1 |
| 单字段采集失败 | 字段置 null,继续 | 0 |
| 推送 2xx | 记 result(created/unchanged/diff_created) | 0 |
| 推送 4xx/5xx/超时 | 记日志,不重试 | 1 |
| 配置错误 | 启动即失败,记日志 | 1 |

- 日志输出到 stdout/stderr(journald 接管),格式:`2026-09-14T07:15:01Z [INFO] push ok: result=unchanged device_id=1`
- 日志级别:INFO(正常流程)/ WARN(字段失败置 null)/ ERROR(中止性失败)

## 六、设计要点回顾(拷问结论落点)

| 设计点 | 结论来源 |
|---|---|
| one-shot 进程(修正"常驻"叫法) | systemd timer 标准模式 |
| 单字段失败置 null,hostname 失败中止 | 需求确认 2026-09-13 |
| 指针类型强制 null 语义 | payload 层最终防线 |
| 占位值归一化(Unknown/NULL/[N/A]... → null) | dg-agent 设计 + 生产验证 2026-09-14 |
| 空槽跳过(内存 No Module / 电源 Not Present) | 生产验证 2026-09-14 |
| GPU 身份 uuid,pcie_id 仅参考 | 需求确认 |
| GPU SN 查询字段 `serial`(非 serial_number) | dg-agent 设计,驱动 610.43.02 验证 |
| 容量归一化在客户端 | 需求确认(规则见 2.3) |
| token 预留(header 位) | icmdb 鉴权待实现 |
| full_sync 暂不推送 | 启用时固定置 true,避开生产高峰 |

## 七、生产验证记录(2026-09-14)

163 / 161 两台生产 GPU 服务器全链路验证通过(采集 → 组装 → 推送本地哑服务,未触生产数据):

- **裸金属检测正确**:`systemd-detect-virt` 直读,两台均报 `bare_metal`
- **机器差异已覆盖**:dmidecode 3.3(SMBIOS 3.6.0)的 `NULL` 占位、`Max Power Capacity` 键名、未插 PSU 空槽;BMC 未配置时的 `0.0.0.0`;消费级 GPU 无 SN
- **结构一致性**:两台 payload 顶层与 hardware 层字段完全一致(同名必同义)
- **遗留项**:`disks[].manufacturer` 置 null 留待更多机器验证;163 未装 ipmitool(`mgmt` null);nics 含 `docker0`/`veth*` 虚拟网卡,是否过滤待定
- **部署坑**:上传前必须先重新构建(`dist/` 下 gz 可能落后于源码),构建命令见 [USAGE.md](./USAGE.md)
