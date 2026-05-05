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
		Long: F(`Creates a {c}.keysync.json{/c} configuration file in the current directory.

This file maps {c}GitHub repos{/c} to projects, global keys, and deployment platform
targets (Vercel, Railway, Supabase). It contains {b}no secret values{/b} — only
non-sensitive metadata like project IDs. Safe to commit to version control.

After scaffolding, edit {c}.keysync.json{/c} to add repos and platform configs.
Then store your platform API tokens with:

  {c}keysync set VERCEL_TOKEN=...{/c}
  {c}keysync set RAILWAY_TOKEN=...{/c}
  {c}keysync set SUPABASE_TOKEN=...{/c}

{b}Examples:{/b}
  {c}keysync init{/c}                                                       # empty config
  {c}keysync init --project my-app --repo org/my-app{/c}                    # with repo entry

{b}See also:{/b}
  Configuration docs: {u}https://github.com/dipockdas/keysync#configuration{/u}
  Tutorial: {u}https://github.com/dipockdas/keysync#tutorials{/u}`),
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

			// If project and repo flags provided, pre-populate
			if project != "" && repoFlag != "" {
				cfg.Repos[repoFlag] = config.RepoConfig{
					Project:   project,
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
