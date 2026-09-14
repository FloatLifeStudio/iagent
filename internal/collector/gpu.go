package collector

import (
	"strconv"
	"strings"

	"iagent/internal/normalize"
	"iagent/internal/payload"
)

// CollectGpus GPU(nvidia-smi --query-gpu)。nvidia-smi 不存在或失败 → null(机器无 GPU 也 null)。
// 注意:SN 的字段名是 serial(不是 serial_number,驱动会拒绝该查询)。
func CollectGpus() (payload.Gpu, error) {
	out, err := cmdOutput("nvidia-smi",
		"--query-gpu=uuid,gpu_name,serial,memory.total,driver_version,pci.bus_id",
		"--format=csv,noheader,nounits")
	if err != nil {
		return payload.Gpu{}, nil
	}
	var gpu payload.Gpu
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(line, ", ")
		if len(f) < 6 {
			continue
		}
		slot := payload.GpuSlot{
			UUID:          normStr(f[0]),
			Name:          normStr(f[1]),
			SerialNumber:  normStr(f[2]),
			DriverVersion: normStr(f[4]),
			PcieID:        normStr(f[5]),
		}
		fillGpuSize(&slot, strings.TrimSpace(f[3]))
		gpu.Slots = append(gpu.Slots, slot)
	}
	return gpu, nil
}

func fillGpuSize(slot *payload.GpuSlot, mibStr string) {
	// memory.total 以 nounits 输出 MiB 数值
	mib, err := strconv.ParseInt(mibStr, 10, 64)
	if err != nil {
		return
	}
	size, unit := normalize.NormalizeCapacity(mib * 1024 * 1024)
	slot.Size = int64Ptr(size)
	slot.SizeUnit = strPtr(unit)
}
