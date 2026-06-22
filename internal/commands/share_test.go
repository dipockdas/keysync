package commands

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dipockdas/keysync/internal/share/ksx"
	"github.com/dipockdas/keysync/internal/store"
)

func TestShareFileCreatesEncryptedBundleAfterExactConfirmation(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	secretSt.Set(context.Background(), store.ScopeProject, project, "", "SYNTHETIC_KEY", "synthetic-value")
	outPath := filepath.Join(t.TempDir(), "test-app.keysync.ksx")
	now := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	restore := stubShareDependencies(t, now, []byte("synthetic-passphrase"))
	defer restore()

	cmd := newShareCmd()
	var output bytes.Buffer
	cmd.SetIn(strings.NewReader("SHARE\n"))
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"--file", "--out", outPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("share error = %v", err)
	}

	bundle, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := ksx.Decrypt(bundle, []byte("synthetic-passphrase"), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload.Secrets) != 1 || payload.Secrets[0].Name != "SYNTHETIC_KEY" || payload.Secrets[0].Value != "synthetic-value" {
		t.Fatalf("payload = %#v", payload)
	}
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
	if strings.Contains(output.String(), "synthetic-value") {
		t.Fatalf("output leaks secret: %s", output.String())
	}
	for _, expected := range []string{"Values: hidden", "SYNTHETIC_KEY", "Created encrypted share bundle"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("output missing %q: %s", expected, output.String())
		}
	}
}

func TestShareDefaultsToFileAndSupportsOneKey(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	secretSt.Set(context.Background(), store.ScopeProject, project, "", "FIRST_KEY", "synthetic-first")
	secretSt.Set(context.Background(), store.ScopeProject, project, "", "SECOND_KEY", "synthetic-second")
	outPath := filepath.Join(t.TempDir(), "one.keysync.ksx")
	now := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	restore := stubShareDependencies(t, now, []byte("synthetic-passphrase"))
	defer restore()

	cmd := newShareCmd()
	cmd.SetIn(strings.NewReader("SHARE\n"))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--key", "SECOND_KEY", "--out", outPath})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	bundle, _ := os.ReadFile(outPath)
	payload, err := ksx.Decrypt(bundle, []byte("synthetic-passphrase"), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload.Secrets) != 1 || payload.Secrets[0].Name != "SECOND_KEY" {
		t.Fatalf("payload keys = %#v", payload.Secrets)
	}
}

func TestShareCancellationAndPassphraseFailureCreateNoFile(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	secretSt.Set(context.Background(), store.ScopeProject, project, "", "SYNTHETIC_KEY", "synthetic-value")

	t.Run("confirmation", func(t *testing.T) {
		outPath := filepath.Join(t.TempDir(), "cancelled.keysync.ksx")
		cmd := newShareCmd()
		cmd.SetIn(strings.NewReader("share\n"))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"--out", outPath})
		if err := cmd.Execute(); err == nil {
			t.Fatal("expected cancellation error")
		}
		if _, err := os.Stat(outPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("output exists after cancellation: %v", err)
		}
	})

	t.Run("passphrase", func(t *testing.T) {
		outPath := filepath.Join(t.TempDir(), "failed.keysync.ksx")
		original := readSharePassphrase
		readSharePassphrase = func() ([]byte, error) { return nil, errors.New("passphrases do not match") }
		defer func() { readSharePassphrase = original }()
		cmd := newShareCmd()
		cmd.SetIn(strings.NewReader("SHARE\n"))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"--out", outPath})
		if err := cmd.Execute(); err == nil {
			t.Fatal("expected passphrase error")
		}
		if _, err := os.Stat(outPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("output exists after passphrase failure: %v", err)
		}
	})
}

func TestShareRejectsInvalidFlags(t *testing.T) {
	defer setupTest(t)()
	tests := [][]string{
		{"--file", "--wormhole"},
		{"--wormhole", "--out", "bundle.ksx"},
	}
	for _, args := range tests {
		cmd := newShareCmd()
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil {
			t.Fatalf("args %v unexpectedly succeeded", args)
		}
	}
	project = ""
	cmd := newShareCmd()
	if err := cmd.Execute(); err == nil {
		t.Fatal("missing project unexpectedly succeeded")
	}
}

func stubShareDependencies(t *testing.T, now time.Time, passphrase []byte) func() {
	t.Helper()
	originalNow := shareNow
	originalPassphrase := readSharePassphrase
	shareNow = func() time.Time { return now }
	readSharePassphrase = func() ([]byte, error) { return append([]byte(nil), passphrase...), nil }
	return func() {
		shareNow = originalNow
		readSharePassphrase = originalPassphrase
	}
}
