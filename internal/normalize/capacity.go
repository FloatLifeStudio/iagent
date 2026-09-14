// Package normalize 容量归一化:统一 size + size_unit,GB/TB 中数值 ≥ 1 的最大单位,
// 结果必须为整数(设备容量均为 2ⁿ,不出现小数),mb 一律不再出现。
package normalize

import (
	"fmt"
	"log"
	"strings"
)

const (
	gb = 1 << 30
	tb = 1 << 40
)

// NormalizeCapacity 输入字节数,输出 size 与 size_unit。
//   1024MB → 1 GB;16384MB → 16 GB;81920MB → 80 GB;8TB → 8 TB
// 输入无法整数化时向上取整并记日志(设备容量均为 2ⁿ,理论上不出现)。
func NormalizeCapacity(bytes int64) (int64, string) {
	if bytes <= 0 {
		log.Printf("[WARN] normalize: non-positive capacity %d, fallback to 1 GB", bytes)
		return 1, "GB"
	}
	gbs := ceilDiv(bytes, gb)
	if gbs >= 1024 {
		return ceilDiv(gbs, 1024), "TB"
	}
	return gbs, "GB"
}

func ceilDiv(a, b int64) int64 {
	if a%b == 0 {
		return a / b
	}
	log.Printf("[WARN] normalize: %d not divisible by %d, rounding up", a, b)
	return a/b + 1
}

// ParseSizeToBytes 解析 dmidecode/lsblk 的容量字符串(如 "64 GB"、"16384 MB")为字节数。
func ParseSizeToBytes(s string) (int64, error) {
	var n int64
	var unit string
	if _, err := fmt.Sscanf(s, "%d %s", &n, &unit); err != nil {
		// 纯数字(无单位)按 MiB 处理(nvidia-smi nounits 风格)
		if _, err := fmt.Sscanf(s, "%d", &n); err == nil {
			return n * 1024 * 1024, nil
		}
		return 0, fmt.Errorf("parse size %q: %w", s, err)
	}
	switch strings.ToUpper(unit) {
	case "KB", "KIB":
		return n * 1024, nil
	case "MB", "MIB":
		return n * 1024 * 1024, nil
	case "GB", "GIB":
		return n * gb, nil
	case "TB", "TIB":
		return n * tb, nil
	default:
		return 0, fmt.Errorf("parse size %q: unknown unit %q", s, unit)
	}
}
