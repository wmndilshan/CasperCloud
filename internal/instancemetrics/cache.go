package instancemetrics

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryStore holds the latest metrics per instance for the worker process (Prometheus + optional Redis fan-out).
type MemoryStore struct {
	mu   sync.RWMutex
	data map[uuid.UUID]Payload
	prev map[uuid.UUID]cpuSample
}

type cpuSample struct {
	t       time.Time
	cpuTime uint64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[uuid.UUID]Payload),
		prev: make(map[uuid.UUID]cpuSample),
	}
}

// Put stores a payload and derives CPUHostPercent from the previous libvirt cpuTime sample.
func (s *MemoryStore) Put(now time.Time, p Payload, libvirtCPUTimeNs uint64, vcpus uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if vcpus > 0 {
		if prev, ok := s.prev[p.InstanceID]; ok {
			dWall := now.Sub(prev.t)
			dCPU := int64(libvirtCPUTimeNs) - int64(prev.cpuTime)
			if dWall > 0 && dCPU >= 0 {
				wallNs := float64(dWall.Nanoseconds())
				if wallNs > 0 {
					p.CPUHostPercent = 100.0 * float64(dCPU) / wallNs / float64(vcpus)
					if p.CPUHostPercent > 100 {
						p.CPUHostPercent = 100
					}
					if p.CPUHostPercent < 0 {
						p.CPUHostPercent = 0
					}
				}
			}
		}
		s.prev[p.InstanceID] = cpuSample{t: now, cpuTime: libvirtCPUTimeNs}
	}

	s.data[p.InstanceID] = p
}

// Get returns the latest payload for an instance, if any.
func (s *MemoryStore) Get(id uuid.UUID) (Payload, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.data[id]
	return p, ok
}

// Delete removes cached entries (e.g. instance no longer running).
func (s *MemoryStore) Delete(id uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	delete(s.prev, id)
}

// Snapshot returns a shallow copy slice of all cached payloads (for Prometheus).
func (s *MemoryStore) Snapshot() []Payload {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Payload, 0, len(s.data))
	for _, p := range s.data {
		out = append(out, p)
	}
	return out
}
