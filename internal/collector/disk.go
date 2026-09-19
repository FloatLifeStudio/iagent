package collector

import (
	"encoding/json"

	"iagent/internal/normalize"
	"iagent/internal/payload"
)

// lsblkJSON is the output structure of lsblk -bJ (only fields we need)
type lsblkJSON struct {
	BlockDevices []struct {
		Name   string  `json:"name"`
		Serial *string `json:"serial"`
		Type   string  `json:"type"`
		Model  *string `json:"model"`
		Size   *int64  `json:"size"`
		Rota   *bool   `json:"rota"` // lsblk -J outputs booleans
	} `json:"blockdevices"`
}

// CollectDisks collects disks (lsblk -bJ JSON output, filters non-disk devices
// such as loop/ram). Failure -> null.
// type is inferred from ROTA (1=HDD, 0=SSD); manufacturer parsing from model is
// unreliable, set to null pending production verification
func CollectDisks() ([]payload.Disk, error) {
	out, err := cmdOutput("lsblk", "-bJ", "-o", "NAME,SERIAL,TYPE,MODEL,SIZE,ROTA")
	if err != nil {
		return nil, nil
	}
	var parsed lsblkJSON
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return nil, nil
	}
	var disks []payload.Disk
	for _, dev := range parsed.BlockDevices {
		if dev.Type != "disk" {
			continue
		}
		d := payload.Disk{
			SerialNumber: dev.Serial,
			Model:        dev.Model,
		}
		if dev.Rota != nil {
			d.Type = strPtr(diskType(*dev.Rota))
		}
		if dev.Size != nil {
			size, unit := normalize.NormalizeCapacity(*dev.Size)
			d.Size = int64Ptr(size)
			d.SizeUnit = strPtr(unit)
		}
		disks = append(disks, d)
	}
	return disks, nil
}

func diskType(rota bool) string {
	if rota {
		return "HDD"
	}
	return "SSD"
}
