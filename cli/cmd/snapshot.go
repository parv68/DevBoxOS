package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/devboxos/devboxos/cli/internal/client"
	"github.com/devboxos/devboxos/cli/internal/output"
	"github.com/devboxos/devboxos/shared/config"
	"github.com/devboxos/devboxos/shared/runtime/docker"
	"github.com/devboxos/devboxos/shared/snapshot"
	pb "github.com/devboxos/devboxos/engine/proto"
	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage environment snapshots",
	Long:  `Save, load, export, and import development environment snapshots.`,
}

var snapshotSaveCmd = &cobra.Command{
	Use:   "save [name]",
	Short: "Save a snapshot of the current environment",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSnapshotSave,
}

var snapshotLoadCmd = &cobra.Command{
	Use:   "load <snapshot-id>",
	Short: "Load a snapshot",
	Args:  cobra.ExactArgs(1),
	RunE:  runSnapshotLoad,
}

var snapshotListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all snapshots",
	RunE:  runSnapshotList,
}

var snapshotDeleteCmd = &cobra.Command{
	Use:   "delete <snapshot-id>",
	Short: "Delete a snapshot",
	Args:  cobra.ExactArgs(1),
	RunE:  runSnapshotDelete,
}

var snapshotExportCmd = &cobra.Command{
	Use:   "export <snapshot-id> <output-path>",
	Short: "Export a snapshot to a tarball",
	Args:  cobra.ExactArgs(2),
	RunE:  runSnapshotExport,
}

var snapshotImportCmd = &cobra.Command{
	Use:   "import <tarball-path>",
	Short: "Import a snapshot from a tarball",
	Args:  cobra.ExactArgs(1),
	RunE:  runSnapshotImport,
}

var (
	snapshotName        string
	snapshotForce       bool
	snapshotIncludeLogs bool
	snapshotKeepLast    int
	snapshotOlderThan   string
)

var snapshotGCCmd = &cobra.Command{
	Use:   "gc",
	Short: "Garbage collect old snapshots",
	Long: `Remove old snapshots based on retention policies.

Examples:
  devbox snapshot gc --keep-last 5
  devbox snapshot gc --older-than 30d`,
	RunE: runSnapshotGC,
}

func init() {
	snapshotSaveCmd.Flags().StringVarP(&snapshotName, "name", "n", "", "Snapshot name")
	snapshotSaveCmd.Flags().BoolVar(&snapshotIncludeLogs, "include-logs", false, "Include logs in snapshot")
	snapshotLoadCmd.Flags().BoolVarP(&snapshotForce, "force", "f", false, "Force load (overwrite existing)")
	snapshotGCCmd.Flags().IntVar(&snapshotKeepLast, "keep-last", 0, "Keep the last N snapshots, remove older ones")
	snapshotGCCmd.Flags().StringVar(&snapshotOlderThan, "older-than", "", "Remove snapshots older than duration (e.g., 30d, 7d, 24h)")
	snapshotCmd.AddCommand(snapshotSaveCmd)
	snapshotCmd.AddCommand(snapshotLoadCmd)
	snapshotCmd.AddCommand(snapshotListCmd)
	snapshotCmd.AddCommand(snapshotDeleteCmd)
	snapshotCmd.AddCommand(snapshotExportCmd)
	snapshotCmd.AddCommand(snapshotImportCmd)
	snapshotCmd.AddCommand(snapshotGCCmd)
	rootCmd.AddCommand(snapshotCmd)
}

func runSnapshotGC(cmd *cobra.Command, args []string) error {
	if snapshotKeepLast == 0 && snapshotOlderThan == "" {
		return fmt.Errorf("specify at least one: --keep-last N or --older-than DURATION")
	}

	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	rt := docker.NewDockerRuntime()
	ctx := context.Background()
	if err := rt.Connect(ctx); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer rt.Close()

	mgr := snapshot.NewManager(rt, projectPath)

	infos, err := mgr.List()
	if err != nil {
		return fmt.Errorf("list snapshots: %w", err)
	}

	if len(infos) == 0 {
		fmt.Println("No snapshots to clean")
		return nil
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].CreatedAt.After(infos[j].CreatedAt)
	})

	var toDelete []string

	if snapshotKeepLast > 0 && len(infos) > snapshotKeepLast {
		for _, info := range infos[snapshotKeepLast:] {
			toDelete = append(toDelete, info.ID)
		}
	}

	if snapshotOlderThan != "" {
		duration, err := time.ParseDuration(snapshotOlderThan)
		if err != nil {
			duration = parseDurationDays(snapshotOlderThan)
		}
		cutoff := time.Now().Add(-duration)
		for _, info := range infos {
			if info.CreatedAt.Before(cutoff) {
				alreadyQueued := false
				for _, id := range toDelete {
					if id == info.ID {
						alreadyQueued = true
						break
					}
				}
				if !alreadyQueued {
					toDelete = append(toDelete, info.ID)
				}
			}
		}
	}

	if len(toDelete) == 0 {
		fmt.Println("Nothing to clean")
		return nil
	}

	var totalFreed int64
	for _, id := range toDelete {
		for _, info := range infos {
			if info.ID == id {
				totalFreed += info.SizeBytes
				break
			}
		}
		if err := mgr.Delete(id); err != nil {
			output.Warning("Could not delete snapshot %s: %v", id[:8], err)
		} else {
			fmt.Printf("  Removed: %s\n", id[:8])
		}
	}

	fmt.Printf("\n✓ Removed %d snapshot(s), freed %s\n", len(toDelete), formatBytes(totalFreed))
	return nil
}

func parseDurationDays(s string) time.Duration {
	if len(s) < 2 {
		return 0
	}
	unit := s[len(s)-1:]
	numStr := s[:len(s)-1]
	var num int
	fmt.Sscanf(numStr, "%d", &num)
	switch unit {
	case "d":
		return time.Duration(num) * 24 * time.Hour
	case "w":
		return time.Duration(num) * 7 * 24 * time.Hour
	case "h":
		return time.Duration(num) * time.Hour
	default:
		return 0
	}
}

func runSnapshotSave(cmd *cobra.Command, args []string) error {
	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if snapshotName == "" && len(args) > 0 {
		snapshotName = args[0]
	}
	if snapshotName == "" {
		snapshotName = fmt.Sprintf("%s-%s", filepath.Base(projectPath), "latest")
	}

	if cl, err := client.New(); err == nil {
		defer cl.Close()
		err = cl.SnapshotSave(projectPath, snapshotName, snapshotIncludeLogs, func(msg string) {
			fmt.Printf("ℹ %s\n", msg)
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Snapshot saved: %s\n", snapshotName)
		return nil
	}

	parser := config.NewParser()
	cfg, err := parser.Parse(projectPath)
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	if snapshotName == fmt.Sprintf("%s-%s", filepath.Base(projectPath), "latest") {
		snapshotName = fmt.Sprintf("%s-%s", cfg.Name, "latest")
	}

	rt := docker.NewDockerRuntime()
	ctx := context.Background()
	if err := rt.Connect(ctx); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer rt.Close()

	mgr := snapshot.NewManager(rt, projectPath)

	statusChan := make(chan string, 64)
	go func() {
		for msg := range statusChan {
			fmt.Printf("ℹ %s\n", msg)
		}
	}()

	manifest, err := mgr.Save(ctx, cfg, snapshotName, snapshotIncludeLogs, statusChan)
	if err != nil {
		close(statusChan)
		return err
	}

	close(statusChan)
	fmt.Printf("✓ Snapshot saved: %s (%s)\n", manifest.Name, manifest.ID[:8])
	return nil
}

func runSnapshotLoad(cmd *cobra.Command, args []string) error {
	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if cl, err := client.New(); err == nil {
		defer cl.Close()
		err = cl.SnapshotLoad(projectPath, args[0], snapshotForce, func(msg string) {
			fmt.Printf("ℹ %s\n", msg)
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Snapshot %s loaded\n", args[0][:8])
		return nil
	}

	rt := docker.NewDockerRuntime()
	ctx := context.Background()
	if err := rt.Connect(ctx); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer rt.Close()

	mgr := snapshot.NewManager(rt, projectPath)

	statusChan := make(chan string, 64)
	go func() {
		for msg := range statusChan {
			fmt.Printf("ℹ %s\n", msg)
		}
	}()

	if err := mgr.Load(ctx, args[0], snapshotForce, statusChan); err != nil {
		close(statusChan)
		return err
	}

	close(statusChan)
	fmt.Printf("✓ Snapshot %s loaded\n", args[0][:8])
	return nil
}

func runSnapshotList(cmd *cobra.Command, args []string) error {
	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if cl, err := client.New(); err == nil {
		defer cl.Close()
		snapshots, err := cl.SnapshotList(projectPath)
		if err != nil {
			return err
		}
		if len(snapshots) == 0 {
			fmt.Println("No snapshots found")
			return nil
		}
		printSnapshotTable(snapshots)
		return nil
	}

	rt := docker.NewDockerRuntime()
	ctx := context.Background()
	if err := rt.Connect(ctx); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer rt.Close()

	mgr := snapshot.NewManager(rt, projectPath)

	infos, err := mgr.List()
	if err != nil {
		return err
	}

	if len(infos) == 0 {
		fmt.Println("No snapshots found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSIZE\tCREATED")
	for _, info := range infos {
		sizeStr := formatBytes(info.SizeBytes)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", info.ID[:8], info.Name, sizeStr, info.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	w.Flush()

	return nil
}

func printSnapshotTable(snapshots []*pb.Snapshot) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSIZE\tCREATED")
	for _, s := range snapshots {
		id := s.Id
		if len(id) > 8 {
			id = id[:8]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, s.Name, formatBytes(s.SizeBytes), s.CreatedAt)
	}
	w.Flush()
}

func runSnapshotDelete(cmd *cobra.Command, args []string) error {
	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if cl, err := client.New(); err == nil {
		defer cl.Close()
		if err := cl.SnapshotDelete(projectPath, args[0]); err != nil {
			return err
		}
		fmt.Printf("✓ Snapshot %s deleted\n", args[0][:8])
		return nil
	}

	rt := docker.NewDockerRuntime()
	ctx := context.Background()
	if err := rt.Connect(ctx); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer rt.Close()

	mgr := snapshot.NewManager(rt, projectPath)

	if err := mgr.Delete(args[0]); err != nil {
		return err
	}

	fmt.Printf("✓ Snapshot %s deleted\n", args[0][:8])
	return nil
}

func runSnapshotExport(cmd *cobra.Command, args []string) error {
	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	outputPath := args[1]
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(projectPath, outputPath)
	}

	if cl, err := client.New(); err == nil {
		defer cl.Close()
		err = cl.SnapshotExport(projectPath, outputPath, args[0], func(msg string) {
			fmt.Printf("ℹ %s\n", msg)
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Snapshot exported to %s\n", outputPath)
		return nil
	}

	rt := docker.NewDockerRuntime()
	ctx := context.Background()
	if err := rt.Connect(ctx); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer rt.Close()

	mgr := snapshot.NewManager(rt, projectPath)

	statusChan := make(chan string, 64)
	go func() {
		for msg := range statusChan {
			fmt.Printf("ℹ %s\n", msg)
		}
	}()

	if err := mgr.Export(args[0], outputPath, statusChan); err != nil {
		close(statusChan)
		return err
	}

	close(statusChan)
	fmt.Printf("✓ Snapshot exported to %s\n", outputPath)
	return nil
}

func runSnapshotImport(cmd *cobra.Command, args []string) error {
	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	tarballPath := args[0]
	if !filepath.IsAbs(tarballPath) {
		tarballPath = filepath.Join(projectPath, tarballPath)
	}

	if cl, err := client.New(); err == nil {
		defer cl.Close()
		err = cl.SnapshotImport(projectPath, tarballPath, false, func(msg string) {
			fmt.Printf("ℹ %s\n", msg)
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Snapshot imported from %s\n", tarballPath)
		return nil
	}

	rt := docker.NewDockerRuntime()
	ctx := context.Background()
	if err := rt.Connect(ctx); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer rt.Close()

	mgr := snapshot.NewManager(rt, projectPath)

	statusChan := make(chan string, 64)
	go func() {
		for msg := range statusChan {
			fmt.Printf("ℹ %s\n", msg)
		}
	}()

	if err := mgr.Import(tarballPath, statusChan); err != nil {
		close(statusChan)
		return err
	}

	close(statusChan)
	fmt.Printf("✓ Snapshot imported from %s\n", tarballPath)
	return nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
