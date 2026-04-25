package apitypes

// AssociateFloatingIPRequest binds a floating IP to a running instance.
type AssociateFloatingIPRequest struct {
	InstanceID string `json:"instance_id" validate:"required,uuid"`
}
