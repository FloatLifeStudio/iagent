// Package payload defines the push payload structure and assembly.
// null semantics: all nullable fields use pointer types, nil is JSON null
// (not collected, keep existing data)
package payload

// Payload is the push payload with four top-level groups: agent / os / mgmt / hardware
type Payload struct {
	Agent    Agent     `json:"agent"`
	OS       OS        `json:"os"`
	Mgmt     *Mgmt     `json:"mgmt"`
	Hardware *Hardware `json:"hardware"`
}

// Agent holds envelope metadata
type Agent struct {
	Version   string `json:"version"`
	Source    string `json:"source"`
	Timestamp string `json:"timestamp"`
}

// OS holds hostname (matching key, required) plus OS info
type OS struct {
	Hostname string  `json:"hostname"`
	Type     string  `json:"type"`
	Version  *string `json:"version"`
	Kernel   *string `json:"kernel"`
	Virt     *string `json:"virt"` // bare_metal or virtualization type (kvm/vmware/qemu/xen...)
}

// Mgmt holds out-of-band management port info
type Mgmt struct {
	MAC          *string `json:"mac"`
	IP           *string `json:"ip"`
	PrefixLength *int    `json:"prefix_length"`
}

// Hardware holds all hardware info
type Hardware struct {
	ChassisSerialNumber *string     `json:"chassis_serial_number"`
	Nics                []Nic       `json:"nics"`
	Memory              *MemoryInfo `json:"memory"`
	Cpus                []CpuSlot   `json:"cpus"`
	Disks               []Disk      `json:"disks"`
	Psus                []Psu       `json:"psus"`
	Gpu                 *Gpu        `json:"gpu"`
}

// Nic is a network interface, name is the identity
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

// MemorySlot is a DIMM, slot is the identity
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

// CpuSlot is a CPU, slot is the identity
type CpuSlot struct {
	Slot  *string `json:"slot"`
	Model *string `json:"model"`
}

// Disk is a disk drive, serial_number is the identity
type Disk struct {
	SerialNumber *string `json:"serial_number"`
	Type         *string `json:"type"`
	Manufacturer *string `json:"manufacturer"`
	Model        *string `json:"model"`
	Size         *int64  `json:"size"`
	SizeUnit     *string `json:"size_unit"`
}

// Psu is a power supply, serial_number is the identity
type Psu struct {
	SerialNumber *string `json:"serial_number"`
	Manufacturer *string `json:"manufacturer"`
	Model        *string `json:"model"`
	MaxPowerW    *int    `json:"max_power_w"`
}

type Gpu struct {
	Slots []GpuSlot `json:"slots"`
}

// GpuSlot is a GPU, uuid is the identity
type GpuSlot struct {
	UUID          *string `json:"uuid"`
	Name          *string `json:"name"`
	SerialNumber  *string `json:"serial_number"`
	Size          *int64  `json:"size"`
	SizeUnit      *string `json:"size_unit"`
	DriverVersion *string `json:"driver_version"`
	PcieID        *string `json:"pcie_id"`
}
