# iagent

CMDB 硬件资产采集客户端——Go 单文件常驻程序,部署于被管服务器,定时采集硬件/系统信息推送到 [icmdb](../icmdb)。

## 文档

- [开发计划](docs/DEVELOPMENT_PLAN.md)——需求总结、分阶段计划、风险与里程碑
- [总体架构](docs/ARCHITECTURE.md)——系统全景、数据流、部署架构、目录结构
- [推送体基准](docs/PAYLOAD_EXAMPLE.md)——推送 JSON 完整示例与关键规则

## 链路

```
采集(iagent,本机 systemd timer 12h)→ JSON → CMDB API → 存储 → UI
```

## 状态

🚧 设计定稿,尚未开发。下一步:阶段 0 技术验证(gopsutil 原型 + nvidia-smi 字段确认)。
