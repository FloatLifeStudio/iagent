package payload

import (
	"errors"
	"strings"
)

// Build assembles the payload, enforcing failure semantics and naming rules.
// Non-empty hostname is the only hard validation; other fields are already
// filled by the collector with single-field failures set to null
func Build(agent Agent, os OS, mgmt *Mgmt, hw *Hardware) (Payload, error) {
	if strings.TrimSpace(os.Hostname) == "" {
		return Payload{}, errors.New("hostname is required")
	}
	if agent.Source == "" {
		agent.Source = "icmdb"
	}
	return Payload{Agent: agent, OS: os, Mgmt: mgmt, Hardware: hw}, nil
}
