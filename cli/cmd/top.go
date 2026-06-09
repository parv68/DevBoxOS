package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	devboxclient "github.com/devboxos/devboxos/cli/internal/client"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cobra"
)

var (
	topInterval int
	topOnce     bool
)

type dockerStats struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage  uint64  `json:"total_usage"`
			PercpuUsage []uint64 `json:"percpu_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs  uint32 `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
	} `json:"memory_stats"`
}

type processStat struct {
	name   string
	cpuPct float64
	mem    uint64
	memPct float64
}

var topCmd = &cobra.Command{
	Use:   "top",
	Short: "Display resource usage dashboard for running services",
	Long: `Show real-time CPU and memory usage for all running services.

Updates every N seconds (default: 2).

Example:
  devbox top
  devbox top --interval 5`,
	RunE: runTop,
}

func init() {
	topCmd.Flags().IntVarP(&topInterval, "interval", "i", 2, "Refresh interval in seconds")
	topCmd.Flags().BoolVarP(&topOnce, "once", "o", false, "Show stats once and exit")
	rootCmd.AddCommand(topCmd)
}

func runTop(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	// Try Docker path first
	dockerAvailable := false
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err == nil {
		_, err := dockerClient.ContainerList(ctx, container.ListOptions{
			Filters: filters.NewArgs(filters.Arg("label", "devboxos.managed")),
		})
		if err == nil {
			dockerAvailable = true
		}
	}

	if dockerAvailable {
		displayStats(ctx, dockerClient)
		if topOnce {
			return nil
		}
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Duration(topInterval) * time.Second):
				displayStats(ctx, dockerClient)
				clearLines(20)
			}
		}
	}

	// Docker unavailable: show host process stats via gopsutil
	cl, err := devboxclient.New()
	if err != nil {
		return fmt.Errorf("Docker not available and engine not running: %w", err)
	}
	defer cl.Close()

	initial, err := cl.Status(".")
	if err == nil && (initial.Status != "running" || len(initial.Services) == 0) {
		fmt.Println("  No running services found")
		return nil
	}

	displayHostStats(cl)
	if topOnce {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Duration(topInterval) * time.Second):
			displayHostStats(cl)
			clearLines(20)
		}
	}
}

func displayHostStats(cl *devboxclient.Client) {
	status, err := cl.Status(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Engine status error: %v\n", err)
		return
	}

	if status.Status != "running" {
		fmt.Println("  No running services (run 'devbox start' first)")
		return
	}

	// Collect service names from engine
	var svcNames []string
	nameSet := make(map[string]bool)
	for _, svc := range status.Services {
		svcNames = append(svcNames, svc.Name)
		nameSet[svc.Name] = true
	}

	// Scan host processes for matching env var DEVBOX_SERVICE_NAME
	procs, _ := process.Processes()
	type procMatch struct {
		name string
		pid  int32
		cpu  float64
		mem  uint64
		memPct float64
	}
	found := make(map[string]*procMatch)

	for _, p := range procs {
		envs, _ := p.Environ()
		svcName := ""
		for _, e := range envs {
			if strings.HasPrefix(e, "DEVBOX_SERVICE_NAME=") {
				svcName = strings.TrimPrefix(e, "DEVBOX_SERVICE_NAME=")
				break
			}
		}
		if svcName == "" || !nameSet[svcName] {
			continue
		}
		cpu, _ := p.CPUPercent()
		memInfo, _ := p.MemoryInfo()
		var mem uint64
		if memInfo != nil {
			mem = memInfo.RSS
		}
		memPct, _ := p.MemoryPercent()
		found[svcName] = &procMatch{
			name:    svcName,
			pid:     p.Pid,
			cpu:     cpu,
			mem:     mem,
			memPct:  float64(memPct),
		}
	}

	// Convert to sorted slice
	var stats []*procMatch
	for _, p := range found {
		stats = append(stats, p)
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].name < stats[j].name })

	sep := strings.Repeat("─", 70)
	fmt.Printf("  %-20s %-8s %-12s %-8s  %s\n", "SERVICE", "CPU%", "MEM", "MEM%", "PID")
	fmt.Printf("  %s\n", sep)

	for _, s := range stats {
		pidStr := "-"
		if s.pid > 0 {
			pidStr = fmt.Sprintf("%d", s.pid)
		}
		fmt.Printf("  %-20s %-8.1f %-12s %-8.1f  %s\n",
			s.name,
			s.cpu,
			formatBytes(int64(s.mem)),
			s.memPct,
			pidStr)
	}

	// Show services without a found process
	for _, name := range svcNames {
		if found[name] == nil {
			fmt.Printf("  %-20s %-8s %-12s %-8s  %s\n", name, "N/A", "N/A", "N/A", "-")
		}
	}
}

func clearLines(n int) {
	for i := 0; i < n; i++ {
		fmt.Print("\033[1A\033[2K")
	}
}

func displayStats(ctx context.Context, dockerClient *client.Client) {
	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", "devboxos.managed=true"),
		),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing containers: %v\n", err)
		return
	}

	sep := strings.Repeat("─", 60)
	fmt.Printf("  %-20s %-8s %-12s %-8s  %s\n", "SERVICE", "CPU%", "MEM", "MEM%", "CONTAINER")
	fmt.Printf("  %s\n", sep)

	if len(containers) == 0 {
		fmt.Println("  No running services found")
		return
	}

	for _, c := range containers {
		svcName := c.Labels["devboxos.service"]
		if svcName == "" {
			svcName = c.Labels["com.docker.compose.service"]
		}
		if svcName == "" {
			svcName = c.ID[:12]
		}

		statsReader, err := dockerClient.ContainerStats(ctx, c.ID, false)
		if err != nil {
			fmt.Printf("  %-20s %-8s %-12s %-8s  %s\n", svcName, "N/A", "N/A", "N/A", c.ID[:12])
			continue
		}

		data, err := io.ReadAll(statsReader.Body)
		statsReader.Body.Close()
		if err != nil {
			continue
		}

		var stats dockerStats
		if err := json.Unmarshal(data, &stats); err != nil {
			continue
		}

		cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
		sysDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
		cpuPct := 0.0
		if sysDelta > 0 && cpuDelta > 0 {
			numCPU := stats.CPUStats.OnlineCPUs
			if numCPU == 0 {
				numCPU = uint32(len(stats.CPUStats.CPUUsage.PercpuUsage))
			}
			if numCPU > 0 {
				cpuPct = (cpuDelta / sysDelta) * float64(numCPU) * 100
			}
		}

		mem := float64(stats.MemoryStats.Usage)
		memLimit := float64(stats.MemoryStats.Limit)
		memPct := 0.0
		if memLimit > 0 {
			memPct = (mem / memLimit) * 100
		}

		fmt.Printf("  %-20s %-8.1f %-12s %-8.1f  %s\n",
			svcName,
			cpuPct,
			formatBytes(int64(mem)),
			memPct,
			c.ID[:12])
	}
}
