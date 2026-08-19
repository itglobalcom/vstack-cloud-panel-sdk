package entities

import "strconv"

// VMware task states. These are the values returned in VmwareTask.State (wire
// field "is_completed") and are part of the public enum. The unified Task model
// reports them in PascalCase.
const (
	// VmwareTaskStateNew is a queued task that has not started yet.
	VmwareTaskStateNew = "New"
	// VmwareTaskStateInProgress is a task that is currently running.
	VmwareTaskStateInProgress = "InProgress"
	// VmwareTaskStateCompleted is a task that finished successfully (terminal).
	VmwareTaskStateCompleted = "Completed"
	// VmwareTaskStateFailed is a task that finished with an error (terminal).
	VmwareTaskStateFailed = "Failed"
	// VmwareTaskStateCanceled is a task that was canceled (terminal).
	VmwareTaskStateCanceled = "Canceled"
)

// VMware task resource types — the values of VmwareTaskResource.Type. The
// unified Task model reports the resources a task touched via Resources rather
// than dedicated per-resource id fields.
const (
	VmwareTaskResourceServer  = "server"
	VmwareTaskResourceNetwork = "network"
)

// VmwareTaskResource is a reference to a resource a task touched. For VMware the
// ID is an integer id rendered as a decimal string.
type VmwareTaskResource struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// VmwareTask represents an asynchronous VMware operation.
//
// The status arrives in the "is_completed" field as a PascalCase enum; the
// resources the task created or changed are listed in Resources, because the
// unified Task model dropped the dedicated server_id/network_id/error fields.
type VmwareTask struct {
	ID              string               `json:"id"`
	Type            string               `json:"type"`
	State           string               `json:"is_completed"`
	ProgressPercent *int                 `json:"progress_percent,omitempty"`
	Created         string               `json:"created"`
	Completed       *string              `json:"completed,omitempty"`
	Resources       []VmwareTaskResource `json:"resources,omitempty"`
}

// IsCompleted reports whether the task finished successfully.
func (t *VmwareTask) IsCompleted() bool { return t.State == VmwareTaskStateCompleted }

// IsFailed reports whether the task finished with an error.
func (t *VmwareTask) IsFailed() bool { return t.State == VmwareTaskStateFailed }

// IsTerminal reports whether the task reached a terminal state (completed,
// failed or canceled) and will not change further.
func (t *VmwareTask) IsTerminal() bool {
	return t.State == VmwareTaskStateCompleted || t.State == VmwareTaskStateFailed || t.State == VmwareTaskStateCanceled
}

// ResourceID returns the id of the first resource of the given type, or an empty
// string when the task touched no such resource.
func (t *VmwareTask) ResourceID(resourceType string) string {
	for _, r := range t.Resources {
		if r.Type == resourceType {
			return r.ID
		}
	}
	return ""
}

// resourceIDInt returns the numeric id of the first resource of the given type,
// reporting false when it is absent or not a decimal integer.
func (t *VmwareTask) resourceIDInt(resourceType string) (int, bool) {
	id := t.ResourceID(resourceType)
	if id == "" {
		return 0, false
	}
	n, err := strconv.Atoi(id)
	if err != nil {
		return 0, false
	}
	return n, true
}

// NetworkID returns the id of the network the task created or changed, reporting
// false when the task references no network.
func (t *VmwareTask) NetworkID() (int, bool) { return t.resourceIDInt(VmwareTaskResourceNetwork) }

// ServerID returns the id of the server the task created or changed, reporting
// false when the task references no server.
func (t *VmwareTask) ServerID() (int, bool) { return t.resourceIDInt(VmwareTaskResourceServer) }
