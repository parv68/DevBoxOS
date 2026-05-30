package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/devboxos/devboxos/cli/internal/client"
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
	Use:   "save",
	Short: "Save a snapshot of the current environment",
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
)

func init() {
	snapshotSaveCmd.Flags().StringVarP(&snapshotName, "name", "n", "", "Snapshot name")
	snapshotSaveCmd.Flags().BoolVar(&snapshotIncludeLogs, "include-logs", false, "Include logs in snapshot")
	snapshotLoadCmd.Flags().BoolVarP(&snapshotForce, "force", "f", false, "Force load (overwrite existing)")
	snapshotCmd.AddCommand(snapshotSaveCmd)
	snapshotCmd.AddCommand(snapshotLoadCmd)
	snapshotCmd.AddCommand(snapshotListCmd)
	snapshotCmd.AddCommand(snapshotDeleteCmd)
	snapshotCmd.AddCommand(snapshotExportCmd)
	snapshotCmd.AddCommand(snapshotImportCmd)
	rootCmd.AddCommand(snapshotCmd)
}

func runSnapshotSave(cmd *cobra.Command, args []string) error {
	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
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

	rt := docker.NewDockerRuntime()
	ctx := context.Background()
	if err := rt.Connect(ctx); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer rt.Close()

	mgr := snapshot.NewManager(rt, projectPath)

	outputPath := args[1]
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(projectPath, outputPath)
	}

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

	rt := docker.NewDockerRuntime()
	ctx := context.Background()
	if err := rt.Connect(ctx); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer rt.Close()

	mgr := snapshot.NewManager(rt, projectPath)

	tarballPath := args[0]
	if !filepath.IsAbs(tarballPath) {
		tarballPath = filepath.Join(projectPath, tarballPath)
	}

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
