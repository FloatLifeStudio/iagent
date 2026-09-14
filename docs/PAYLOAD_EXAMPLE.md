# iagent 推送体示例

> 推送到 `POST /api/v1/devices` 的 JSON 基准。层级与字段定义见 [DEVELOPMENT_PLAN.md](./DEVELOPMENT_PLAN.md)。
> 命名原则:同名必同义(整机 SN 为 `chassis_serial_number`,条目级 SN 为 `serial_number`)。

## 完整示例

```json
{
  "agent": {
    "version": "0.1.0",
    "source": "icmdb",
    "timestamp": "2026-09-14T07:14:55Z"
  },
  "os": {
    "hostname": "gpu-node-01",
    "type": "linux",
    "version": "Ubuntu 22.04.5 LTS",
    "kernel": "5.15.0-186-generic",
    "virt": "bare_metal"
  },
  "mgmt": {
    "mac": null,
    "ip": null,
    "prefix_length": null
  },
  "hardware": {
    "chassis_serial_number": "SN0000000001",
    "nics": [
      {
        "name": "ens15f0",
        "mac": "AA:BB:CC:DD:EE:01",
        "ips": [
          {"ip": "10.10.1.101", "prefix_length": 24},
          {"ip": "fe80::aabb:11ff:feed:ee01", "prefix_length": 64}
        ]
      },
      {"name": "ens15f1", "mac": "AA:BB:CC:DD:EE:02", "ips": null},
      {"name": "ens15f2", "mac": "AA:BB:CC:DD:EE:03", "ips": null},
      {"name": "ens15f3", "mac": "AA:BB:CC:DD:EE:04", "ips": null},
      {"name": "enx722960f42d0f", "mac": "AA:BB:CC:DD:EE:05", "ips": null},
      {
        "name": "docker0",
        "mac": "AA:BB:CC:DD:EE:06",
        "ips": [{"ip": "172.17.0.1", "prefix_length": 16}]
      }
    ],
    "memory": {
      "slots": [
        {
          "slot": "CPU1_DIMM_A1",
          "manufacturer": "Samsung",
          "part_number": "M321R8GA0BB0-CQKZJ",
          "type": "DDR5",
          "size": 64,
          "size_unit": "GB",
          "speed_mts": 4800,
          "serial_number": "A1B2C3D4"
        },
        {
          "slot": "CPU1_DIMM_B1",
          "manufacturer": "Samsung",
          "part_number": "M321R8GA0BB0-CQKZJ",
          "type": "DDR5",
          "size": 64,
          "size_unit": "GB",
          "speed_mts": 4800,
          "serial_number": "A1B2C3D5"
        }
      ]
    },
    "cpus": [
      {"slot": "CPU1", "model": "Intel(R) Xeon(R) Gold 6454S"},
      {"slot": "CPU2", "model": "Intel(R) Xeon(R) Gold 6454S"}
    ],
    "disks": [
      {
        "serial_number": "DISKSN000001",
        "type": "HDD",
        "manufacturer": null,
        "model": "HGST HUH728080AL",
        "size": 8,
        "size_unit": "TB"
      },
      {
        "serial_number": "DISKSN000002",
        "type": "SSD",
        "manufacturer": null,
        "model": "Samsung SSD 990 EVO 1TB",
        "size": 932,
        "size_unit": "GB"
      }
    ],
    "psus": [
      {
        "serial_number": "2P010000001",
        "manufacturer": "Great Wall",
        "model": "CRPS2700D2",
        "max_power_w": 2700
      },
      {
        "serial_number": "2P010000002",
        "manufacturer": "Great Wall",
        "model": "CRPS2700D2",
        "max_power_w": 2700
      },
      {
        "serial_number": "2P010000003",
        "manufacturer": "Great Wall",
        "model": "CRPS2700D2",
        "max_power_w": 2700
      },
      {
        "serial_number": "2P010000004",
        "manufacturer": "Great Wall",
        "model": "CRPS2700D2",
        "max_power_w": 2700
      }
    ],
    "gpu": {
      "slots": [
        {
          "uuid": "GPU-5a3de5f8-9b2c-4d1e-a7f0-3c8e2b9d1a44",
          "name": "NVIDIA RTX 6000D",
          "serial_number": "GPUSN0000001",
          "size": 84,
          "size_unit": "GB",
          "driver_version": "610.43.02",
          "pcie_id": "00000000:16:00.0"
        },
        {
          "uuid": "GPU-7c1e8a2b-4d3f-9e0a-b5c8-1d2e7f3a9b06",
          "name": "NVIDIA RTX 6000D",
          "serial_number": "GPUSN0000002",
          "size": 84,
          "size_unit": "GB",
          "driver_version": "610.43.02",
          "pcie_id": "00000000:27:00.0"
        }
      ]
    }
  }
}
```

## 关键规则

- **失败语义**:单字段采集失败 → 推 `null`,不省略 key、不省略组;唯一例外 `hostname` 失败 → 整个推送体不推
- **占位值归一化**:`Unknown` / `None` / `N/A` / `NULL` / `Not Specified` 等占位输出视为未采集 → `null`(采纳 dg-agent 的 `_normalize` 思路)
- **空槽跳过**:未插内存(`No Module`)、未插电源(`Status: Not Present`)的槽位不出现,不推全 null 条目
- **容量归一化**:`size` + `size_unit` 统一格式,GB/TB 中数值 ≥ 1 的最大单位,整数值,不出现 mb
- **条目身份**:网卡 `name`、内存/CPU `slot`、硬盘/电源 `serial_number`、GPU `uuid`
- **`full_sync`**:暂不推送;启用后由采集端固定置 `true`

## 字段来源

| 字段 | 来源 |
|---|---|
| `os.hostname` / `type` / `version` / `kernel` | `hostname -f`、`/etc/os-release`、`uname -r` |
| `os.virt` | `systemd-detect-virt`,兜底 DMI sys_vendor;`bare_metal` 或虚拟化类型(kvm/vmware/qemu/xen...) |
| `agent.version` / `source` / `timestamp` | 采集器自身;`timestamp` 统一 UTC(RFC3339) |
| `mgmt.*` | `ipmitool lan print`(带外通道);BMC IP 未分配输出 `0.0.0.0` → `null` |
| `hardware.nics.*` | gopsutil net(排除 `lo`) |
| `hardware.memory/cpus/psus.*` | `dmidecode -t memory / processor / 39` |
| `hardware.disks.*` | `lsblk -bJ -o NAME,SERIAL,TYPE,MODEL,SIZE,ROTA`(JSON,过滤 loop/ram);type 按 ROTA 推断 |
| `hardware.gpu.slots.*` | `nvidia-smi --query-gpu=uuid,gpu_name,serial,memory.total,driver_version,pci.bus_id --format=csv,noheader,nounits` |
