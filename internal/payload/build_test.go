package payload

import "testing"

func TestBuildOK(t *testing.T) {
	p, err := Build(
		Agent{Version: "0.1.0", Timestamp: "2026-09-13T10:00:00+08:00"},
		OS{Hostname: "S1A01DC-VL101", Type: "linux"},
		nil, nil,
	)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if p.OS.Hostname != "S1A01DC-VL101" {
		t.Errorf("hostname = %q", p.OS.Hostname)
	}
	if p.Agent.Source != "icmdb" {
		t.Errorf("source default = %q, want icmdb", p.Agent.Source)
	}
}

func TestBuildEmptyHostname(t *testing.T) {
	_, err := Build(
		Agent{Version: "0.1.0", Timestamp: "x"},
		OS{Hostname: "", Type: "linux"},
		nil, nil,
	)
	if err == nil {
		t.Fatal("Build with empty hostname should fail")
	}
}
