package commands

import (
	"fmt"
	"os"

	"github.com/dipockdas/keysync/internal/config"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Scaffold .keysync.json configuration",
		Long: `Creates a .keysync.json configuration file in the current directory.
This file maps projects to their deployment platform targets (Vercel, Railway, Supabase).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			cfgPath := config.DefaultConfigPath(dir)
			if _, err := os.Stat(cfgPath); err == nil {
				return fmt.Errorf(".keysync.json already exists at %s", cfgPath)
			}

			cfg := config.DefaultConfig()

			// If a project flag was provided, pre-populate
			if project != "" {
				cfg.Projects[project] = config.ProjectConfig{
					Platforms: config.PlatformConfig{},
				}
			}

			if err := config.SaveConfig(cfg, cfgPath); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			fmt.Printf("Created %s\n", cfgPath)
			return nil
		},
	}
}
