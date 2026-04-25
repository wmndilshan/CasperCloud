package apitypes

// InstanceActionRequest is the JSON body for POST .../instances/{id}/actions.
type InstanceActionRequest struct {
	Action string `json:"action" validate:"required,min=1,max=32"`
}

// CreateInstanceRequest is the JSON body for POST .../instances.
type CreateInstanceRequest struct {
	Name         string   `json:"name" validate:"required,min=1,max=128"`
	ImageID      string   `json:"image_id" validate:"required,uuid"`
	NetworkID    string   `json:"network_id,omitempty" validate:"omitempty,uuid"`
	Hostname     string   `json:"hostname" validate:"required,min=1,max=253"`
	Username     string   `json:"username" validate:"required,min=1,max=32"`
	SSHPublicKey string   `json:"ssh_public_key" validate:"required,min=1,max=8192"`
	Packages     []string `json:"packages" validate:"max=64,dive,max=128"`
	RunCommands  []string `json:"run_commands" validate:"max=32,dive,max=2048"`
}

// InstanceVolumeAttachRequest is the JSON body for attach/detach volume calls.
type InstanceVolumeAttachRequest struct {
	VolumeID string `json:"volume_id" validate:"required,uuid"`
}

// CreateSnapshotRequest is the JSON body for POST .../instances/{id}/snapshots.
type CreateSnapshotRequest struct {
	Name string `json:"name" validate:"required,min=1,max=128"`
}
