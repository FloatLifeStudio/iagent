package collector

import (
	"net"

	"iagent/internal/payload"
)

// CollectMgmt 带外管理口信息(ipmitool)。ipmitool 不存在或失败 → 字段全 null。
func CollectMgmt() (payload.Mgmt, error) {
	m := payload.Mgmt{}
	out, err := cmdOutput("ipmitool", "lan", "print")
	if err != nil {
		return m, nil
	}
	fields := parseColonFields(out)
	m.MAC = strPtr(fields["MAC Address"])
	// BMC IP 未分配时 ipmitool 输出 0.0.0.0,视为未采集
	if ipStr := fields["IP Address"]; ipStr != "" && ipStr != "0.0.0.0" {
		m.IP = strPtr(ipStr)
	}
	if mask := fields["Subnet Mask"]; mask != "" && mask != "0.0.0.0" && m.IP != nil {
		if ip := net.ParseIP(mask); ip != nil {
			ones, _ := net.IPMask(ip.To4()).Size()
			m.PrefixLength = intPtr(ones)
		}
	}
	return m, nil
}
