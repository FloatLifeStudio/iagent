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

// CollectOS os 组:hostname 失败返回 error(整个流程中止),其余字段失败置 null。
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

// osReleasePrettyName 从 /etc/os-release 读 PRETTY_NAME,如 "Ubuntu 22.04.5 LTS"。
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

// detectVirt 虚拟化类型检测:systemd-detect-virt > DMI sys_vendor > bare_metal。
// 不用 gopsutil host.Virtualization:其启发式(/proc/modules 等)在未加载
// guest 内核模块的 VM 上会误报 bare_metal(本机 VMware VM 已踩坑验证)。
func detectVirt() string {
	if out, err := cmdOutput("systemd-detect-virt"); err == nil && out != "" {
		if out == "none" {
			return "bare_metal"
		}
		return out
	}
	// 兜底:DMI sys_vendor(老系统无 systemd-detect-virt 时)
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
