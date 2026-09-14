package payload

import (
	"errors"
	"strings"
)

// Build 组装推送体,强制执行失败语义与命名规范
// hostname 非空是唯一硬性校验;各字段已由 collector 按单字段失败置 null 填充
func Build(agent Agent, os OS, mgmt *Mgmt, hw *Hardware) (Payload, error) {
	if strings.TrimSpace(os.Hostname) == "" {
		return Payload{}, errors.New("hostname is required")
	}
	if agent.Source == "" {
		agent.Source = "icmdb"
	}
	return Payload{Agent: agent, OS: os, Mgmt: mgmt, Hardware: hw}, nil
}
