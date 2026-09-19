package collector

import (
	"os"
	"strings"

	"iagent/internal/payload"
)

// CollectHardware collects the hardware group: nics/memory/cpus/disks/psus/gpu
// are collected independently, single-category failure is set to null
// without affecting others
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

// chassisSerial reads the chassis serial number, root only
// (/sys/class/dmi/id/product_serial)
func chassisSerial() *string {
	b, err := os.ReadFile("/sys/class/dmi/id/product_serial")
	if err != nil {
		return nil
	}
	return strPtr(strings.TrimSpace(string(b)))
}
