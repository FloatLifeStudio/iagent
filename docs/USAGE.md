# iagent 使用手册

> 构建配置、部署安装、托管运维、运行验证、问题排查。
> 设计与需求背景见 [DEVELOPMENT_PLAN.md](./DEVELOPMENT_PLAN.md) / [ARCHITECTURE.md](./ARCHITECTURE.md)。

---

## 一、本地构建

```bash
# 依赖:Go 1.26+
go build ./...

# 单元测试
go test ./...

# 构建带版本号的发布二进制(版本号注入 agent.version)
CGO_ENABLED=0 go build -ldflags "-X main.version=0.1.0" -o dist/iagent-0.1.0 ./cmd/iagent

# 需要经网络分发时打压缩包(注意:gzip 会移除原二进制)
gzip -f -9 dist/iagent-0.1.0
```

`CGO_ENABLED=0` 产出纯静态二进制,可直接拷贝到目标机(含老系统)运行,无运行时依赖。

> **坑**:分发前必须先重新构建再上传,不要直接用 `dist/` 下可能落后于源码的旧二进制。上传后可用 `md5sum` 与本地比对确认版本一致。

## 二、配置

配置文件位于 `/etc/iagent/config.yml`,YAML 格式。**完整配置文件**(含全部选项与注释):

```yaml
# /etc/iagent/config.yml
# iagent 配置文件。优先级:命令行参数 > 本文件 > 内置默认值。
# 四个选项中只有 server_url 必填,其余可省略(省略走默认值)。

# CMDB API 地址(必填)
# iagent 推送目标,即 icmdb 的 ingest 接口根地址。
# 留空或缺失会启动即失败(exit 1),不会推送。
server_url: http://192.168.201.18:8080

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
```

### 2.1 命令行参数

| 参数 | 默认 | 说明 |
|---|---|---|
| `--config` | `/etc/iagent/config.yml` | 配置文件路径;传 `/dev/null` 可跳过文件(仅用参数 + 默认值) |
| `--server` | - | CMDB API 地址,**覆盖**配置文件的 `server_url` |
| `--token` | - | API token,**覆盖**配置文件的 `token` |

### 2.2 配置校验行为

- `server_url` 为空/缺失 → 启动即失败:`[ERROR] config: server_url is required`,退出码 1
- `interval` / `timeout` ≤ 0 → 自动回落默认值(12h / 30s),不报错
- 配置文件不存在 → 读文件报错退出;确认路径后再启动
- YAML 格式错误 → 解析报错退出,错误信息带文件路径

## 三、手动部署(单机)

目标机要求:Linux + systemd、root 权限。**主机名(`hostname -f`)是匹配键,部署前确认已正确设置。**

```bash
# 1. 分发二进制(控制机 → 目标机)
scp dist/iagent-0.1.0 user@target:/tmp/

# 2. 安装二进制与配置(目标机上,root)
sudo cp /tmp/iagent-0.1.0 /usr/local/bin/iagent
sudo chmod +x /usr/local/bin/iagent
sudo mkdir -p /etc/iagent
sudo vim /etc/iagent/config.yml        # 按第二节填写

# 3. 手动验证一次(不等 timer,确认能采能推)
sudo /usr/local/bin/iagent --config /etc/iagent/config.yml
# 成功输出:[INFO] push ok: result=created device_id=1 pending_change_id=<nil>

# 4. 安装 systemd 单元(仓库 systemd/ 目录)
sudo cp systemd/iagent.service systemd/iagent.timer /etc/systemd/system/
sudo systemctl daemon-reload

# 5. 启用定时器
sudo systemctl enable --now iagent.timer
```

## 四、Ansible 批量部署

```bash
cd ansible

# 1. 准备清单(复制示例并填写真实 IP/连接方式)
cp inventory.example inventory.yml
vim inventory.yml

# 2. 构建发布二进制(版本号与清单中 iagent_version 一致)
cd .. && CGO_ENABLED=0 go build -ldflags "-X main.version=0.1.0" -o dist/iagent-0.1.0 ./cmd/iagent

# 3. 部署
cd ansible && ansible-playbook -i inventory.yml playbook.yml
```

- **幂等**:二进制版本相同则跳过,可重复执行
- **升级**:改 `iagent_version` + 重新构建 + 重跑 playbook
- **回滚**:清单中把 `iagent_version` 改回旧版本号(需保留旧二进制)
- 真实清单(IP/token)在 `.gitignore` 中排除,不入仓库

清单(`inventory.example`)变量说明:

```ini
[cmdb_targets]                # 目标机列表,每台声明 ansible_user
192.168.10.101 ansible_user=root

[cmdb_targets:vars]           # 全组变量
iagent_version=0.1.0                        # 部署的二进制版本号
iagent_server_url=http://192.168.201.18:8080  # CMDB API 地址(渲染进 config.yml)
iagent_token=                               # token 预留位
iagent_on_calendar=*-*-* 03,15:00:00        # timer 触发时刻(由 interval 决定)
```

## 五、托管与日常运维

iagent 是 **one-shot 进程**,由 systemd timer 托管:timer 每 12h 触发一次 service,进程采集 → 推送 → 退出。没有常驻守护进程,所有状态通过 systemd 查看。

### 5.1 日常查看

```bash
# 查看上次运行结果(oneshot 直接看状态与退出码)
systemctl status iagent.service

# 查看日志(journald 接管 stdout/stderr)
journalctl -u iagent.service -n 50 --no-pager
journalctl -u iagent.service -f          # 跟踪

# 查看下次触发时间
systemctl list-timers iagent.timer
```

### 5.2 启停操作

```bash
# 手动立即跑一次(不等 timer)
sudo systemctl start iagent.service

# 暂停定时采集(不卸载)
sudo systemctl stop iagent.timer

# 恢复
sudo systemctl start iagent.timer

# 改配置后无需重启任何东西:one-shot 每次触发都重新读配置文件
# (仅修改 systemd 单元本身才需要 daemon-reload)
sudo vim /etc/iagent/config.yml
```

### 5.3 升级与卸载

```bash
# 升级:替换二进制即可,下次触发自动生效(agent.version 随 -ldflags 上报)
sudo cp dist/iagent-0.1.1 /usr/local/bin/iagent

# 卸载
sudo systemctl disable --now iagent.timer
sudo rm /etc/systemd/system/iagent.service /etc/systemd/system/iagent.timer
sudo systemctl daemon-reload
sudo rm /usr/local/bin/iagent
sudo rm -r /etc/iagent
```

### 5.4 托管行为说明

| 行为 | 说明 |
|---|---|
| 触发时刻 | 默认 03:00 / 15:00(`OnCalendar`,避开整点与工作高峰),由 Ansible 模板按 `interval` 生成 |
| `Persistent=true` | 机器关机错过的周期,开机后补跑一次,减少"疑似下线"误判 |
| 失败自愈 | 推送失败不重试、无本地队列;下个周期全量同步自然自愈 |
| CMDB 侧状态 | `iagent.timer` 持续启用即视为在线;CMDB 超过下线阈值(3 天)未收到推送则判疑似下线 |

## 六、运行与验证

推送结果(也写入 CMDB):

| result | 含义 |
|---|---|
| `created` | CMDB 中无该主机,首次创建 |
| `unchanged` | 字段级无差异,仅刷新 last_pushed_at |
| `diff_created` | 有差异,进待裁决,在 CMDB UI 裁决 |

## 七、生产验证情况

2026-09-14 已在两台生产 GPU 服务器完成全链路验证:采集 → 组装 → 推送全字段正常,占位值/空槽/BMC 未配置等边界场景符合设计语义。详细结论见 [DESIGN.md](./DESIGN.md) 第七节。

**注意**:单字段采集失败时推送体对应字段为 `null`(不是省略、不是空列表),CMDB 侧保留已有数据不进裁决——这是设计语义,不是 bug。整体性失败(hostname 缺失)则整个推送体不发,该周期视为未推送。

## 八、问题排查

| 现象 | 排查 |
|---|---|
| 启动即 `config: server_url is required` | 配置文件缺 `server_url` 或 `--config` 路径错误;按第二节补齐 |
| 启动即 `config: read config "..."` | 配置文件不存在;确认 `--config` 路径 |
| 启动即 `config: parse config` | YAML 格式错误;检查缩进与冒号 |
| `push: HTTP 4xx` | 校验失败(缺 hostname)或鉴权失败(token);查看日志中的具体错误 |
| `push: connection refused / timeout` | CMDB 不可达;检查 `server_url` 与网络 |
| `memory.slots` / `cpus` / `psus` / `chassis_serial_number` 为 null | 这些字段需要 **root**;确认 service 以 root 运行、dmidecode 可用(`sudo dmidecode -t memory`) |
| `gpu.slots` 为 null | 确认机器有 GPU 且 `nvidia-smi --query-gpu=uuid,gpu_name,serial --format=csv,noheader` 可用(SN 字段名是 `serial`,不是 `serial_number`) |
| `gpu.slots[].serial_number` 为 null | 消费级卡(如 GeForce 4090D)驱动输出 `[N/A]`,无 SN,置 null 属预期 |
| `mgmt` 全 null | 确认 `ipmitool lan print` 可用(需 BMC/带外通道;未装 ipmitool 也会全 null) |
| `mgmt.ip` 为 null | BMC 未分配 IP 时 ipmitool 输出 `0.0.0.0`,按设计置 null;`mac` 有值说明带外通道正常 |
| 采集到的字段出现 `"NULL"` 字符串 | 部分机器 SMBIOS 用字面 `NULL` 占位;新版已归一化为 null,若出现说明二进制过旧,重新构建上传 |
| 定时器没触发 | `systemctl list-timers` 看状态;`Persistent=true` 时关机错过的周期开机后补跑 |
| 改了配置没生效 | one-shot 每次触发重新读配置,无需重启;确认改的是 `/etc/iagent/config.yml` 且下次触发已到 |
