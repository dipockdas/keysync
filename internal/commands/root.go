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

var (
	cfgFile    string
	project    string
	repoFlag   string
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
		Use:   "keysync",
		Short: "Unified secret management for GitHub, local stores, and deployment platforms",
		Long: `keysync manages secrets across GitHub Secrets (source of truth),
local OS secret stores (macOS Keychain, Linux libsecret, Windows Credential Manager),
and deployment platforms (Vercel, Railway, Supabase).`,
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
	cmd.PersistentFlags().StringVar(&repoFlag, "repo", "", "GitHub repository (owner/repo). Auto-detected if not set.")

	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newSetCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newPullCmd())
	// cmd.AddCommand(newInjectCmd()) // disabled: see inject.go
	cmd.AddCommand(newSyncCmd())
	cmd.AddCommand(newRotateCmd())
	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newTestSecretsCmd())
	cmd.AddCommand(newMigrateCmd())

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

	// Create OS secret store
	secretSt = openStore(ctx)
	return nil
}

// openStore creates the appropriate Store for the current platform.
func openStore(ctx context.Context) store.Store {
	// Attempt macOS Keychain first on darwin
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
