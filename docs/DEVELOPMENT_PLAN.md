# iagent 采集客户端开发计划

> 为 icmdb 设计的硬件资产采集客户端
> 总体架构与目录结构见 [ARCHITECTURE.md](./ARCHITECTURE.md),代码级设计见 [DESIGN.md](./DESIGN.md)。

---

## 一、项目背景与目标

CMDB v2 链路为 `采集 → JSON → CMDB API → 存储 → UI`,系统只管存储和查询,不做采集。
本项目的目标:

> **开发一个跑在被管服务器上的常驻采集客户端,定时采集硬件/系统信息,按统一规范组装 JSON 推送到 CMDB API。**

- **S(具体)**:采集硬件六类 + GPU + OS 核心字段,推送到 `POST /api/v1/devices`
- **M(可衡量)**:全链路跑通——目标机定时推送、CMDB 正确入库、冲突进裁决;12h 周期稳定运行
- **A(可实现)**:Go 单文件 + gopsutil + nvidia-smi,root 运行,Ansible 部署
- **R(相关)**:CMDB 的数据质量上限由采集端决定;agent_version 字段支撑后续版本管理台账
- **T(时限)**:推送周期 12h(可配),与 CMDB 下线阈值 3 天咬合(12h ≈ 6 次推送/阈值周期)

## 二、需求总结(已确认的决策)

### 2.1 形态与部署

| 决策点 | 结论 |
|---|---|
| 语言/形态 | **Go 编译单文件程序,one-shot 模式**(否决 shell 与 Rust:维护成本与兼容性考量;systemd timer 触发,采集→推送→退出) |
| 运行权限 | **root**(内存槽位 SMBIOS、电源 dmidecode、整机 SN 均依赖特权,无 root 路线已验证走不通) |
| 部署 | **Ansible** 下发安装/升级/配置(常驻 agent 模式) |
| 定时驱动 | 本机 **systemd timer**,12h 一次,间隔可配 |
| 鉴权 | CMDB 后续实现 token,客户端预留配置位 |

### 2.2 采集范围

| 类别 | 字段 | 来源 |
|---|---|---|
| os | hostname(匹配键)/ type / version / kernel / virt(裸金属/虚拟机类型) | `uname -n`、`/etc/os-release`、`uname -r`、`systemd-detect-virt` |
| agent | version / source / timestamp | 采集器自身 |
| mgmt | mac / ip / prefix_length | IPMI/BMC(带外通道) |
| hardware.nics | name(身份)/ mac / ips[ip, prefix_length] | `/sys/class/net`、`ip` |
| hardware.memory.slots | slot(身份)/ manufacturer / part_number / type / size+size_unit / speed_mts / serial_number | `dmidecode` |
| hardware.cpus | slot(身份)/ model | `/proc/cpuinfo`、`dmidecode` |
| hardware.disks | serial_number(身份)/ type / manufacturer / model / size+size_unit | `lsblk`、`/sys/block`、`dmidecode` |
| hardware.psus | serial_number(身份)/ manufacturer / model / max_power_w | `dmidecode` |
| hardware.gpu.slots | **uuid(身份)**/ name / serial_number / size+size_unit / driver_version / pcie_id | `nvidia-smi --query-gpu`(SN 字段名为 `serial`) |

- GPU 身份用 **uuid**:恒有且跨重启稳定;`pcie_id` 不稳定仅作参考;部分型号无 SN
- `full_sync` 暂不启用(removed 候删暂不可用);启用时由采集端固定置 true,并避开生产高峰(首次启用会全量推进待裁决)

### 2.3 失败语义(核心规则)

> **单字段采集失败 → 推 null,不省略 key、不省略组。唯一例外:hostname 失败 → 整个推送体不推(无匹配键)。**

- 服务端现有语义已覆盖(None = 未采集,保留现状,不进 diff),零服务端改动即可工作
- 服务端需将 `nics`、`mgmt` 改为 Optional,否则推显式 null 会 422

### 2.4 容量归一化规则

> 客户端归一化:统一 `size` + `size_unit` 格式;单位取 **GB/TB 中数值 ≥ 1 的最大单位**;数值必须为整数(设备容量均为 2ⁿ,不出现小数);mb 一律不再出现。

示例:1024MB → `1 GB`;16384MB → `16 GB`;81920MB(H100)→ `80 GB`;8TB → `8 TB`。

### 2.5 推送体结构(基准示例)

顶层四个组:**`agent`**(信封元数据)/ **`os`**(hostname + OS 信息)/ **`mgmt`**(带外管理口)/ **`hardware`**(全部硬件)。完整示例见 [PAYLOAD_EXAMPLE.md](./PAYLOAD_EXAMPLE.md)。

```json
{
  "agent":  {"version": "0.1.0", "source": "icmdb",
             "timestamp": "2026-09-14T07:14:55Z"},
  "os":     {"hostname": "gpu-node-01", "type": "linux",
             "version": "Ubuntu 22.04.5 LTS", "kernel": "5.15.0-186-generic"},
  "mgmt":   {"mac": "AA:BB:CC:DD:EE:01", "ip": "192.168.10.101", "prefix_length": 24},
  "hardware": {"chassis_serial_number": "SN0000000001",
               "nics": [...], "memory": {"slots": [...]}, "cpus": [...],
               "disks": [...], "psus": [...], "gpu": {"slots": [...]}}
}
```

命名原则:**同名必同义**——整机 SN 命名 `chassis_serial_number`,与条目级 `serial_number` 区分。

## 三、开发阶段

### 阶段 0:技术验证(先行,1~2 天)

| 任务 | 产出 | 验收标准 |
|---|---|---|
| gopsutil 原型 | 在目标机实际运行的原型脚本 | 逐字段确认可得性(root 下),与 2.2 表对照 |
| nvidia-smi 字段确认 | `--query-gpu` 实际输出清单 | uuid/SN/driver/显存字段真实值确认 |
| 目标机环境摸底 | 系统清单(CentOS 7?systemd 版本?) | Go 静态二进制兼容性确认 |

### 阶段 1:采集器 v1(核心)✅ 已完成(2026-09-14)

> 生产验证:163/161 两台 GPU 服务器全链路跑通(采集 → 组装 → 推送),占位值归一化、空槽跳过、BMC 未配置、GPU 混插等边界场景符合设计;详见 [DESIGN.md](./DESIGN.md) 第七节。

| 任务 | 产出 | 验收标准 |
|---|---|---|
| 项目骨架 | Go module、配置加载(服务器地址/token/间隔可配) | 配置文件 + 参数两种方式可用 |
| 采集模块 | os / agent / mgmt / hardware 各类独立采集,单字段失败置 null | 按 2.2 表逐类实现;失败语义符合 2.3 |
| 容量归一化 | GB/TB 归一化函数 | 符合 2.4 规则,含边界用例单测 |
| 推送模块 | POST /api/v1/devices,解析 result 三分支 | created/unchanged/diff_created 正确处理;推送失败下周期自愈(无本地队列) |
| 定时驱动 | systemd service + timer 单元 | 12h 触发可配,`list-timers` 可查,日志进 journald |
| 单元测试 | 各采集/归一化/推送模块测试 | `go test ./...` 通过 |

### 阶段 2:部署(Ansible)

| 任务 | 产出 | 验收标准 |
|---|---|---|
| playbook | 安装/升级/回滚、配置下发、幂等 | 指定服务器批量部署成功;重复执行无副作用 |
| 版本管理 | 二进制版本化发布 | 升级后 agent.version 上报正确 |

### 阶段 3:服务端对接(icmdb 侧)

| 任务 | 产出 | 优先级 |
|---|---|---|
| GPU/OS schema + hostname 路径 | DevicePush/DeviceOut/diff/裁决/UI,匹配键改 `os.hostname` | P0 |
| nics/mgmt 改 Optional | 推 null 不 422 | P0 |
| token 鉴权 | 推送接口共享 token | P1 |
| GPU/OS 上线日首推处理 | 新 schema 上线时 pending 泛滥的配套(首推自动采纳或人工清理) | P1 |

### 阶段 4:联调验收

1. 目标机 12h 周期推送,CMDB 数据正确入库(字段级比对)
2. 制造字段变化(换内存/改 IP)→ 进待裁决,裁决生效写变更历史
3. 采集失败场景(撤权模拟)→ 对应字段为 null,库中数据保留
4. 全 fleet 部署,agent_version 台账可查

## 四、风险与待验证项

| 风险 | 影响 | 应对 |
|---|---|---|
| gopsutil 字段可得性不足(电源/内存槽位) | 采集不完整 | 阶段 0 先行验证;缺口用 dmidecode 直读兜底 |
| GPU/OS schema 上线日 pending 泛滥 | 裁决队列一次性堆积 N 条 | 服务端上线新 schema 时配套处理 |
| API 无鉴权窗口期 | 伪造推送/误删 | 尽快落地 token(P1) |
| 主机改名/克隆 | 新设备残留/冲突 | 已知语义,人工裁决,不防御 |
| CentOS 7 老系统兼容 | 部署失败 | 阶段 0 摸底;Go 静态编译规避 |

## 五、里程碑

- **M1**:阶段 0 完成,字段可得性报告 ✅(2026-09-13)
- **M2**:采集器 v1 + 单测通过 + 生产机器验证 ✅(2026-09-14,163/161 全链路)
- **M3**:Ansible 批量部署上线(预计 1~2 天)
- **M4**:服务端对接完成,全链路验收(依赖 icmdb 侧排期)
