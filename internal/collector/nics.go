package collector

import (
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v4/net"

	"iagent/internal/payload"
)

// CollectNics collects NIC list (excludes lo). Failure -> nil (JSON null)
func CollectNics() ([]payload.Nic, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil
	}
	var nics []payload.Nic
	for _, ifc := range ifaces {
		if ifc.Name == "lo" {
			continue
		}
		nic := payload.Nic{Name: ifc.Name}
		if ifc.HardwareAddr != "" {
			nic.MAC = strPtr(ifc.HardwareAddr)
		}
		for _, addr := range ifc.Addrs {
			// addr.Addr is CIDR form, e.g. "10.10.1.101/24"
			ip, ipStr, _ := strings.Cut(addr.Addr, "/")
			nicIP := payload.NicIP{IP: ip}
			if ip == addr.Addr {
				nicIP.IP = addr.Addr
			} else if pl, err := strconv.Atoi(ipStr); err == nil {
				nicIP.PrefixLength = intPtr(pl)
			}
			nic.IPs = append(nic.IPs, nicIP)
		}
		nics = append(nics, nic)
	}
	return nics, nil
}
