// Phase 0 prototype: verify field availability in DEVELOPMENT_PLAN.md 2.2 table
// For verification only, not part of the deliverable
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

type check struct {
	name   string
	source string
	value  string
	err    error
}

func report(c check) {
	status := "OK  "
	if c.err != nil {
		status = "FAIL"
		fmt.Printf("[%s] %-28s (%s): %v\n", status, c.name, c.source, c.err)
		return
	}
	v := c.value
	if len(v) > 80 {
		v = v[:80] + "..."
	}
	fmt.Printf("[%s] %-28s (%s): %s\n", status, c.name, c.source, v)
}

func cmdOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func main() {
	var checks []check

	// --- os ---
	if info, err := host.Info(); err == nil {
		checks = append(checks, check{"os.hostname", "gopsutil host", info.Hostname, nil})
		checks = append(checks, check{"os.kernel", "gopsutil host", info.KernelVersion, nil})
		checks = append(checks, check{"os.type", "gopsutil host", info.OS, nil})
		checks = append(checks, check{"os.platform_version", "gopsutil host", info.PlatformVersion, nil})
	} else {
		checks = append(checks, check{"os.*", "gopsutil host", "", err})
	}

	// --- hardware.chassis_serial_number (/sys/class/dmi/id) ---
	if b, err := os.ReadFile("/sys/class/dmi/id/product_serial"); err == nil {
		checks = append(checks, check{"hardware.chassis_serial (root)", "/sys/class/dmi/id", strings.TrimSpace(string(b)), nil})
	} else {
		checks = append(checks, check{"hardware.chassis_serial (root)", "/sys/class/dmi/id", "", err})
	}
	if b, err := os.ReadFile("/sys/class/dmi/id/product_name"); err == nil {
		checks = append(checks, check{"hardware.product_name", "/sys/class/dmi/id", strings.TrimSpace(string(b)), nil})
	} else {
		checks = append(checks, check{"hardware.product_name", "/sys/class/dmi/id", "", err})
	}

	// --- memory total (gopsutil) ---
	if m, err := mem.VirtualMemory(); err == nil {
		checks = append(checks, check{"memory.total", "gopsutil mem", fmt.Sprintf("%d GB", m.Total/1024/1024/1024), nil})
	} else {
		checks = append(checks, check{"memory.total", "gopsutil mem", "", err})
	}

	// --- memory slots (dmidecode, root) ---
	if out, err := cmdOutput("dmidecode", "-t", "memory"); err == nil {
		n := strings.Count(out, "Memory Device")
		checks = append(checks, check{"memory.slots (root)", "dmidecode -t memory", fmt.Sprintf("%d Memory Device entries", n), nil})
	} else {
		checks = append(checks, check{"memory.slots (root)", "dmidecode -t memory", "", err})
	}

	// --- cpu ---
	if infos, err := cpu.Info(); err == nil && len(infos) > 0 {
		checks = append(checks, check{"cpus[].model", "gopsutil cpu", fmt.Sprintf("%d sockets: %s", len(infos), infos[0].ModelName), nil})
	} else {
		checks = append(checks, check{"cpus[].model", "gopsutil cpu", "", err})
	}

	// --- nics (gopsutil) ---
	if ios, err := net.IOCounters(true); err == nil {
		names := make([]string, 0, len(ios))
		for _, io := range ios {
			if io.Name == "lo" {
				continue
			}
			names = append(names, io.Name)
		}
		checks = append(checks, check{"hardware.nics", "gopsutil net", strings.Join(names, " "), nil})
	} else {
		checks = append(checks, check{"hardware.nics", "gopsutil net", "", err})
	}

	// --- disks (lsblk) ---
	if out, err := cmdOutput("lsblk", "-bndo", "NAME,SERIAL"); err == nil {
		lines := strings.Split(out, "\n")
		checks = append(checks, check{"hardware.disks[].serial", "lsblk", fmt.Sprintf("%d disks: %s", len(lines), strings.Join(lines[:min(3, len(lines))], " | ")), nil})
	} else {
		checks = append(checks, check{"hardware.disks[].serial", "lsblk", "", err})
	}

	// --- psus (dmidecode type 39, root) ---
	if out, err := cmdOutput("dmidecode", "-t", "39"); err != nil {
		checks = append(checks, check{"hardware.psus (root)", "dmidecode -t 39", "", err})
	} else if strings.Contains(out, "No suitable") || len(strings.TrimSpace(out)) < 50 {
		checks = append(checks, check{"hardware.psus (root)", "dmidecode -t 39", "no info (empty table)", nil})
	} else {
		checks = append(checks, check{"hardware.psus (root)", "dmidecode -t 39", strings.Split(out, "\n")[2], nil})
	}

	// --- gpu (nvidia-smi) ---
	if _, err := cmdOutput("nvidia-smi", "--version"); err != nil {
		checks = append(checks, check{"hardware.gpu (root)", "nvidia-smi", "nvidia-smi not found (no GPU on this machine?)", nil})
	} else if out, err := cmdOutput("nvidia-smi", "--query-gpu=uuid,name,serial_number,memory.total,driver_version,pci.bus_id", "--format=csv,noheader,nounits"); err != nil {
		checks = append(checks, check{"hardware.gpu (root)", "nvidia-smi --query-gpu", "", err})
	} else {
		checks = append(checks, check{"hardware.gpu (root)", "nvidia-smi --query-gpu", strings.Split(out, "\n")[0], nil})
	}

	// --- mgmt (ipmitool) ---
	if out, err := cmdOutput("ipmitool", "lan", "print"); err != nil {
		checks = append(checks, check{"mgmt (root)", "ipmitool lan print", "ipmitool missing or unavailable", nil})
	} else {
		checks = append(checks, check{"mgmt (root)", "ipmitool lan print", strings.Split(out, "\n")[0], nil})
	}

	fmt.Println("=== iagent phase 0 field availability check ===")
	fmt.Println()
	for _, c := range checks {
		report(c)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
