# iagent 推送体示例

> 推送到 `POST /api/v1/devices` 的 JSON 基准。层级与字段定义见 [DEVELOPMENT_PLAN.md](./DEVELOPMENT_PLAN.md)。
> 命名原则:同名必同义(整机 SN 为 `chassis_serial_number`,条目级 SN 为 `serial_number`)。

## 完整示例

```json
{
  "agent": {
    "version": "0.1.0",
    "source": "collector",
    "timestamp": "2026-09-13T10:00:00+08:00"
  },
  "os": {
    "hostname": "S1A01DC-VL101",
    "type": "linux",
    "version": "Ubuntu 22.04.5 LTS",
    "kernel": "5.15.0-131-generic"
  },
  "mgmt": {
    "mac": "AA:BB:CC:DD:EE:01",
    "ip": "192.168.10.101",
    "prefix_length": 24
  },
  "hardware": {
    "chassis_serial_number": "PF4ABC123456",
    "nics": [
      {
        "name": "eth0",
        "mac": "AA:BB:CC:DD:EE:02",
        "ips": [{"ip": "10.10.1.101", "prefix_length": 24}]
      },
      {
        "name": "eth1",
        "mac": "AA:BB:CC:DD:EE:03",
        "ips": [{"ip": "10.10.2.101", "prefix_length": 24}]
      }
    ],
    "memory": {
      "slots": [
        {
          "slot": "DIMM_A1",
          "manufacturer": "Samsung",
          "part_number": "M321R8GA0BB0-CQKZJ",
          "type": "DDR5",
          "size": 64,
          "size_unit": "GB",
          "speed_mts": 4800,
          "serial_number": "123123456"
        },
        {
          "slot": "DIMM_B1",
          "manufacturer": "Kingston",
          "part_number": "KSM56R46BD4PMI-32HAI",
          "type": "DDR5",
          "size": 32,
          "size_unit": "GB",
          "speed_mts": 4800,
          "serial_number": "123123458"
        }
      ]
    },
    "cpus": [
      {"slot": "CPU0", "model": "Intel(R) Xeon(R) Gold 6448Y"},
      {"slot": "CPU1", "model": "Intel(R) Xeon(R) Gold 6448Y"}
    ],
    "disks": [
      {
        "serial_number": "123123123",
        "type": "SSD",
        "manufacturer": "Samsung",
        "model": "990EVO",
        "size": 8,
        "size_unit": "TB"
      },
      {
        "serial_number": "123456",
        "type": "HDD",
        "manufacturer": "HGST",
        "model": "HUH728080ALE604",
        "size": 8,
        "size_unit": "TB"
      }
    ],
    "psus": [
      {
        "serial_number": "2P0123123132",
        "manufacturer": "GreatWall",
        "model": "CRPS2700D2",
        "max_power_w": 2700
      }
    ],
    "gpu": {
      "slots": [
        {
          "uuid": "GPU-5a3de5f8-9b2c-4d1e-a7f0-3c8e2b9d1a44",
          "name": "NVIDIA H100 80GB HBM3",
          "serial_number": "2Q4123123132",
          "size": 80,
          "size_unit": "GB",
          "driver_version": "535.183.01",
          "pcie_id": "0000:1B:00.0"
        },
        {
          "uuid": "GPU-7c1e8a2b-4d3f-9e0a-b5c8-1d2e7f3a9b06",
          "name": "NVIDIA H100 80GB HBM3",
          "serial_number": "2Q4123123133",
          "size": 80,
          "size_unit": "GB",
          "driver_version": "535.183.01",
          "pcie_id": "0000:1C:00.0"
        }
      ]
    }
  }
}
```

## 关键规则

- **失败语义**:单字段采集失败 → 推 `null`,不省略 key、不省略组;唯一例外 `hostname` 失败 → 整个推送体不推
- **容量归一化**:`size` + `size_unit` 统一格式,GB/TB 中数值 ≥ 1 的最大单位,整数值,不出现 mb
- **条目身份**:网卡 `name`、内存/CPU `slot`、硬盘/电源 `serial_number`、GPU `uuid`
- **`full_sync`**:暂不推送;启用后由采集端固定置 `true`
- GPU 示例为 2 槽,GPU 服务器实际可能 8 槽,结构不变

## 字段来源

| 字段 | 来源 |
|---|---|
| `os.hostname` / `type` / `version` / `kernel` | `uname -n`、`/etc/os-release`、`uname -r` |
| `agent.version` / `source` / `timestamp` | 采集器自身 |
| `mgmt.*` | IPMI/BMC(带外通道) |
| `hardware.nics.*` | `/sys/class/net`、`ip` |
| `hardware.memory/cpus/disks/psus.*` | `dmidecode`、`/proc/cpuinfo`、`lsblk` |
| `hardware.gpu.slots.*` | `nvidia-smi --query-gpu=uuid,name,serial_number,memory.total,driver_version,pci.bus_id` |
