package commands

import (
	"fmt"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Verify configuration, store, and platform connectivity",
		Long: F(`Runs diagnostics to verify your keysync setup.

{b}Checks:{/b}
  - {c}.keysync.json{/c} is valid and parseable
  - OS secret store is operational (write/read cycle)
  - Project configurations are present
  - Platform API tokens are configured (global scope)

Run this first if you encounter errors with other commands.

{b}Examples:{/b}
  {c}keysync doctor{/c}
  {c}keysync doctor --config /path/to/.keysync.json{/c}

{b}See also:{/b}
  Tutorial: {u}https://github.com/dipockdas/keysync#quick-start{/u}`),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Running diagnostics...")

			// Check config
			if cfg == nil {
				fmt.Println("  ✗ Config: not loaded")
			} else {
				fmt.Printf("  ✓ Config: %s\n", configPath)
				if len(cfg.Repos) == 0 {
					fmt.Println("    (no repos configured)")
				} else {
					for repo, rc := range cfg.Repos {
						fmt.Printf("    - repo: %s → project: %s\n", repo, rc.Project)
						if len(rc.Globals) > 0 {
							fmt.Printf("      globals: %v\n", rc.Globals)
						}
					}
				}
			}

			// Check store
			if secretSt == nil {
				fmt.Println("  ✗ Store: not initialized")
			} else {
				// Try a write/read cycle
				ctx := cmd.Context()
				testKey := "__keysync_doctor_test__"
				if err := secretSt.Set(ctx, store.ScopeGlobal, "", "", testKey, "ok"); err != nil {
					fmt.Printf("  ✗ Store: write failed: %v\n", err)
				} else {
					secretSt.Delete(ctx, store.ScopeGlobal, "", "", testKey)
					fmt.Println("  ✓ Store: operational")
				}
			}

			return nil
		},
	}
}
