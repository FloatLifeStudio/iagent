package collector

import (
	"strings"

	"iagent/internal/payload"
)

// CollectPsus collects power supplies (dmidecode -t 39 System Power Supply).
// Failure -> null.
// Note: dmidecode type 39 has no Serial Number field on many machines, PSU SN
// identity pending production verification
func CollectPsus() ([]payload.Psu, error) {
	out, err := cmdOutput("dmidecode", "-t", "39")
	if err != nil {
		return nil, nil
	}
	var psus []payload.Psu
	for _, block := range strings.Split(out, "System Power Supply")[1:] {
		fields := parseColonFields(block)
		// skip empty slots (Status: Not Present), same as memory "No Module"
		if fields["Status"] == "Not Present" {
			continue
		}
		psu := payload.Psu{
			SerialNumber: normStr(fields["Serial Number"]),
			Manufacturer: normStr(fields["Manufacturer"]),
			Model:        normStr(fields["Name"]),
		}
		// dmidecode version difference: 3.3 uses "Max Power Capacity", newer uses
		// "Maximum Power Capacity"
		capacity := fields["Maximum Power Capacity"]
		if capacity == "" {
			capacity = fields["Max Power Capacity"]
		}
		if capacity != "" {
			if n, ok := leadingInt(capacity); ok {
				psu.MaxPowerW = intPtr(n)
			}
		}
		psus = append(psus, psu)
	}
	return psus, nil
}
