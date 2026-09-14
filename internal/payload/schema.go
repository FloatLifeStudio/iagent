// Package payload 推送体结构定义与组装
// null 语义:所有可失败字段用指针类型,nil 即 JSON null(未采集,保留现状)
package payload

// Payload 推送体,四组层级:agent / os / mgmt / hardware
type Payload struct {
	Agent    Agent     `json:"agent"`
	OS       OS        `json:"os"`
	Mgmt     *Mgmt     `json:"mgmt"`
	Hardware *Hardware `json:"hardware"`
}

// Agent 信封元数据
type Agent struct {
	Version   string `json:"version"`
	Source    string `json:"source"`
	Timestamp string `json:"timestamp"`
}

// OS hostname(匹配键,必填)+ OS 信息
type OS struct {
	Hostname string  `json:"hostname"`
	Type     string  `json:"type"`
	Version  *string `json:"version"`
	Kernel   *string `json:"kernel"`
	Virt     *string `json:"virt"` // bare_metal 或虚拟化类型(kvm/vmware/qemu/xen...)
}

// Mgmt 带外管理口信息
type Mgmt struct {
	MAC          *string `json:"mac"`
	IP           *string `json:"ip"`
	PrefixLength *int    `json:"prefix_length"`
}

// Hardware 全部硬件信息
type Hardware struct {
	ChassisSerialNumber *string     `json:"chassis_serial_number"`
	Nics                []Nic       `json:"nics"`
	Memory              *MemoryInfo `json:"memory"`
	Cpus                []CpuSlot   `json:"cpus"`
	Disks               []Disk      `json:"disks"`
	Psus                []Psu       `json:"psus"`
	Gpu                 *Gpu        `json:"gpu"`
}

// Nic 网卡,name 为身份
type Nic struct {
	Name string  `json:"name"`
	MAC  *string `json:"mac"`
	IPs  []NicIP `json:"ips"`
}

type NicIP struct {
	IP           string `json:"ip"`
	PrefixLength *int   `json:"prefix_length"`
}

type MemoryInfo struct {
	Slots []MemorySlot `json:"slots"`
}

// MemorySlot 内存条,slot 为身份
type MemorySlot struct {
	Slot         *string `json:"slot"`
	Manufacturer *string `json:"manufacturer"`
	PartNumber   *string `json:"part_number"`
	Type         *string `json:"type"`
	Size         *int64  `json:"size"`
	SizeUnit     *string `json:"size_unit"`
	SpeedMTS     *int    `json:"speed_mts"`
	SerialNumber *string `json:"serial_number"`
}

// CpuSlot CPU,slot 为身份
type CpuSlot struct {
	Slot  *string `json:"slot"`
	Model *string `json:"model"`
}

// Disk 硬盘,serial_number 为身份
type Disk struct {
	SerialNumber *string `json:"serial_number"`
	Type         *string `json:"type"`
	Manufacturer *string `json:"manufacturer"`
	Model        *string `json:"model"`
	Size         *int64  `json:"size"`
	SizeUnit     *string `json:"size_unit"`
}

// Psu 电源,serial_number 为身份
type Psu struct {
	SerialNumber *string `json:"serial_number"`
	Manufacturer *string `json:"manufacturer"`
	Model        *string `json:"model"`
	MaxPowerW    *int    `json:"max_power_w"`
}

type Gpu struct {
	Slots []GpuSlot `json:"slots"`
}

// GpuSlot GPU,uuid 为身份
type GpuSlot struct {
	UUID          *string `json:"uuid"`
	Name          *string `json:"name"`
	SerialNumber  *string `json:"serial_number"`
	Size          *int64  `json:"size"`
	SizeUnit      *string `json:"size_unit"`
	DriverVersion *string `json:"driver_version"`
	PcieID        *string `json:"pcie_id"`
}
