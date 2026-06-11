package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newTrustCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "trust",
		Short: "Grant this keysync binary access to all indexed keychain items (macOS)",
		Long: F(`Grants the current keysync binary access to all indexed keychain items so
{c}get{/c}, {c}export{/c}, and {c}mv{/c} stop prompting. Updates partition lists (team ID
on signed builds) and trusted-application ACLs (required for unsigned {c}make build{/c}
binaries). Does not read or print secret values.

You will be prompted once for your Mac login keychain password (input is hidden and not
stored), then all indexed items are updated. {c}keysync mv{/c} and {c}keysync set{/c} do
not ask for that password.

Run after {c}make build{/c} or copying a new binary to {c}~/.local/bin{/c}.
For persistent "Always Allow" across rebuilds, use a signed binary: {c}make build-signed{/c}.`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if repairer, ok := secretSt.(interface {
				RepairTrust() (succeeded, failed int, err error)
			}); ok {
				ok, failed, err := repairer.RepairTrust()
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Keychain trust updated for this binary (%d items", ok)
				if failed > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), ", %d failed — item missing from keychain or stale index entry", failed)
				}
				fmt.Fprintln(cmd.OutOrStdout(), ").")
				return nil
			}
			return fmt.Errorf("trust is only supported on macOS keychain")
		},
	}
}
