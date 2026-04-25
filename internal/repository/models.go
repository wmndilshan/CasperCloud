package repository

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Project struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	OwnerID   uuid.UUID `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Image struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"project_id"`
	Name        string    `json:"name"`
	SourceURL   string    `json:"source_url"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Instance struct {
	ID                uuid.UUID  `json:"id"`
	ProjectID         uuid.UUID  `json:"project_id"`
	ImageID           uuid.UUID  `json:"image_id"`
	Name              string     `json:"name"`
	State             string     `json:"state"`
	CloudInitData     string     `json:"cloud_init_data,omitempty"`
	NetworkID         *uuid.UUID `json:"network_id,omitempty"`
	MACAddress        *string    `json:"mac_address,omitempty"`
	BridgeName        string     `json:"bridge_name,omitempty"`
	IPv4Address       *string    `json:"ipv4_address,omitempty"`
	NetworkConfigYAML string     `json:"-"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type Task struct {
	ID         uuid.UUID  `json:"id"`
	Type       string     `json:"type"`
	ProjectID  uuid.UUID  `json:"project_id"`
	InstanceID *uuid.UUID `json:"instance_id,omitempty"`
	Status     string     `json:"status"`
	Error      *string    `json:"error,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// FloatingIP is a public address row; unallocated rows have ProjectID nil.
type FloatingIP struct {
	ID          uuid.UUID  `json:"id"`
	ProjectID   *uuid.UUID `json:"project_id,omitempty"`
	PublicIP    string     `json:"public_ip"`
	InstanceID  *uuid.UUID `json:"instance_id,omitempty"`
	PrivateIP   *string    `json:"private_ip,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
