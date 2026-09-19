// Package collector collects each category independently; a single-field
// failure is set to null without interrupting other categories.
// Only os group hostname failure returns an error (aborts the whole push).
package collector

import (
	"os/exec"
	"strconv"
	"strings"
)

// strPtr returns a pointer for non-empty strings, nil for empty
// (empty counts as not collected -> JSON null)
func strPtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// normStr follows dg-agent's _normalize idea: placeholder values like
// Unknown/None/N/A count as not collected
func normStr(s string) *string {
	switch strings.TrimSpace(s) {
	case "", "Unknown", "unknown", "None", "N/A", "NA", "[N/A]", "No", "none", "NULL", "null", "Not Specified":
		return nil
	}
	return &s
}

func intPtr(n int) *int { return &n }

func int64Ptr(n int64) *int64 { return &n }

func cmdOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// parseColonFields parses "Key: Value" style command output (ipmitool / dmidecode)
func parseColonFields(out string) map[string]string {
	fields := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return fields
}

// leadingInt takes the integer at the start of a string, e.g. "4800 MT/s" -> 4800
func leadingInt(s string) (int, bool) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(s[:i])
	return n, err == nil
}
