package collector

import (
	"net"

	"iagent/internal/payload"
)

// CollectMgmt collects out-of-band management port info (ipmitool).
// ipmitool missing or failure -> all fields null
func CollectMgmt() (payload.Mgmt, error) {
	m := payload.Mgmt{}
	out, err := cmdOutput("ipmitool", "lan", "print")
	if err != nil {
		return m, nil
	}
	fields := parseColonFields(out)
	m.MAC = strPtr(fields["MAC Address"])
	// ipmitool outputs 0.0.0.0 when the BMC IP is unassigned, count as not collected
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
