package entities

// TaskResponse represents an asynchronous task
type TaskResponse struct {
	ID          string `json:"id"`
	Created     string `json:"created"`
	IsCompleted string `json:"is_completed"`

	ServerID              string `json:"server_id,omitempty"`
	LocationID            string `json:"location_id,omitempty"`
	NetworkID             string `json:"network_id,omitempty"`
	VolumeID              int    `json:"volume_id,omitempty"`
	NicID                 int    `json:"nic_id,omitempty"`
	SnapshotID            int    `json:"snapshot_id,omitempty"`
	DomainName            string `json:"domain_id,omitempty"`
	RecordID              int    `json:"record_id,omitempty"`
	GatewayID             string `json:"gateway_id,omitempty"`
	KubernetesClusterID   string `json:"cluster_id,omitempty"`
	KubernetesNodeGroupID string `json:"node_group_id,omitempty"`
}
