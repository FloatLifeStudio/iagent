// Package normalize unifies capacity into size + size_unit: the largest unit
// among GB/TB with value >= 1, integer results only (device capacities are all
// powers of two, no decimals), mb never appears.
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

// NormalizeCapacity takes bytes and returns size and size_unit.
//   1024MB -> 1 GB; 16384MB -> 16 GB; 81920MB -> 80 GB; 8TB -> 8 TB
// Rounds up and logs when the input cannot be integer-ized (device capacities
// are all powers of two, should not happen in practice)
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

// ParseSizeToBytes parses dmidecode/lsblk capacity strings (e.g. "64 GB",
// "16384 MB") into bytes
func ParseSizeToBytes(s string) (int64, error) {
	var n int64
	var unit string
	if _, err := fmt.Sscanf(s, "%d %s", &n, &unit); err != nil {
		// bare number (no unit) counts as MiB (nvidia-smi nounits style)
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
