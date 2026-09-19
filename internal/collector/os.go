package collector

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/shirou/gopsutil/v4/host"

	"iagent/internal/payload"
)

// CollectOS collects the os group: hostname failure returns an error
// (aborts the whole flow), other field failures are set to null
func CollectOS() (payload.OS, error) {
	out, err := exec.Command("hostname", "-f").Output()
	if err != nil {
		return payload.OS{}, fmt.Errorf("collect hostname: %w", err)
	}
	hostname := strings.TrimSpace(string(out))
	if hostname == "" {
		return payload.OS{}, fmt.Errorf("collect hostname: empty")
	}
	os := payload.OS{Hostname: hostname, Type: "linux"}
	if h, err := host.Info(); err == nil {
		os.Kernel = strPtr(h.KernelVersion)
	}
	os.Virt = strPtr(detectVirt())
	if pretty := osReleasePrettyName(); pretty != "" {
		os.Version = strPtr(pretty)
	}
	return os, nil
}

// osReleasePrettyName reads PRETTY_NAME from /etc/os-release, e.g. "Ubuntu 22.04.5 LTS"
func osReleasePrettyName() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if k, v, ok := strings.Cut(sc.Text(), "="); ok && k == "PRETTY_NAME" {
			return strings.Trim(v, `"`)
		}
	}
	return ""
}

// detectVirt detects virtualization: systemd-detect-virt > DMI sys_vendor > bare_metal.
// Does not use gopsutil host.Virtualization: its heuristics (/proc/modules etc.)
// misreport bare_metal on VMs without guest kernel modules loaded (verified on
// the local VMware VM)
func detectVirt() string {
	if out, err := cmdOutput("systemd-detect-virt"); err == nil && out != "" {
		if out == "none" {
			return "bare_metal"
		}
		return out
	}
	// Fallback: DMI sys_vendor (for old systems without systemd-detect-virt)
	if b, err := os.ReadFile("/sys/class/dmi/id/sys_vendor"); err == nil {
		vendor := strings.ToLower(strings.TrimSpace(string(b)))
		for _, kv := range [][2]string{
			{"vmware", "vmware"},
			{"qemu", "kvm"},
			{"kvm", "kvm"},
			{"microsoft", "hyper-v"},
			{"oracle", "vbox"},
			{"innotek", "vbox"},
			{"xen", "xen"},
		} {
			if strings.Contains(vendor, kv[0]) {
				return kv[1]
			}
		}
	}
	return "bare_metal"
}
