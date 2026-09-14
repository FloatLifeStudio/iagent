package collector

import (
	"strings"

	"iagent/internal/payload"
)

// CollectPsus 电源(dmidecode -t 39 System Power Supply)。采集失败 → null。
// 注意:dmidecode type 39 在很多机器上无 Serial Number 字段,SN 身份问题留待生产验证。
func CollectPsus() ([]payload.Psu, error) {
	out, err := cmdOutput("dmidecode", "-t", "39")
	if err != nil {
		return nil, nil
	}
	var psus []payload.Psu
	for _, block := range strings.Split(out, "System Power Supply")[1:] {
		fields := parseColonFields(block)
		// 未插电源的空槽(Status: Not Present)跳过,同内存条 "No Module" 的处理
		if fields["Status"] == "Not Present" {
			continue
		}
		psu := payload.Psu{
			SerialNumber: normStr(fields["Serial Number"]),
			Manufacturer: normStr(fields["Manufacturer"]),
			Model:        normStr(fields["Name"]),
		}
		// dmidecode 版本差异:3.3 用 "Max Power Capacity",新版用 "Maximum Power Capacity"
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
