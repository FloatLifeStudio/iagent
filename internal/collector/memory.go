package collector

import (
	"log"
	"strings"

	"iagent/internal/normalize"
	"iagent/internal/payload"
)

// CollectMemory collects memory slots (dmidecode). Failure -> null
func CollectMemory() (payload.MemoryInfo, error) {
	out, err := cmdOutput("dmidecode", "-t", "memory")
	if err != nil {
		return payload.MemoryInfo{}, nil
	}
	return payload.MemoryInfo{Slots: parseMemorySlots(out)}, nil
}

func parseMemorySlots(out string) []payload.MemorySlot {
	var slots []payload.MemorySlot
	for _, block := range strings.Split(out, "Memory Device")[1:] {
		fields := parseColonFields(block)
		size := fields["Size"]
		if size == "" || strings.Contains(size, "No Module") {
			continue // skip empty slots
		}
		slot := payload.MemorySlot{
			Slot:         strPtr(fields["Locator"]),
			Manufacturer: normStr(fields["Manufacturer"]),
			PartNumber:   normStr(fields["Part Number"]),
			Type:         normStr(fields["Type"]),
			SerialNumber: normStr(fields["Serial Number"]),
		}
		if bytes, err := normalize.ParseSizeToBytes(size); err == nil {
			s, unit := normalize.NormalizeCapacity(bytes)
			slot.Size = int64Ptr(s)
			slot.SizeUnit = strPtr(unit)
		} else {
			log.Printf("[WARN] memory size parse %q: %v", size, err)
		}
		if speed := fields["Speed"]; speed != "" {
			if n, ok := leadingInt(speed); ok {
				slot.SpeedMTS = intPtr(n)
			}
		}
		slots = append(slots, slot)
	}
	return slots
}
