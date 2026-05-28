package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/devboxos/devboxos/cli/internal/client"
	"github.com/devboxos/devboxos/shared/secrets"
	"github.com/spf13/cobra"
)

var (
	secretsKeyPath   string
	secretsStorePath string
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage encrypted secrets",
	Long:  `Manage encrypted secrets for your DevBoxOS environment.`,
}

var secretsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stored secrets",
	RunE:  runSecretsList,
}

var secretsGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Retrieve a secret value",
	Args:  cobra.ExactArgs(1),
	RunE:  runSecretsGet,
}

var secretsSetCmd = &cobra.Command{
	Use:   "set <name> <value>",
	Short: "Store a secret",
	Args:  cobra.ExactArgs(2),
	RunE:  runSecretsSet,
}

var secretsRotateCmd = &cobra.Command{
	Use:   "rotate <name>",
	Short: "Rotate a secret",
	Args:  cobra.ExactArgs(1),
	RunE:  runSecretsRotate,
}

var secretsRmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove a secret",
	Args:  cobra.ExactArgs(1),
	RunE:  runSecretsRm,
}

func init() {
	secretsCmd.PersistentFlags().StringVar(&secretsKeyPath, "key-path", "", "Path to encryption key")
	secretsCmd.PersistentFlags().StringVar(&secretsStorePath, "store-path", "", "Path to encrypted store")
	secretsCmd.AddCommand(secretsListCmd)
	secretsCmd.AddCommand(secretsGetCmd)
	secretsCmd.AddCommand(secretsSetCmd)
	secretsCmd.AddCommand(secretsRotateCmd)
	secretsCmd.AddCommand(secretsRmCmd)
	rootCmd.AddCommand(secretsCmd)
}

func getProjectPath() (string, error) {
	return os.Getwd()
}

func getResolver() (*secrets.Resolver, error) {
	projectPath, err := getProjectPath()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	if secretsKeyPath == "" {
		secretsKeyPath = filepath.Join(projectPath, ".devbox", "secrets.key")
	}
	if secretsStorePath == "" {
		secretsStorePath = filepath.Join(projectPath, ".devbox", "secrets.enc")
	}

	return secrets.NewResolver(projectPath, secretsKeyPath, secretsStorePath)
}

func tryEngineClient() (*client.Client, error) {
	return client.New()
}

func printSecretEntries(entries []secrets.SecretEntry) {
	if len(entries) == 0 {
		fmt.Println("No secrets stored")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSOURCE\tCREATED")
	for _, entry := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\n", entry.Name, entry.Source, entry.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	w.Flush()
}

func runSecretsList(cmd *cobra.Command, args []string) error {
	projectPath, _ := getProjectPath()

	if conn, err := tryEngineClient(); err == nil {
		defer conn.Close()
		pbEntries, err := conn.SecretList(projectPath)
		if err == nil {
			for _, e := range pbEntries {
				fmt.Printf("%s\t%s\t%s\n", e.Name, e.Source, e.CreatedAt)
			}
			return nil
		}
	}

	resolver, err := getResolver()
	if err != nil {
		return err
	}

	entries := resolver.List()
	printSecretEntries(entries)
	return nil
}

func runSecretsGet(cmd *cobra.Command, args []string) error {
	projectPath, _ := getProjectPath()

	if conn, err := tryEngineClient(); err == nil {
		defer conn.Close()
		value, err := conn.SecretGet(projectPath, args[0])
		if err == nil {
			fmt.Println(value)
			return nil
		}
	}

	resolver, err := getResolver()
	if err != nil {
		return err
	}

	value, err := resolver.Get(args[0])
	if err != nil {
		return err
	}

	fmt.Println(value)
	return nil
}

func runSecretsSet(cmd *cobra.Command, args []string) error {
	projectPath, _ := getProjectPath()

	if conn, err := tryEngineClient(); err == nil {
		defer conn.Close()
		if err := conn.SecretSet(projectPath, args[0], args[1]); err == nil {
			fmt.Printf("✓ Secret %s stored (encrypted)\n", args[0])
			return nil
		}
	}

	resolver, err := getResolver()
	if err != nil {
		return err
	}

	if err := resolver.Set(args[0], args[1]); err != nil {
		return err
	}

	fmt.Printf("✓ Secret %s stored (encrypted)\n", args[0])
	return nil
}

func runSecretsRotate(cmd *cobra.Command, args []string) error {
	projectPath, _ := getProjectPath()

	if conn, err := tryEngineClient(); err == nil {
		defer conn.Close()
		if err := conn.SecretRotate(projectPath, args[0]); err == nil {
			fmt.Printf("✓ Secret %s rotated\n", args[0])
			return nil
		}
	}

	resolver, err := getResolver()
	if err != nil {
		return err
	}

	if err := resolver.Rotate(args[0]); err != nil {
		return err
	}

	fmt.Printf("✓ Secret %s rotated\n", args[0])
	return nil
}

func runSecretsRm(cmd *cobra.Command, args []string) error {
	projectPath, _ := getProjectPath()

	if conn, err := tryEngineClient(); err == nil {
		defer conn.Close()
		if err := conn.SecretDelete(projectPath, args[0]); err == nil {
			fmt.Printf("✓ Secret %s removed\n", args[0])
			return nil
		}
	}

	resolver, err := getResolver()
	if err != nil {
		return err
	}

	if err := resolver.Delete(args[0]); err != nil {
		return err
	}

	fmt.Printf("✓ Secret %s removed\n", args[0])
	return nil
}
