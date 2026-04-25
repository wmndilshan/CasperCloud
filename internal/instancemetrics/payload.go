package instancemetrics

import (
	"time"

	"github.com/google/uuid"
)

// Payload is the JSON shape stored in Redis and returned by GET .../stats.
type Payload struct {
	ProjectID    uuid.UUID `json:"project_id"`
	InstanceID   uuid.UUID `json:"instance_id"`
	InstanceName string    `json:"instance_name"`

	CollectedAt   time.Time `json:"collected_at"`
	DomainState   string    `json:"domain_state"`
	HostInterface string    `json:"host_network_interface,omitempty"`

	MaxMemoryKiB uint64 `json:"max_memory_kib"`
	MemoryKiB    uint64 `json:"memory_kib"`
	VCPUCount    uint   `json:"vcpu_count"`
	CPUTimeNs    uint64 `json:"cpu_time_ns"`
	// CPUHostPercent is an approximate guest CPU usage vs wall clock since the previous poll (0–100 per poll window).
	CPUHostPercent float64 `json:"cpu_host_percent,omitempty"`

	DiskReadBytes  int64 `json:"disk_read_bytes"`
	DiskWriteBytes int64 `json:"disk_write_bytes"`
	DiskReadReq    int64 `json:"disk_read_req"`
	DiskWriteReq   int64 `json:"disk_write_req"`

	NetRxBytes   int64 `json:"net_rx_bytes"`
	NetTxBytes   int64 `json:"net_tx_bytes"`
	NetRxPackets int64 `json:"net_rx_packets"`
	NetTxPackets int64 `json:"net_tx_packets"`
}
