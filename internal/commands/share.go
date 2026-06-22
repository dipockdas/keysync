package commands

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	sharepkg "github.com/dipockdas/keysync/internal/share"
	"github.com/dipockdas/keysync/internal/share/ksx"
	wormholepkg "github.com/dipockdas/keysync/internal/share/wormhole"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	shareNow            = time.Now
	readSharePassphrase = readSharePassphraseTerminal
	commandWormhole     = wormholepkg.New()
)

func newShareCmd() *cobra.Command {
	var key, outPath string
	var fileMode, wormholeMode bool

	cmd := &cobra.Command{
		Use:          "share --project name [--key KEY] [--file|--wormhole] [--out path]",
		Short:        "Create an encrypted, short-lived secret share",
		SilenceUsage: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("share accepts no positional arguments")
			}
			if len(args) == 1 && !mightBeTrailingProjectArg(args) {
				return fmt.Errorf("unknown argument %q — use --project NAME", args[0])
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			commandArgs(args)
			if project == "" || project == ProjectListSentinel {
				return fmt.Errorf("--project requires a project name")
			}
			if fileMode && wormholeMode {
				return fmt.Errorf("choose exactly one transport: --file or --wormhole")
			}
			if wormholeMode && outPath != "" {
				return fmt.Errorf("--out only applies to --file")
			}
			if outPath == "" {
				outPath = project + ".keysync.ksx"
			}
			if wormholeMode {
				return runWormholeShare(cmd, key)
			}
			return runFileShare(cmd, key, outPath)
		},
	}
	cmd.Flags().StringVarP(&key, "key", "k", "", "share one key instead of all project-wide keys")
	cmd.Flags().BoolVar(&fileMode, "file", false, "write an encrypted .ksx file (default)")
	cmd.Flags().BoolVar(&wormholeMode, "wormhole", false, "send through Magic Wormhole")
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "output path for file mode")
	return cmd
}

func runFileShare(cmd *cobra.Command, key, outPath string) error {
	bundle, err := prepareShareBundle(cmd, key)
	if err != nil {
		return err
	}
	if err := writeBundleAtomic(outPath, bundle); err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Created encrypted share bundle:")
	fmt.Fprintln(out, outPath)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Send this file separately from its passphrase. It expires in 10 minutes.")
	return nil
}

func runWormholeShare(cmd *cobra.Command, key string) error {
	bundle, err := prepareShareBundle(cmd, key)
	if err != nil {
		return err
	}
	filename := project + ".keysync.ksx"
	code, result, err := commandWormhole.Send(cmd.Context(), filename, bundle)
	if err != nil {
		return wormholeShareError(err)
	}
	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Wormhole code:")
	fmt.Fprintln(out, code)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Recipient should run:")
	fmt.Fprintf(out, "keysync accept %s\n", code)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Waiting for recipient...")
	transfer, ok := <-result
	if !ok {
		return wormholeShareError(wormholepkg.ErrInterrupted)
	}
	if transfer.Err != nil {
		return wormholeShareError(transfer.Err)
	}
	if !transfer.OK {
		return wormholeShareError(wormholepkg.ErrInterrupted)
	}
	fmt.Fprintln(out, "Transfer complete.")
	return nil
}

func prepareShareBundle(cmd *cobra.Command, key string) ([]byte, error) {
	ctx := cmd.Context()
	plan, err := sharepkg.BuildPlan(ctx, secretSt, project, key)
	if err != nil {
		return nil, err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Preparing share bundle")
	fmt.Fprintln(out)
	fprintPreview(out, plan.Preview())
	fmt.Fprintln(out)
	fmt.Fprint(out, "Type SHARE to continue: ")
	confirmation, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read share confirmation: %w", err)
	}
	confirmation = strings.TrimSuffix(strings.TrimSuffix(confirmation, "\n"), "\r")
	if confirmation != "SHARE" {
		return nil, fmt.Errorf("share cancelled: confirmation did not match SHARE")
	}

	passphrase, err := readSharePassphrase()
	if err != nil {
		return nil, fmt.Errorf("read share passphrase: %w", err)
	}
	defer clear(passphrase)
	if len(passphrase) == 0 {
		return nil, fmt.Errorf("read share passphrase: passphrase must not be empty")
	}

	secrets, err := plan.LoadSecrets(ctx, secretSt)
	if err != nil {
		return nil, err
	}
	now := shareNow().UTC()
	payload := ksx.Payload{
		Project:   project,
		CreatedAt: now,
		ExpiresAt: now.Add(ksx.FileTTL),
		Secrets:   secrets,
	}
	bundle, err := ksx.Encrypt(payload, passphrase)
	if err != nil {
		return nil, err
	}
	return bundle, nil
}

func wormholeShareError(err error) error {
	return fmt.Errorf("%w\n\nWormhole transfer failed. You can use file mode instead:\nkeysync share -p %s --file", err, project)
}

func fprintPreview(w io.Writer, preview sharepkg.Preview) {
	fmt.Fprint(w, preview.String())
}

func readSharePassphraseTerminal() ([]byte, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil, fmt.Errorf("interactive terminal required")
	}
	fmt.Fprint(os.Stderr, "Create passphrase for this share: ")
	first, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, err
	}
	if len(first) == 0 {
		return nil, fmt.Errorf("passphrase must not be empty")
	}
	fmt.Fprint(os.Stderr, "Confirm passphrase: ")
	second, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		clear(first)
		return nil, err
	}
	defer clear(second)
	if !bytes.Equal(first, second) {
		clear(first)
		return nil, fmt.Errorf("passphrases do not match")
	}
	return first, nil
}

func writeBundleAtomic(path string, bundle []byte) (returnErr error) {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("write share bundle: output already exists: %s", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("write share bundle: inspect output: %w", err)
	}
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".keysync-share-*.tmp")
	if err != nil {
		return fmt.Errorf("write share bundle: create temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		temporary.Close()
		if returnErr != nil {
			os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("write share bundle: restrict permissions: %w", err)
	}
	if _, err := temporary.Write(bundle); err != nil {
		return fmt.Errorf("write share bundle: write encrypted data: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("write share bundle: sync encrypted data: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("write share bundle: close encrypted data: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("write share bundle: publish output: %w", err)
	}
	return nil
}
