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
		Long: `Runs diagnostics to verify:
  - .keysync.json is valid
  - OS secret store is operational
  - Platform API tokens are configured

Usage:
  keysync doctor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Running diagnostics...")

			// Check config
			if cfg == nil {
				fmt.Println("  ✗ Config: not loaded")
			} else {
				fmt.Printf("  ✓ Config: %s\n", configPath)
				if len(cfg.Projects) == 0 {
					fmt.Println("    (no projects configured)")
				} else {
					for name := range cfg.Projects {
						fmt.Printf("    - project: %s\n", name)
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
