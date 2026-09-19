# iagent 总体架构与项目目录结构

> 采集客户端的架构设计与目录结构说明需求与计划见 [DEVELOPMENT_PLAN.md](./DEVELOPMENT_PLAN.md),推送体基准见 [PAYLOAD_EXAMPLE.md](./PAYLOAD_EXAMPLE.md),代码级设计见 [DESIGN.md](./DESIGN.md)

---

## 一,系统全景

iagent 在 CMDB 链路中的位置:

```
┌─────────────────────────── 被管服务器 × N ───────────────────────────┐
│                                                                       │
│   systemd timer(12h,可配)                                             │
│        │                                                              │
│        ▼                                                              │
│   iagent(Go 单文件 one-shot 程序,root 运行)                           │
│                                                                       │
│   ┌───────────┐    ┌───────────┐    ┌───────────┐    ┌───────────┐    │
│   │ collector │ →  │ normalize │ →  │ payload   │ →  │   push    │    │
│   │  采集模块 │    │ 容量归一化│    │ 组装+null │    │ HTTP 推送 │    │
│   └───────────┘    └───────────┘    └───────────┘    └───────────┘    │
│    os / mgmt /      GB·TB 取 ≥1      单字段失败       POST /api/v1    │
│    hardware七类      最大单位,整数     置 null         /devices       │
│                                                                       │
└────────────────────────────┬─────────────────────────────────────────┘
                             │ JSON over HTTP(12h 一次,失败下周期自愈)
                             ▼
              ┌──────────────────────────────────┐
              │  CMDB API(FastAPI :8080)          │
              │  ingest → diff → 存储 → UI        │
              │  created/unchanged/diff_created   │
              └──────────────────────────────────┘
```

CMDB 侧只管存储和查询:推送是唯一数据写入入口,字段级 diff 进待裁决,UI 只读 + 裁决

## 二,采集器内部数据流

1. **collector(采集)**:各类别**独立采集**--`os` / `mgmt` / `hardware` 七类互不依赖,单类挂掉不殃及别类;单字段失败置 null;`hostname` 失败则整个流程中止(无匹配键不推)
2. **normalize(归一化)**:容量字段归一化--`size` + `size_unit` 统一格式,GB/TB 中数值 ≥ 1 的最大单位,整数值,不出现 mb
3. **payload(组装)**:按 `agent` / `os` / `mgmt` / `hardware` 四组装装,强制执行失败语义与命名规范(同名必同义)
4. **push(推送)**:POST 推送,解析 `created` / `unchanged` / `diff_created` 三分支;推送失败**不重试,不入本地队列**--下个周期全量同步自然自愈

## 三,部署架构

- **Ansible**(控制节点)→ SSH 批量下发二进制 / 配置 / systemd 单元,幂等安装与升级
- **目标机**:systemd timer 本机驱动,采集与 Ansible 解耦--Ansible 不在线也照常推送
- **版本管理**:二进制版本化发布,升级后 `agent.version` 上报随之更新;CMDB 反过来成为 agent 版本台账

## 四,项目目录结构

```
/data/iagent/
├── docs/                          # 文档
│   ├── DEVELOPMENT_PLAN.md        # 开发计划(需求总结 + 分阶段)
│   ├── PAYLOAD_EXAMPLE.md         # 推送体 JSON 基准
│   ├── ARCHITECTURE.md            # 本文档
│   ├── DESIGN.md                  # 代码级详细设计
│   └── USAGE.md                   # 使用手册(构建/部署/验证/排查)
│
├── cmd/
│   ├── iagent/
│   │   └── main.go                # 入口(one-shot):加载配置 → 采集 → 组装 → 推送 → 退出
│   └── prototype/
│       └── main.go                # 阶段 0 字段可得性验证原型
│
├── internal/
│   ├── config/
│   │   └── config.go              # 服务器地址 / token(预留)/ 推送间隔;文件 + 参数两种来源
│   │
│   ├── collector/                 # 采集模块(每个类别独立文件,互不依赖)
│   │   ├── collector.go           # 公共辅助(命令输出解析,null 指针转换)
│   │   ├── os.go                  # hostname / type / version / kernel
│   │   ├── mgmt.go                # IPMI/BMC 带外管理口
│   │   ├── hardware.go            # hardware 组分发器
│   │   ├── nics.go                # 网卡(/sys/class/net,ip)
│   │   ├── memory.go              # 内存(dmidecode)
│   │   ├── cpu.go                 # CPU(/proc/cpuinfo,dmidecode)
│   │   ├── disk.go                # 硬盘(lsblk,/sys/block)
│   │   ├── psu.go                 # 电源(dmidecode)
│   │   └── gpu.go                 # GPU(nvidia-smi --query-gpu)
│   │
│   ├── normalize/
│   │   └── capacity.go            # 容量归一化(GB/TB 规则)+ 单测
│   │
│   ├── payload/
│   │   ├── schema.go              # 推送体结构定义(四组层级)
│   │   └── build.go               # 组装 + 失败语义(null 规则)强制执行
│   │
│   └── push/
│       └── client.go              # HTTP 客户端:POST + result 三分支解析
│
├── systemd/
│   ├── iagent.service             # 服务单元(root 运行)
│   └── iagent.timer               # 12h 定时单元(间隔可配)
│
├── ansible/
│   ├── inventory.example          # 目标机清单
│   ├── playbook.yml               # 部署入口
│   └── roles/
│       └── iagent/
│           ├── tasks/main.yml     # 安装/升级/回滚(幂等)
│           ├── templates/         # 配置文件模板(含 token 占位)
│           └── handlers/main.yml  # 配置变更后重启服务
│
├── tests/
│   ├── fixtures/                  # 集成测试样例数据
│   │   └── collector_example.json
│   ├── local_server.py            # 本地哑服务(:18080 接收推送/提供下载),生产验证用
│   └── session_holder.py          # 堡垒机会话保持辅助(pexpect + FIFO 命令通道)
│
├── .gitignore
└── go.mod
```

## 五,结构设计原则

- **Go 标准布局**:`cmd/` + `internal/`;单测(`*_test.go`)紧跟各模块源码,不另设测试目录;`tests/fixtures/` 只放集成测试样例
- **collector 按类别一文件**:对应开发计划的采集范围表逐类实现,单字段失败置 null 的逻辑收敛在各文件内,`payload` 层做最终防线
- **`systemd/` 和 `ansible/` 与代码同仓**:部署物版本化,升级时二进制版本与 `agent.version` 上报一致
- **配置与代码分离**:`config` 支持文件 + 命令行参数两种来源,token 位预留,CMDB 鉴权就绪后零改动接入
- **文档在 `docs/`**:开发计划,推送体基准,架构文档是后续开发的唯一依据,变更需同步更新
