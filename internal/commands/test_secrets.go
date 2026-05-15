package commands

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

func newTestSecretsCmd() *cobra.Command {
	var testCount int

	cmd := &cobra.Command{
		Use:   "test-secrets --project name [--count N] [--env name]",
		Short: "Generate ephemeral test secrets for CI/local test runs",
		Long: F(`Generates temporary, prefixed secrets for use in CI test runs or local
development testing. These are stored in the local OS secret store with a
{b}TEST_{/b} prefix and hex-encoded random values.

{c}--project{/c} is required. Use {c}--count{/c} to control how many test secrets to create
(default: 3). Secrets are scoped to the project (and {c}--env{/c} if provided).

{b}Examples:{/b}
  {c}keysync test-secrets --project my-app{/c}
  {c}keysync test-secrets --project my-app --count 5{/c}
  {c}keysync test-secrets --project my-app --env staging{/c}

{b}See also:{/b}
  {c}keysync list --help{/c}
  {c}keysync get --help{/c}`),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			ctx := cobraCmd.Context()

			scope := store.ScopeProject
			if project == "" {
				return fmt.Errorf("--project is required for test-secrets")
			}

			// Generate test secrets
			names := make([]string, 0, testCount)
			for i := 0; i < testCount; i++ {
				key := fmt.Sprintf("TEST_SECRET_%d", i+1)
				value, err := generateTestValue()
				if err != nil {
					return fmt.Errorf("generate value: %w", err)
				}

				if err := secretSt.Set(ctx, scope, project, envFlag, key, value); err != nil {
					return fmt.Errorf("set %s: %w", key, err)
				}
				names = append(names, key)
			}

			sort.Strings(names)
			fmt.Printf("Created %d test secrets for project %q (env: %s):\n", testCount, project, envFlag)
			for _, name := range names {
				fmt.Printf("  %s\n", name)
			}

			if len(names) > 0 {
				fmt.Println("\nTo retrieve a test secret:")
				fmt.Printf("  keysync get TEST_SECRET_1 --project %s\n", project)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&testCount, "count", "c", 3, "number of test secrets to generate")
	return cmd
}

func generateTestValue() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random value: %w", err)
	}
	return hex.EncodeToString(b), nil
}
