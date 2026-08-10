package entities

// VMware task states. These are the values returned in VmwareTask.State and are
// part of the public enum.
const (
	// VmwareTaskStateNew is a queued task that has not started yet. Part of the
	// public state enum (SDK-3).
	VmwareTaskStateNew = "new"
	// VmwareTaskStateInProgress is a task that is currently running. Part of the
	// public state enum (SDK-3).
	VmwareTaskStateInProgress = "in_progress"
	// VmwareTaskStateCompleted is a task that finished successfully (terminal).
	VmwareTaskStateCompleted = "completed"
	// VmwareTaskStateFailed is a task that finished with an error (terminal).
	VmwareTaskStateFailed = "failed"
	// VmwareTaskStateCanceled is a task that was canceled (terminal).
	VmwareTaskStateCanceled = "canceled"
)

// VmwareTask represents an asynchronous VMware operation.
type VmwareTask struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`
	State           string  `json:"state"`
	ProgressPercent *int    `json:"progress_percent,omitempty"`
	ServerID        *int    `json:"server_id,omitempty"`
	NetworkID       *int    `json:"network_id,omitempty"`
	Created         string  `json:"created"`
	Completed       *string `json:"completed,omitempty"`
	Error           *string `json:"error,omitempty"`
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
