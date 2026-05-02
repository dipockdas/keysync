package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

var getUnmask bool

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get KEY [--project name]",
		Short: "Retrieve a secret from the local OS secret store",
		Long: `Reads a secret from the local OS secret store and copies it to the clipboard.

By default the value is copied to your clipboard and the key name is displayed.
Use --unmask (-u) to print the value to stdout instead (for scripting or piping).

Resolution order:
  1. Project-scoped secret (if --project is provided)
  2. Global secret (fallback)

Usage:
  keysync get DATABASE_URL
  keysync get STRIPE_KEY --project my-app
  keysync get DATABASE_URL -u
  keysync get DATABASE_URL --unmask`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			ctx := cmd.Context()

			// Try project scope first
			if project != "" {
				val, err := secretSt.Get(ctx, store.ScopeProject, project, key)
				if err == nil {
					return outputSecret(key, val)
				}
				if err != store.ErrNotFound {
					return fmt.Errorf("get secret: %w", err)
				}
				// Fall through to global
			}

			// Try global scope
			val, err := secretSt.Get(ctx, store.ScopeGlobal, "", key)
			if err == store.ErrNotFound {
				fmt.Fprintf(os.Stderr, "secret not found: %s\n", key)
				os.Exit(1)
			}
			if err != nil {
				return fmt.Errorf("get secret: %w", err)
			}
			return outputSecret(key, val)
		},
	}

	cmd.Flags().BoolVarP(&getUnmask, "unmask", "u", false, "print the value to stdout instead of copying to clipboard")
	return cmd
}

// outputSecret either copies the value to clipboard (default) or prints it to stdout (--unmask).
func outputSecret(key, value string) error {
	if getUnmask {
		fmt.Printf("%s=%s", key, value)
		return nil
	}

	if err := copyToClipboard(value); err != nil {
		// Fall back to printing on clipboard failure
		fmt.Printf("%s=%s", key, value)
		return nil
	}
	fmt.Printf("Key %s copied to clipboard\n", key)
	return nil
}

// copyToClipboard copies text to the system clipboard using the OS-native tool.
func copyToClipboard(text string) error {
	clipboards := []struct {
		name string
		args []string
	}{
		{"pbcopy", nil},                          // macOS
		{"xclip", []string{"-selection", "clipboard"}}, // Linux
		{"wl-copy", nil},                         // Linux (Wayland)
		{"xsel", []string{"--clipboard", "--input"}},   // Linux
		{"clip", nil},                            // Windows
	}

	for _, c := range clipboards {
		if _, err := exec.LookPath(c.name); err != nil {
			continue
		}
		cmd := exec.Command(c.name, c.args...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("no clipboard tool found")
}
