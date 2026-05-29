package commands

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts for keysync commands",
		Long: F(`{b}Generate shell completion scripts{/b}

Tab completion for keysync commands, flags, and values. Supports PowerShell, bash, zsh, and fish.

{y}Benefits:{/y}
  • Press Tab to autocomplete commands ({c}keysync s{/c}<Tab> → {c}keysync set{/c})
  • Autocomplete flag names ({c}--pro{/c}<Tab> → {c}--project{/c})
  • Suggest flag values where applicable ({c}--store {/c}<Tab> → {c}fallback{/c})

{y}Quick Setup:{/y}
  {c}PowerShell:{/c}
    keysync completion powershell | Out-String | Invoke-Expression

  {c}bash:{/c}
    source <(keysync completion bash)

  {c}zsh:{/c}
    source <(keysync completion zsh)

  {c}fish:{/c}
    keysync completion fish | source

{g}Documentation:{/g} {u}https://github.com/dipockdas/keysync/blob/main/docs/shell-completion.md{/u}

See each sub-command's help for persistent setup instructions (add to shell profile).`),
	}

	// bash subcommand
	bashCmd := &cobra.Command{
		Use:   "bash",
		Short: "Generate bash completion script",
		Long: F(`{b}Generate bash completion script{/b}

{y}Quick setup (current session):{/y}
  source <(keysync completion bash)

{y}Persistent setup:{/y}
  # Add to ~/.bashrc or ~/.bash_profile:
  source <(keysync completion bash)

  # Or generate to a file:
  keysync completion bash > ~/.config/keysync/completion.bash
  echo 'source ~/.config/keysync/completion.bash' >> ~/.bashrc

{g}Full documentation:{/g} {u}https://github.com/dipockdas/keysync/blob/main/docs/shell-completion.md{/u}`),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Root().GenBashCompletion(os.Stdout)
		},
		DisableFlagsInUseLine: true,
	}

	// zsh subcommand
	zshCmd := &cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completion script",
		Long: F(`{b}Generate zsh completion script{/b}

{y}Quick setup (current session):{/y}
  source <(keysync completion zsh)

{y}Persistent setup:{/y}
  # Add to ~/.zshrc:
  source <(keysync completion zsh)

  # Or generate to a file:
  mkdir -p ~/.config/keysync
  keysync completion zsh > ~/.config/keysync/completion.zsh
  echo 'source ~/.config/keysync/completion.zsh' >> ~/.zshrc

{g}Full documentation:{/g} {u}https://github.com/dipockdas/keysync/blob/main/docs/shell-completion.md{/u}`),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Root().GenZshCompletion(os.Stdout)
		},
		DisableFlagsInUseLine: true,
	}

	// fish subcommand
	fishCmd := &cobra.Command{
		Use:   "fish",
		Short: "Generate fish completion script",
		Long: F(`{b}Generate fish completion script{/b}

{y}Quick setup (current session):{/y}
  keysync completion fish | source

{y}Persistent setup:{/y}
  # Add to ~/.config/fish/config.fish:
  keysync completion fish | source

  # Or generate to fish completions directory:
  keysync completion fish > ~/.config/fish/completions/keysync.fish

{g}Full documentation:{/g} {u}https://github.com/dipockdas/keysync/blob/main/docs/shell-completion.md{/u}`),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Root().GenFishCompletion(os.Stdout, true)
		},
		DisableFlagsInUseLine: true,
	}

	// powershell subcommand
	powershellCmd := &cobra.Command{
		Use:   "powershell",
		Short: "Generate PowerShell completion script",
		Long: F(`{b}Generate PowerShell completion script{/b}

{y}Quick setup (current session):{/y}
  keysync completion powershell | Out-String | Invoke-Expression

{y}Persistent setup:{/y}
  # Add to your PowerShell profile:
  # Find profile location:
  $PROFILE

  # Add this line to the profile file:
  keysync completion powershell | Out-String | Invoke-Expression

  # Or generate to a file and dot-source it:
  keysync completion powershell > $HOME\.config\keysync\completion.ps1
  # Then add to profile:
  . $HOME\.config\keysync\completion.ps1

{g}Full documentation:{/g} {u}https://github.com/dipockdas/keysync/blob/main/docs/shell-completion.md{/u}`),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		},
		DisableFlagsInUseLine: true,
	}

	cmd.AddCommand(bashCmd, zshCmd, fishCmd, powershellCmd)

	return cmd
}
