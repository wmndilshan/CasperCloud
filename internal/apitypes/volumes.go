package apitypes

// CreateVolumeRequest is the JSON body for POST .../volumes.
type CreateVolumeRequest struct {
	Name   string `json:"name" validate:"required,min=1,max=128"`
	SizeGB int    `json:"size_gb" validate:"required,min=1,max=4096"`
}
