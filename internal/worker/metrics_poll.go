package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"caspercloud/internal/instancemetrics"
	"caspercloud/internal/libvirt"
	"caspercloud/internal/repository"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const defaultMetricsPollInterval = 12 * time.Second

// RunMetricsPoller samples libvirt for all DB-running instances, updates the in-memory store, and optionally writes Redis for the API.
func RunMetricsPoller(ctx context.Context, repo *repository.Repository, lv libvirt.Adapter, mem *instancemetrics.MemoryStore, rdb *redis.Client, interval time.Duration) {
	if interval <= 0 {
		interval = defaultMetricsPollInterval
	}
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			pollOnce(ctx, repo, lv, mem, rdb)
		}
	}
}

func pollOnce(ctx context.Context, repo *repository.Repository, lv libvirt.Adapter, mem *instancemetrics.MemoryStore, rdb *redis.Client) {
	rows, err := repo.ListRunningInstancesForMetrics(ctx)
	if err != nil {
		log.Printf("caspercloud worker: metrics list instances: %v", err)
		return
	}
	alive := make(map[uuid.UUID]struct{}, len(rows))
	now := time.Now().UTC()

	for _, row := range rows {
		alive[row.ID] = struct{}{}
		m, err := lv.GetInstanceStats(ctx, row.Name)
		if err != nil {
			mem.Delete(row.ID)
			if rdb != nil {
				_ = rdb.Del(ctx, instancemetrics.StatsRedisKey(row.ID)).Err()
			}
			continue
		}

		p := instancemetrics.Payload{
			ProjectID:      row.ProjectID,
			InstanceID:     row.ID,
			InstanceName:   row.Name,
			CollectedAt:    now,
			DomainState:    libvirt.DomainStateString(m.DomainState),
			HostInterface:  m.NetIface,
			MaxMemoryKiB:   m.MaxMemoryKiB,
			MemoryKiB:      m.MemoryKiB,
			VCPUCount:      m.VirtCPUCount,
			CPUTimeNs:      m.CPUTimeNs,
			DiskReadBytes:  m.DiskReadBytes,
			DiskWriteBytes: m.DiskWriteBytes,
			DiskReadReq:    m.DiskReadReq,
			DiskWriteReq:   m.DiskWriteReq,
			NetRxBytes:     m.NetRxBytes,
			NetTxBytes:     m.NetTxBytes,
			NetRxPackets:   m.NetRxPackets,
			NetTxPackets:   m.NetTxPackets,
		}
		mem.Put(now, p, m.CPUTimeNs, m.VirtCPUCount)

		if rdb != nil {
			b, jerr := json.Marshal(p)
			if jerr != nil {
				log.Printf("caspercloud worker: metrics json: %v", jerr)
				continue
			}
			if err := rdb.Set(ctx, instancemetrics.StatsRedisKey(row.ID), b, 45*time.Second).Err(); err != nil {
				log.Printf("caspercloud worker: metrics redis set: %v", err)
			}
		}
	}

	for _, prev := range mem.Snapshot() {
		if _, ok := alive[prev.InstanceID]; !ok {
			mem.Delete(prev.InstanceID)
			if rdb != nil {
				_ = rdb.Del(ctx, instancemetrics.StatsRedisKey(prev.InstanceID)).Err()
			}
		}
	}
}
