// Package collector 各类别独立采集,单字段失败置 null,不中断其他类别。
// 仅 os 组的 hostname 失败返回 error(整个推送流程中止)。
package collector

import (
	"os/exec"
	"strconv"
	"strings"
)

// strPtr 有值返回指针,空串返回 nil(空值视为未采集 → JSON null)。
func strPtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// normStr 采纳 dg-agent 的 _normalize 思路:Unknown/None/N/A 等占位值视为未采集。
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

// parseColonFields 解析 "Key: Value" 风格的命令输出(ipmitool / dmidecode)。
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

// leadingInt 取字符串开头的整数,如 "4800 MT/s" → 4800。
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
