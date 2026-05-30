package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var topInterval int

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
	rootCmd.AddCommand(topCmd)
}

func runTop(cmd *cobra.Command, args []string) error {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}

	ctx := context.Background()

	for {
		displayStats(ctx, dockerClient)
		time.Sleep(time.Duration(topInterval) * time.Second)
		clearLines(20)
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
