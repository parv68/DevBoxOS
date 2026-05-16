package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for DevBoxOS CLI.

To load completions:

Bash:
  $ source <(devbox completion bash)
  $ devbox completion bash > /etc/bash_completion.d/devbox

Zsh:
  $ devbox completion zsh > "${fpath[1]}/_devbox"

Fish:
  $ devbox completion fish | source
  $ devbox completion fish > ~/.config/fish/completions/devbox.fish

PowerShell:
  PS> devbox completion powershell | Out-String | Invoke-Expression
  PS> devbox completion powershell > devbox-completions.ps1
`,
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Args:      cobra.ExactValidArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletion(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
