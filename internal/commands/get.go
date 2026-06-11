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
		Use:   "get KEY [--project name] [--env name]",
		Short: "Retrieve a secret from the local OS secret store",
		Long: F(`Reads a secret from the local OS secret store and copies it to the clipboard.

By default the value is copied to your clipboard and the key name is displayed.
Use {c}--unmask{/c} ({c}-u{/c}) to print the value to stdout instead (for scripting or piping).

{b}Resolution order{/b} (when {c}--project{/c} is provided):
  1. Project + environment-scoped secret (only if {c}--env{/c} is passed)
  2. Project-scoped secret (no environment)
  3. Global secret

Without {c}--project{/c}, only the global scope is checked.

Only one keychain read is performed (location is resolved from the local index first).

{b}Examples:{/b}
  {c}keysync get DATABASE_URL{/c}                                    {g}# global only{/g}
  {c}keysync get STRIPE_KEY --project my-app{/c}                     {g}# project → global{/g}
  {c}keysync get DATABASE_URL --project my-app --env staging{/c}     {g}# env → project → global{/g}
  {c}keysync get DATABASE_URL -u{/c}                                 {g}# print to stdout{/g}

{b}See also:{/b}
  {c}keysync export KEY{/c} — same resolution order, prints {c}export KEY=VALUE{/c} for shell eval
  Tutorial: {u}https://github.com/dipockdas/keysync#quick-start{/u}`),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("requires a KEY to retrieve (e.g. keysync get DATABASE_URL)")
			}
			if len(args) > 2 {
				return fmt.Errorf("accepts only one KEY at a time")
			}
			if len(args) == 2 && !mightBeTrailingProjectArg(args) {
				return fmt.Errorf("accepts only one KEY at a time")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			args = commandArgs(args)
			key := args[0]
			if err := validateKeyName(key); err != nil {
				return err
			}
			ctx := cmd.Context()

			explicitEnv := envForGet(cmd)
			scope, proj, env, found := locateSecret(ctx, key, project, explicitEnv)
			if !found {
				fmt.Fprintf(os.Stderr, "secret not found: %s\n", key)
				os.Exit(1)
			}

			val, err := secretSt.Get(ctx, scope, proj, env, key)
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
		return fmt.Errorf("clipboard unavailable: %w\n\nTo print to stdout instead, run with: keysync get %s --unmask", err, key)
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
		{"pbcopy", nil},
		{"xclip", []string{"-selection", "clipboard"}},
		{"wl-copy", nil},
		{"xsel", []string{"--clipboard", "--input"}},
		{"clip", nil},
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
