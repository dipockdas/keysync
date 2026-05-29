package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newTrustCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "trust",
		Short: "Grant this keysync binary access to all indexed keychain items (macOS)",
		Long: F(`Updates macOS keychain ACLs so the current keysync binary can read secrets
without repeated password prompts. Does not read or print secret values.

Run after {c}make build{/c} or copying a new binary to {c}~/.local/bin{/c}.
For persistent "Always Allow" across rebuilds, use a signed binary: {c}make build-signed{/c}.`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if repairer, ok := secretSt.(interface{ RepairTrust() }); ok {
				repairer.RepairTrust()
				fmt.Fprintln(cmd.OutOrStdout(), "Keychain trust updated for this binary.")
				return nil
			}
			return fmt.Errorf("trust is only supported on macOS keychain")
		},
	}
}
