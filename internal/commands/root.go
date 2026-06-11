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
	cfgFile      string
	project      string
	envFlag      string
	effectiveEnv string // resolved in PersistentPreRunE (see env.go)
	repoFlag     string
	storeFlag    string
	cfg          *config.Config
	secretSt     store.Store
	configPath   string
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
				effectiveEnv = ""
				return nil
			}
			if project == ProjectListSentinel && cmd.Name() != "list" {
				return fmt.Errorf("--project requires a project name")
			}
			if err := initializeRuntime(); err != nil {
				return err
			}
			effectiveEnv = resolveEnvironmentForCommand(cmd)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "path to .keysync.json (searches parents by default)")
	cmd.PersistentFlags().StringVarP(&project, "project", "p", "", "project name (from .keysync.json)")
	cmd.PersistentFlags().Lookup("project").NoOptDefVal = ProjectListSentinel
	cmd.PersistentFlags().StringVarP(&envFlag, "env", "e", "", "environment name (with --project: omit for project-wide; get/export use --env only when passed)")
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
	cmd.AddCommand(newMvCmd())
	cmd.AddCommand(newPullCmd())
	// cmd.AddCommand(newInjectCmd()) // disabled: see inject.go
	cmd.AddCommand(newPushCmd())
	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newRotateCmd())
	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newTestSecretsCmd())
	cmd.AddCommand(newMigrateCmd())
	cmd.AddCommand(newExportCmd())
	cmd.AddCommand(newTrustCmd())
	cmd.AddCommand(newCompletionCmd())

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
	// Use fallback store when explicitly requested via --store=fallback
	if storeFlag == "fallback" {
		st, err := store.NewFallbackStore()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to open fallback store: %v\n", err)
			fmt.Fprintf(os.Stderr, "Fallback store requires write access to: ~/.config/keysync/\n")
			os.Exit(1)
		}

		// Show security warning when using fallback mode
		fmt.Fprintf(os.Stderr, "\n⚠️  WARNING: Using fallback file-based storage instead of OS keychain\n")
		fmt.Fprintf(os.Stderr, "   Storage location: ~/.config/keysync/store.json\n")
		fmt.Fprintf(os.Stderr, "   Encryption key:   ~/.config/keysync/key\n")
		fmt.Fprintf(os.Stderr, "   This provides weaker security than OS-native secret storage.\n")
		fmt.Fprintf(os.Stderr, "   The encryption key and encrypted data are stored on the same disk.\n\n")

		return st
	}

	// Attempt OS-native keychain
	st, err := tryKeychain(ctx)
	if err == nil {
		return st
	}

	// Fail closed when OS keychain unavailable and fallback not requested
	fmt.Fprintf(os.Stderr, "ERROR: OS keychain unavailable: %v\n\n", err)
	fmt.Fprintf(os.Stderr, "Keysync requires OS-native secret storage:\n")
	fmt.Fprintf(os.Stderr, "  • macOS:   Keychain Access\n")
	fmt.Fprintf(os.Stderr, "  • Linux:   libsecret (GNOME Keyring / KDE Wallet)\n")
	fmt.Fprintf(os.Stderr, "  • Windows: Credential Manager\n\n")
	fmt.Fprintf(os.Stderr, "To use file-based storage instead (not recommended), run with:\n")
	fmt.Fprintf(os.Stderr, "  keysync --store=fallback [command]\n\n")
	fmt.Fprintf(os.Stderr, "⚠️  WARNING: Fallback mode stores secrets in an encrypted file with the\n")
	fmt.Fprintf(os.Stderr, "   encryption key stored alongside. This provides weaker protection than\n")
	fmt.Fprintf(os.Stderr, "   OS-native secret storage and should only be used when necessary.\n")
	os.Exit(1)
	return nil // unreachable
}
