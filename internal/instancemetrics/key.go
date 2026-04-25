package instancemetrics

import (
	"fmt"

	"github.com/google/uuid"
)

// StatsRedisKey is the Redis key for JSON-serialized Payload for an instance.
func StatsRedisKey(instanceID uuid.UUID) string {
	return fmt.Sprintf("caspercloud:inst:stats:%s", instanceID.String())
}
