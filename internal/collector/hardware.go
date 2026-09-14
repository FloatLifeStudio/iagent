package collector

import (
	"os"
	"strings"

	"iagent/internal/payload"
)

// CollectHardware hardware 组:nics/memory/cpus/disks/psus/gpu 各自独立采集,
// 单类别失败置 null,互不影响。
func CollectHardware() (payload.Hardware, error) {
	hw := payload.Hardware{}

	hw.ChassisSerialNumber = chassisSerial()

	if nics, err := CollectNics(); err == nil {
		hw.Nics = nics
	}
	if mem, err := CollectMemory(); err == nil {
		hw.Memory = &mem
	}
	if cpus, err := CollectCpus(); err == nil {
		hw.Cpus = cpus
	}
	if disks, err := CollectDisks(); err == nil {
		hw.Disks = disks
	}
	if psus, err := CollectPsus(); err == nil {
		hw.Psus = psus
	}
	if gpu, err := CollectGpus(); err == nil {
		hw.Gpu = &gpu
	}
	return hw, nil
}

// chassisSerial 整机序列号,root only(/sys/class/dmi/id/product_serial)。
func chassisSerial() *string {
	b, err := os.ReadFile("/sys/class/dmi/id/product_serial")
	if err != nil {
		return nil
	}
	return strPtr(strings.TrimSpace(string(b)))
}
