package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type Stats struct {
	ServiceName string
	CPU         float64
	Memory      MemoryStats
	Network     NetworkStats
	BlockIO     BlockIOStats
	Timestamp   time.Time
}

type MemoryStats struct {
	Usage   uint64
	Limit   uint64
	Percent float64
}

type NetworkStats struct {
	RxBytes   uint64
	TxBytes   uint64
	RxPackets uint64
	TxPackets uint64
}

type BlockIOStats struct {
	ReadBytes  uint64
	WriteBytes uint64
}

type Monitor struct {
	client     *client.Client
	interval   time.Duration
	mu         sync.RWMutex
	stats      map[string]*Stats
	history    map[string][]*Stats
	maxHistory int
	stopCh     chan struct{}
	running    bool
}

type Option func(*Monitor)

func WithInterval(d time.Duration) Option {
	return func(m *Monitor) {
		m.interval = d
	}
}

func WithMaxHistory(n int) Option {
	return func(m *Monitor) {
		m.maxHistory = n
	}
}

func New(cli *client.Client, opts ...Option) *Monitor {
	m := &Monitor{
		client:     cli,
		interval:   2 * time.Second,
		maxHistory: 100,
		stats:      make(map[string]*Stats),
		history:    make(map[string][]*Stats),
		stopCh:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

func (m *Monitor) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("monitor already running")
	}
	m.running = true
	m.mu.Unlock()

	go m.collect(ctx)
	return nil
}

func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.running = false
	close(m.stopCh)
	m.stopCh = make(chan struct{})
}

func (m *Monitor) GetStats(serviceName string) (*Stats, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats, ok := m.stats[serviceName]
	return stats, ok
}

func (m *Monitor) GetAllStats() map[string]*Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*Stats)
	for k, v := range m.stats {
		result[k] = v
	}
	return result
}

func (m *Monitor) GetHistory(serviceName string) []*Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if history, ok := m.history[serviceName]; ok {
		result := make([]*Stats, len(history))
		copy(result, history)
		return result
	}
	return nil
}

func (m *Monitor) collect(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.collectOnce(ctx)
		}
	}
}

func (m *Monitor) collectOnce(ctx context.Context) {
	containers, err := m.client.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", "devboxos.managed=true"),
		),
	})
	if err != nil {
		return
	}

	for _, c := range containers {
		serviceName := c.Labels["devboxos.service"]
		if serviceName == "" {
			continue
		}

		statsResp, err := m.client.ContainerStatsOneShot(ctx, c.ID)
		if err != nil {
			continue
		}

		var stats container.StatsResponse
		if err := json.NewDecoder(statsResp.Body).Decode(&stats); err != nil {
			statsResp.Body.Close()
			continue
		}
		statsResp.Body.Close()

		s := parseStats(serviceName, &stats)
		if s == nil {
			continue
		}

		m.mu.Lock()
		m.stats[serviceName] = s

		if m.maxHistory > 0 {
			m.history[serviceName] = append(m.history[serviceName], s)
			if len(m.history[serviceName]) > m.maxHistory {
				m.history[serviceName] = m.history[serviceName][1:]
			}
		}
		m.mu.Unlock()
	}
}

func parseStats(serviceName string, stats *container.StatsResponse) *Stats {
	if stats == nil {
		return nil
	}

	s := &Stats{
		ServiceName: serviceName,
		Timestamp:   time.Now(),
	}

	// CPU
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage) - float64(stats.PreCPUStats.SystemUsage)
	if systemDelta > 0 && cpuDelta > 0 {
		numCPUs := float64(stats.CPUStats.OnlineCPUs)
		if numCPUs == 0 {
			numCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
		}
		if numCPUs > 0 {
			s.CPU = (cpuDelta / systemDelta) * numCPUs * 100.0
		}
	}

	// Memory
	s.Memory.Usage = stats.MemoryStats.Usage
	s.Memory.Limit = stats.MemoryStats.Limit
	if s.Memory.Limit > 0 {
		s.Memory.Percent = float64(s.Memory.Usage) / float64(s.Memory.Limit) * 100.0
	}

	// Network
	for _, net := range stats.Networks {
		s.Network.RxBytes += net.RxBytes
		s.Network.TxBytes += net.TxBytes
		s.Network.RxPackets += net.RxPackets
		s.Network.TxPackets += net.TxPackets
	}

	// Block IO
	for _, blk := range stats.BlkioStats.IoServiceBytesRecursive {
		switch blk.Op {
		case "Read":
			s.BlockIO.ReadBytes += blk.Value
		case "Write":
			s.BlockIO.WriteBytes += blk.Value
		}
	}

	return s
}
