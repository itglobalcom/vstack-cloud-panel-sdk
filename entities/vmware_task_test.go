package entities

import (
	"encoding/json"
	"testing"
)

// The unified Task model reports the VMware task status in "is_completed"
// (PascalCase enum) and the touched resources in "resources".
func TestVmwareTaskUnmarshalUnifiedShape(t *testing.T) {
	const payload = `{
		"id": "vmw88",
		"type": "CreateNetwork",
		"is_completed": "Completed",
		"progress_percent": 100,
		"created": "2026-01-01T00:00:00Z",
		"completed": "2026-01-01T00:05:00Z",
		"resources": [
			{"type": "network", "id": "42"},
			{"type": "server", "id": "7"}
		]
	}`

	var task VmwareTask
	if err := json.Unmarshal([]byte(payload), &task); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if task.State != VmwareTaskStateCompleted {
		t.Errorf("State = %q, want %q", task.State, VmwareTaskStateCompleted)
	}
	if !task.IsTerminal() || !task.IsCompleted() {
		t.Fatalf("expected terminal+completed, got state %q", task.State)
	}
	if netID, ok := task.NetworkID(); !ok || netID != 42 {
		t.Errorf("NetworkID() = (%d,%v), want (42,true)", netID, ok)
	}
	if srvID, ok := task.ServerID(); !ok || srvID != 7 {
		t.Errorf("ServerID() = (%d,%v), want (7,true)", srvID, ok)
	}
	if task.ProgressPercent == nil || *task.ProgressPercent != 100 {
		t.Errorf("ProgressPercent mismatch: %v", task.ProgressPercent)
	}
}

func TestVmwareTaskInProgressNotTerminal(t *testing.T) {
	const payload = `{"id":"vmw1","type":"CreateNetwork","is_completed":"InProgress","progress_percent":40,"created":"2026-01-01T00:00:00Z"}`

	var task VmwareTask
	if err := json.Unmarshal([]byte(payload), &task); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if task.State != VmwareTaskStateInProgress {
		t.Errorf("State = %q, want %q", task.State, VmwareTaskStateInProgress)
	}
	if task.IsTerminal() {
		t.Errorf("InProgress must not be terminal (state=%q)", task.State)
	}
	if _, ok := task.NetworkID(); ok {
		t.Errorf("NetworkID() must be false when there are no resources")
	}
}

func TestVmwareTaskResourceIDEdgeCases(t *testing.T) {
	noNet := VmwareTask{Resources: []VmwareTaskResource{{Type: VmwareTaskResourceServer, ID: "5"}}}
	if id, ok := noNet.NetworkID(); ok {
		t.Errorf("NetworkID() must be false when no network resource present, got (%d,true)", id)
	}
	badID := VmwareTask{Resources: []VmwareTaskResource{{Type: VmwareTaskResourceNetwork, ID: "abc"}}}
	if id, ok := badID.NetworkID(); ok {
		t.Errorf("NetworkID() must be false for a non-numeric id, got (%d,true)", id)
	}
	empty := VmwareTask{}
	if _, ok := empty.ServerID(); ok {
		t.Errorf("ServerID() must be false with no resources")
	}
}

func TestVmwareTaskFailedIsTerminalNotCompleted(t *testing.T) {
	const payload = `{"id":"vmw2","type":"CreateNetwork","is_completed":"Failed","created":"2026-01-01T00:00:00Z"}`

	var task VmwareTask
	if err := json.Unmarshal([]byte(payload), &task); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !task.IsTerminal() || !task.IsFailed() || task.IsCompleted() {
		t.Errorf("Failed must be terminal+failed, not completed (state=%q)", task.State)
	}
}
