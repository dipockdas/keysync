package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	sharepkg "github.com/dipockdas/keysync/internal/share"
	"github.com/dipockdas/keysync/internal/share/ksx"
	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	acceptNow            = time.Now
	readAcceptPassphrase = readAcceptPassphraseTerminal
)

func newAcceptCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "accept <file-or-code>",
		Short:        "Accept an encrypted, short-lived secret share",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAccept(cmd, args[0])
		},
	}
}

func runAccept(cmd *cobra.Command, source string) error {
	info, err := os.Stat(source)
	if err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("accept source is not a regular file")
		}
		bundle, readErr := readBundleFile(source)
		if readErr != nil {
			return readErr
		}
		return runAcceptBundle(cmd, bundle, "Encrypted keysync share bundle detected.")
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect accept source: %w", err)
	}
	_, bundle, err := commandWormhole.Receive(cmd.Context(), source)
	if err != nil {
		return wormholeAcceptError(err)
	}
	return runAcceptBundle(cmd, bundle, "Received encrypted share via Wormhole.")
}

func runAcceptBundle(cmd *cobra.Command, bundle []byte, detectedMessage string) error {
	now := acceptNow().UTC()
	metadata, err := ksx.Inspect(bundle, now)
	if err != nil {
		if errors.Is(err, ksx.ErrExpired) {
			return fmt.Errorf("%w\nCreated: %s\nExpired: %s\nAsk the sender to create a new share", err, metadata.CreatedAt.Format(time.RFC3339), metadata.ExpiresAt.Format(time.RFC3339))
		}
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, detectedMessage)
	passphrase, err := readAcceptPassphrase()
	if err != nil {
		return fmt.Errorf("read share passphrase: %w", err)
	}
	defer clear(passphrase)
	if len(passphrase) == 0 {
		return fmt.Errorf("read share passphrase: passphrase must not be empty")
	}

	payload, err := ksx.Decrypt(bundle, passphrase, now)
	if err != nil {
		return err
	}
	if err := validateAcceptPayload(payload); err != nil {
		return err
	}

	preview := sharepkg.Plan{Project: payload.Project, Secrets: payload.Secrets}.Preview()
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Bundle contents:")
	fmt.Fprintln(out)
	fprintPreview(out, preview)
	fmt.Fprintf(out, "Created at: %s\n", payload.CreatedAt.Format(time.RFC3339))
	fmt.Fprintln(out)
	fmt.Fprint(out, "Type ACCEPT to continue: ")
	confirmation, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read accept confirmation: %w", err)
	}
	confirmation = strings.TrimSuffix(strings.TrimSuffix(confirmation, "\n"), "\r")
	if confirmation != "ACCEPT" {
		return fmt.Errorf("accept cancelled: confirmation did not match ACCEPT")
	}

	toWrite, skipped, err := preflightAccept(cmd, payload.Secrets)
	if err != nil {
		return err
	}
	if err := writeAcceptedSecrets(cmd, toWrite); err != nil {
		return err
	}

	for _, secret := range skipped {
		fmt.Fprintf(out, "Skipped existing key: %s\n", secret.Name)
	}
	fmt.Fprintf(out, "Imported %d %s into the local keychain.\n", len(toWrite), plural(len(toWrite), "key", "keys"))
	if len(skipped) > 0 {
		fmt.Fprintf(out, "Skipped %d existing %s.\n", len(skipped), plural(len(skipped), "key", "keys"))
	}
	return nil
}

func wormholeAcceptError(err error) error {
	return fmt.Errorf("%w\n\nWormhole transfer failed. Ask the sender to create a file share with --file", err)
}

func readBundleFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open share bundle: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect share bundle: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("accept source is not a regular file")
	}
	if info.Size() <= 0 || info.Size() > ksx.MaxBundleSize {
		return nil, fmt.Errorf("share bundle size is invalid")
	}
	bundle, err := io.ReadAll(io.LimitReader(file, ksx.MaxBundleSize+1))
	if err != nil {
		return nil, fmt.Errorf("read share bundle: %w", err)
	}
	if len(bundle) > ksx.MaxBundleSize {
		return nil, fmt.Errorf("share bundle exceeds maximum size")
	}
	return bundle, nil
}

func validateAcceptPayload(payload ksx.Payload) error {
	seen := make(map[string]struct{}, len(payload.Secrets))
	for _, secret := range payload.Secrets {
		if err := validateKeyName(secret.Name); err != nil {
			return fmt.Errorf("invalid shared key name: %w", err)
		}
		if secret.Environment != "" {
			return fmt.Errorf("invalid share bundle: environment-scoped keys are not supported in v1")
		}
		switch secret.Scope {
		case store.ScopeProject:
			if secret.Project != payload.Project {
				return fmt.Errorf("invalid share bundle: project scope does not match bundle project")
			}
		case store.ScopeGlobal:
			if secret.Project != "" {
				return fmt.Errorf("invalid share bundle: global key contains a project")
			}
		default:
			return fmt.Errorf("invalid share bundle: unsupported secret scope")
		}
		target := fmt.Sprintf("%s\x00%s\x00%s", secret.Scope, secret.Project, secret.Name)
		if _, exists := seen[target]; exists {
			return fmt.Errorf("invalid share bundle: duplicate key %q", secret.Name)
		}
		seen[target] = struct{}{}
	}
	return nil
}

func preflightAccept(cmd *cobra.Command, secrets []ksx.Secret) (toWrite, skipped []ksx.Secret, err error) {
	type bucket struct {
		scope   store.Scope
		project string
	}
	buckets := make(map[bucket]map[string]struct{})
	for _, secret := range secrets {
		key := bucket{scope: secret.Scope, project: secret.Project}
		if _, loaded := buckets[key]; !loaded {
			entries, listErr := secretSt.List(cmd.Context(), secret.Scope, secret.Project, "")
			if listErr != nil {
				return nil, nil, fmt.Errorf("check existing keys before acceptance: %w", listErr)
			}
			names := make(map[string]struct{}, len(entries))
			for _, entry := range entries {
				if entry.Scope == secret.Scope && entry.Project == secret.Project && entry.Environment == "" {
					names[entry.Key] = struct{}{}
				}
			}
			buckets[key] = names
		}
		if _, exists := buckets[key][secret.Name]; exists {
			skipped = append(skipped, secret)
		} else {
			toWrite = append(toWrite, secret)
		}
	}
	sort.Slice(skipped, func(i, j int) bool { return skipped[i].Name < skipped[j].Name })
	return toWrite, skipped, nil
}

func writeAcceptedSecrets(cmd *cobra.Command, secrets []ksx.Secret) error {
	written := make([]ksx.Secret, 0, len(secrets))
	for _, secret := range secrets {
		if err := secretSt.Set(cmd.Context(), secret.Scope, secret.Project, "", secret.Name, secret.Value); err != nil {
			rollbackErr := rollbackAcceptedSecrets(cmd, written)
			if rollbackErr != nil {
				return fmt.Errorf("import key %q failed after %d writes; rollback also failed: %v", secret.Name, len(written), rollbackErr)
			}
			return fmt.Errorf("import key %q failed; rolled back %d prior writes: %w", secret.Name, len(written), err)
		}
		written = append(written, secret)
	}
	return nil
}

func rollbackAcceptedSecrets(cmd *cobra.Command, written []ksx.Secret) error {
	var failures []string
	for i := len(written) - 1; i >= 0; i-- {
		secret := written[i]
		if err := secretSt.Delete(cmd.Context(), secret.Scope, secret.Project, "", secret.Name); err != nil {
			failures = append(failures, secret.Name)
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("could not remove %d imported keys: %s", len(failures), strings.Join(failures, ", "))
	}
	return nil
}

func readAcceptPassphraseTerminal() ([]byte, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil, fmt.Errorf("interactive terminal required")
	}
	fmt.Fprint(os.Stderr, "Enter share passphrase: ")
	passphrase, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, err
	}
	if len(passphrase) == 0 {
		return nil, fmt.Errorf("passphrase must not be empty")
	}
	return passphrase, nil
}

func plural(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}
