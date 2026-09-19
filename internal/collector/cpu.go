package collector

import (
	"strings"

	"iagent/internal/payload"
)

// CollectCpus collects per-socket CPU info (dmidecode -t processor,
// Socket Designation as identity).
// Skips unpopulated slots; failure -> null
func CollectCpus() ([]payload.CpuSlot, error) {
	out, err := cmdOutput("dmidecode", "-t", "processor")
	if err != nil {
		return nil, nil
	}
	var cpus []payload.CpuSlot
	for _, block := range strings.Split(out, "Processor Information")[1:] {
		fields := parseColonFields(block)
		socket := fields["Socket Designation"]
		if socket == "" {
			continue
		}
		if status := strings.ToLower(fields["Status"]); strings.Contains(status, "unpopulated") {
			continue
		}
		cpus = append(cpus, payload.CpuSlot{
			Slot:  strPtr(socket),
			Model: normStr(fields["Version"]),
		})
	}
	return cpus, nil
}
