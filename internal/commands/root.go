package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dipockdas/keysync/internal/config"
	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags. Default is "dev" for local builds.
var Version = "dev"

var (
	cfgFile    string
	project    string
	envFlag    string
	repoFlag   string
	storeFlag  string
	cfg        *config.Config
	secretSt   store.Store
	configPath string
)

// Execute is the entry point for the CLI.
func Execute() {
	rootCmd := newRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "keysync",
		Version: Version,
		Short:   "Local OS secret management, GitHub Secrets, and deployment platform sync",
		Long: F(`keysync manages secrets in {g}local OS secret stores{/g} (macOS Keychain, Linux libsecret,
Windows Credential Manager), {u}GitHub Secrets{/u} (source of truth), and {c}deployment platforms{/c}
(Vercel, Railway, Supabase). Native client libraries ({c}Go{/c}, {c}Python{/c}, {c}TypeScript{/c}, {c}Swift{/c})
let applications read secrets directly from the OS keychain at runtime.

See {u}https://github.com/dipockdas/keysync{/u} for full documentation and tutorials.`),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip commands that bootstrap the config or don't need it
			if cmd.Name() == "init" || cmd.Name() == "migrate" || cmd.Name() == "help" {
				return nil
			}
			return initializeRuntime()
		},
	}

	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "path to .keysync.json (searches parents by default)")
	cmd.PersistentFlags().StringVarP(&project, "project", "p", "", "project name (from .keysync.json)")
	cmd.PersistentFlags().StringVarP(&envFlag, "env", "e", "", "environment name (optional; omit for project scope, specify for env scope like dev, staging, production)")
	cmd.PersistentFlags().StringVar(&repoFlag, "repo", "", "GitHub repository (owner/repo). Auto-detected if not set.")
	cmd.PersistentFlags().StringVar(&storeFlag, "store", "", `secret store backend ("fallback" to skip OS keychain and use NaCl-encrypted file)`)
	cmd.RegisterFlagCompletionFunc("store", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"fallback"}, cobra.ShellCompDirectiveDefault
	})

	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newSetCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newRmCmd())
	cmd.AddCommand(newPullCmd())
	// cmd.AddCommand(newInjectCmd()) // disabled: see inject.go
	cmd.AddCommand(newPushCmd())
	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newRotateCmd())
	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newTestSecretsCmd())
	cmd.AddCommand(newMigrateCmd())
	cmd.AddCommand(newExportCmd())

	// Version command
	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
		},
	})

	cmd.SetHelpTemplate(helpTemplate())
	cmd.SetUsageTemplate(usageTemplate())

	return cmd
}

// initializeRuntime loads config and creates the OS secret store.
func initializeRuntime() error {
	ctx := context.Background()

	// Load config
	searchDir, _ := os.Getwd()
	if cfgFile != "" {
		configPath = cfgFile
		var err error
		cfg, _, err = config.LoadConfig(filepath.Dir(cfgFile))
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
	} else {
		var err error
		cfg, configPath, err = config.LoadConfig(searchDir)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
	}

	// Allow store backend to be set via environment variable
	if storeFlag == "" {
		if v, ok := os.LookupEnv("KEYSYNC_STORE"); ok {
			storeFlag = v
		}
	}

	// Create OS secret store
	secretSt = openStore(ctx)
	return nil
}

// helpTemplate returns cobra's help template with ANSI-colorized section headings
// when the terminal supports color.
func helpTemplate() string {
	b, r, g, o := "", "", "", ""
	if !noColor {
		b = "\033[1m"
		g = "\033[38;5;40m"
		o = "\033[38;5;202m"
		r = "\033[0m"
	}
	return `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}` + g + `Version: ` + r + `{{.Version}}
` + o + `Source: ` + r + `https://github.com/dipockdas/keysync
{{if or .Runnable .HasSubCommands}}` + b + `Usage:` + r + `{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  ` + o + `{{.CommandPath}}` + r + ` [command]{{end}}{{if gt (len .Aliases) 0}}

` + b + `Aliases:` + r + `
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

` + b + `Examples:` + r + `
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

` + b + `Available Commands:` + r + `{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} ` + g + `{{.Short}}` + r + `{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

` + b + `Flags:` + r + `
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

` + b + `Global Flags:` + r + `
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

` + b + `Additional help topics:` + r + `{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} ` + g + `{{.Short}}` + r + `{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}{{end}}`
}

// usageTemplate returns cobra's usage template with ANSI-colorized section headings
// when the terminal supports color.
func usageTemplate() string {
	b, r, g, o := "", "", "", ""
	if !noColor {
		b = "\033[1m"
		g = "\033[38;5;40m"
		o = "\033[38;5;202m"
		r = "\033[0m"
	}
	return `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  ` + o + `{{.CommandPath}}` + r + ` [command]{{end}}{{if gt (len .Aliases) 0}}

` + b + `Aliases:` + r + `
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

` + b + `Examples:` + r + `
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

` + b + `Available Commands:` + r + `{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} ` + g + `{{.Short}}` + r + `{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

` + b + `Flags:` + r + `
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

` + b + `Global Flags:` + r + `
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

` + b + `Additional help topics:` + r + `{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} ` + g + `{{.Short}}` + r + `{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}`
}

// openStore creates the appropriate Store for the current platform.
func openStore(ctx context.Context) store.Store {
	// Use fallback store when explicitly requested (avoids keychain prompts).
	if storeFlag == "fallback" {
		st, err := store.NewFallbackStore()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to open fallback store: %v\n", err)
			return store.NewMemoryStore()
		}
		return st
	}

	// Attempt OS-native keychain first
	if st, err := tryKeychain(ctx); err == nil {
		return st
	}

	// Fall back to encrypted file store
	st, err := store.NewFallbackStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to open fallback store: %v\n", err)
		return store.NewMemoryStore()
	}
	return st
}
