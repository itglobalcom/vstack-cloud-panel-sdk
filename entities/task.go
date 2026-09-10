package entities

// Task states. These are the values returned in TaskResponse.IsCompleted (wire
// field "is_completed"): a five-member PascalCase enum shared by every service,
// not a boolean. Completed, Failed and Canceled are terminal.
const (
	// TaskStateNew is a queued task that has not started yet.
	TaskStateNew = "New"
	// TaskStateInProgress is a task that is currently running.
	TaskStateInProgress = "InProgress"
	// TaskStateCompleted is a task that finished successfully (terminal).
	TaskStateCompleted = "Completed"
	// TaskStateFailed is a task that finished with an error (terminal).
	TaskStateFailed = "Failed"
	// TaskStateCanceled is a task that was canceled (terminal).
	TaskStateCanceled = "Canceled"
)

// Task resource types — the values of TaskResource.Type.
const (
	TaskResourceServer   = "server"
	TaskResourceNetwork  = "network"
	TaskResourceVolume   = "volume"
	TaskResourceSnapshot = "snapshot"
	TaskResourceNIC      = "nic"
	TaskResourceGateway  = "gateway"
	TaskResourceDomain   = "domain"
	TaskResourceRecord   = "record"
	TaskResourcePTR      = "ptr"
	TaskResourceCluster  = "cluster"
)

// TaskResource is a reference to a resource a task touched.
//
// ID is a string in the format the owning service addresses that resource with
// in its own routes: an encoded id for vStack, a decimal integer for VMware and
// K8s, a domain name or a decimal integer for DNS.
type TaskResource struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// IsTaskStateTerminal reports whether a task state will not change any further.
func IsTaskStateTerminal(state string) bool {
	switch state {
	case TaskStateCompleted, TaskStateFailed, TaskStateCanceled:
		return true
	default:
		return false
	}
}

// taskResourceID returns the id of the first resource of the given type, or an
// empty string when the list holds no such resource.
func taskResourceID(resources []TaskResource, resourceType string) string {
	for _, r := range resources {
		if r.Type == resourceType {
			return r.ID
		}
	}
	return ""
}

// TaskResponse represents an asynchronous task.
//
// ID through Resources are the unified task model every service returns.
// Resources lists everything the task touched and is the only complete source:
// the per-resource ID fields below it are deprecated on the API side, are filled
// by some services only, only for a finished task, and never for a VMware task,
// and some resources (a DNS ptr record, a K8s cluster) appear nowhere else.
type TaskResponse struct {
	ID              string         `json:"id"`
	Type            string         `json:"type"`
	IsCompleted     string         `json:"is_completed"`
	ProgressPercent *int           `json:"progress_percent,omitempty"`
	Created         string         `json:"created"`
	Completed       *string        `json:"completed,omitempty"`
	Resources       []TaskResource `json:"resources,omitempty"`

	// LocationID is the location the task runs in — a scope attribute rather
	// than a touched resource, so it has no Resources entry and is not
	// deprecated.
	LocationID string `json:"location_id,omitempty"`

	// Deprecated: read Resources (TaskResourceServer) instead.
	ServerID string `json:"server_id,omitempty"`
	// Deprecated: read Resources (TaskResourceNetwork) instead.
	NetworkID string `json:"network_id,omitempty"`
	// Deprecated: read Resources (TaskResourceVolume) instead.
	VolumeID int `json:"volume_id,omitempty"`
	// Deprecated: read Resources (TaskResourceNIC) instead.
	NicID int `json:"nic_id,omitempty"`
	// Deprecated: read Resources (TaskResourceSnapshot) instead.
	SnapshotID int `json:"snapshot_id,omitempty"`
	// DomainName carries the domain name: DNS addresses a zone by name, and the
	// wire field kept the "domain_id" spelling from before that.
	//
	// Deprecated: read Resources (TaskResourceDomain) instead.
	DomainName string `json:"domain_id,omitempty"`
	// Deprecated: read Resources (TaskResourceRecord) instead.
	RecordID int `json:"record_id,omitempty"`
	// Deprecated: read Resources (TaskResourceGateway) instead.
	GatewayID string `json:"gateway_id,omitempty"`
	// Deprecated: read Resources (TaskResourceCluster) instead.
	KubernetesClusterID string `json:"k8s_cluster_id,omitempty"`
	// KubernetesNodeGroupID is never filled: no task of any service carries a
	// node group id. It is kept only so that published code keeps compiling.
	//
	// Deprecated: the field has no wire counterpart; read Resources instead.
	KubernetesNodeGroupID string `json:"node_group_id,omitempty"`
}

// IsSucceeded reports whether the task finished successfully.
//
// It is not called IsCompleted because that name belongs to the wire field
// carrying the state.
func (t *TaskResponse) IsSucceeded() bool { return t.IsCompleted == TaskStateCompleted }

// IsFailed reports whether the task finished with an error.
func (t *TaskResponse) IsFailed() bool { return t.IsCompleted == TaskStateFailed }

// IsTerminal reports whether the task reached a terminal state (completed,
// failed or canceled) and will not change further.
func (t *TaskResponse) IsTerminal() bool { return IsTaskStateTerminal(t.IsCompleted) }

// ResourceID returns the id of the first resource of the given type, or an empty
// string when the task touched no such resource.
func (t *TaskResponse) ResourceID(resourceType string) string {
	return taskResourceID(t.Resources, resourceType)
}
