package collector

import (
	"encoding/json"

	"iagent/internal/normalize"
	"iagent/internal/payload"
)

// lsblkJSON lsblk -bJ 的输出结构(只取需要的字段)。
type lsblkJSON struct {
	BlockDevices []struct {
		Name   string  `json:"name"`
		Serial *string `json:"serial"`
		Type   string  `json:"type"`
		Model  *string `json:"model"`
		Size   *int64  `json:"size"`
		Rota   *bool   `json:"rota"` // lsblk -J 输出布尔值
	} `json:"blockdevices"`
}

// CollectDisks 硬盘(lsblk -bJ JSON 输出,过滤 loop/ram 等非 disk 设备)。采集失败 → null。
// type 按 ROTA 推断(1=HDD,0=SSD);manufacturer 从 model 解析不可靠,置 null 留待生产验证。
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
