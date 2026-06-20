package docker

import (
	"context"
	"encoding/json"

	"github.com/docker/docker/api/types/container"
)

// Stat is a per-container resource sample for the frontend.
type Stat struct {
	ID       string  `json:"id"`
	CPU      float64 `json:"cpu"` // percent of total host CPU
	Mem      int64   `json:"mem"`
	MemLimit int64   `json:"memLimit"`
}

// Sampler computes container CPU% by diffing successive one-shot readings,
// avoiding a persistent stats stream per container. One Sampler backs the single
// shared /api/stats SSE channel.
type Sampler struct {
	c    *Client
	prev map[string]cpuSample
}

type cpuSample struct{ total, system uint64 }

func NewSampler(c *Client) *Sampler {
	return &Sampler{c: c, prev: map[string]cpuSample{}}
}

// Sample reads stats for all running containers once and returns the deltas.
// The first reading for a container reports CPU 0 (two samples are needed).
func (s *Sampler) Sample(ctx context.Context) ([]Stat, error) {
	cli, err := s.c.api(ctx)
	if err != nil {
		return nil, err
	}
	running, err := cli.ContainerList(ctx, container.ListOptions{}) // running only
	if err != nil {
		return nil, err
	}

	out := make([]Stat, 0, len(running))
	seen := make(map[string]bool, len(running))
	for _, ct := range running {
		seen[ct.ID] = true
		resp, err := cli.ContainerStatsOneShot(ctx, ct.ID)
		if err != nil {
			continue
		}
		var sr container.StatsResponse
		decErr := json.NewDecoder(resp.Body).Decode(&sr)
		resp.Body.Close()
		if decErr != nil {
			continue
		}
		out = append(out, Stat{
			ID:       ct.ID,
			CPU:      s.cpuPercent(ct.ID, sr),
			Mem:      usedMemory(sr.MemoryStats),
			MemLimit: int64(sr.MemoryStats.Limit),
		})
	}

	// Drop remembered samples for containers that are no longer running.
	for id := range s.prev {
		if !seen[id] {
			delete(s.prev, id)
		}
	}
	return out, nil
}

func (s *Sampler) cpuPercent(id string, sr container.StatsResponse) float64 {
	cur := cpuSample{total: sr.CPUStats.CPUUsage.TotalUsage, system: sr.CPUStats.SystemUsage}
	prev, ok := s.prev[id]
	s.prev[id] = cur
	if !ok {
		return 0
	}
	cpuDelta := float64(cur.total) - float64(prev.total)
	sysDelta := float64(cur.system) - float64(prev.system)
	if sysDelta <= 0 || cpuDelta < 0 {
		return 0
	}
	cpus := float64(sr.CPUStats.OnlineCPUs)
	if cpus == 0 {
		cpus = float64(len(sr.CPUStats.CPUUsage.PercpuUsage))
	}
	if cpus == 0 {
		cpus = 1
	}
	return (cpuDelta / sysDelta) * cpus * 100.0
}

// usedMemory matches the docker CLI: usage minus reclaimable page cache.
func usedMemory(m container.MemoryStats) int64 {
	used := m.Usage
	if cache, ok := m.Stats["inactive_file"]; ok && cache < used {
		used -= cache
	}
	return int64(used)
}
