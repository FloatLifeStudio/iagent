# iagent

iCMDB 硬件资产采集客户端--Go 单文件程序(one-shot),部署于被管服务器,systemd timer 定时采集硬件/系统信息推送到 icmdb

## 快速开始

```bash
# 本地构建(带版本号)
CGO_ENABLED=0 go build -ldflags "-X main.version=0.1.0" -o dist/iagent-0.1.0 ./cmd/iagent

# 手动跑一次(先配好 /etc/iagent/config.yml,见使用手册)
sudo ./dist/iagent-0.1.0 --config /etc/iagent/config.yml

# 或 Ansible 批量部署
cd ansible && ansible-playbook -i inventory.yml playbook.yml
```

详细构建配置,部署安装,运行验证,问题排查见 **[使用手册](docs/USAGE.md)**

## 链路

```
采集(iagent,本机 systemd timer 12h)→ JSON → CMDB API → 存储 → UI
```

- 推送是唯一数据写入入口;字段级失败推送 null,CMDB 保留已有数据
- 单字段失败置 null,仅 hostname 失败中止;推送失败下周期自愈
- token 预留,icmdb 鉴权就绪后填入配置

## 文档

- [使用手册](docs/USAGE.md)--构建配置,部署安装,运行验证,问题排查
- [开发计划](docs/DEVELOPMENT_PLAN.md)--需求总结,分阶段计划,风险与里程碑
- [总体架构](docs/ARCHITECTURE.md)--系统全景,数据流,部署架构,目录结构
- [详细设计](docs/DESIGN.md)--代码级设计,模块接口与类型定义
- [推送体基准](docs/PAYLOAD_EXAMPLE.md)--推送 JSON 完整示例与关键规则