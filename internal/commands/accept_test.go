package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dipockdas/keysync/internal/share/ksx"
	"github.com/dipockdas/keysync/internal/store"
)

func TestAcceptFileImportsNewKeysAndSkipsExistingKeys(t *testing.T) {
	defer setupTest(t)()
	now := acceptTestTime()
	passphrase := []byte("synthetic-accept-passphrase")
	bundlePath := writeAcceptTestBundle(t, now, passphrase, []ksx.Secret{
		{Name: "PROJECT_KEY", Value: "synthetic-project-value", Scope: store.ScopeProject, Project: "shared-app"},
		{Name: "GLOBAL_KEY", Value: "synthetic-incoming-global", Scope: store.ScopeGlobal},
	})
	if err := secretSt.Set(context.Background(), store.ScopeGlobal, "", "", "GLOBAL_KEY", "synthetic-existing-global"); err != nil {
		t.Fatal(err)
	}
	restore := stubAcceptDependencies(t, now, passphrase)
	defer restore()

	cmd := newAcceptCmd()
	var output bytes.Buffer
	cmd.SetIn(strings.NewReader("ACCEPT\n"))
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{bundlePath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("accept error = %v", err)
	}

	got, err := secretSt.Get(context.Background(), store.ScopeProject, "shared-app", "", "PROJECT_KEY")
	if err != nil || got != "synthetic-project-value" {
		t.Fatalf("project secret = %q, %v", got, err)
	}
	got, err = secretSt.Get(context.Background(), store.ScopeGlobal, "", "", "GLOBAL_KEY")
	if err != nil || got != "synthetic-existing-global" {
		t.Fatalf("existing global secret = %q, %v", got, err)
	}
	for _, expected := range []string{"Encrypted keysync share bundle detected", "Values: hidden", "PROJECT_KEY", "GLOBAL_KEY", "Imported 1 key", "Skipped 1 existing key"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("output missing %q: %s", expected, output.String())
		}
	}
	for _, value := range []string{"synthetic-project-value", "synthetic-incoming-global", "synthetic-existing-global"} {
		if strings.Contains(output.String(), value) {
			t.Fatalf("output leaks value %q: %s", value, output.String())
		}
	}
}

func TestAcceptRejectsExpiredBundleBeforePassphrase(t *testing.T) {
	defer setupTest(t)()
	created := acceptTestTime()
	bundlePath := writeAcceptTestBundle(t, created, []byte("synthetic-passphrase"), []ksx.Secret{
		{Name: "PROJECT_KEY", Value: "synthetic-value", Scope: store.ScopeProject, Project: "shared-app"},
	})
	originalNow := acceptNow
	originalReader := readAcceptPassphrase
	acceptNow = func() time.Time { return created.Add(ksx.FileTTL) }
	passphraseCalls := 0
	readAcceptPassphrase = func() ([]byte, error) {
		passphraseCalls++
		return nil, errors.New("must not be called")
	}
	defer func() {
		acceptNow = originalNow
		readAcceptPassphrase = originalReader
	}()

	cmd := newAcceptCmd()
	cmd.SetArgs([]string{bundlePath})
	err := cmd.Execute()
	if !errors.Is(err, ksx.ErrExpired) {
		t.Fatalf("error = %v, want ErrExpired", err)
	}
	if passphraseCalls != 0 {
		t.Fatalf("passphrase reader called %d times", passphraseCalls)
	}
	if !strings.Contains(err.Error(), created.Format(time.RFC3339)) || !strings.Contains(err.Error(), created.Add(ksx.FileTTL).Format(time.RFC3339)) {
		t.Fatalf("expiry error lacks timestamps: %v", err)
	}
	assertSecretMissing(t, secretSt, store.ScopeProject, "shared-app", "PROJECT_KEY")
}

func TestAcceptAuthenticationAndFormatFailuresWriteNothing(t *testing.T) {
	defer setupTest(t)()
	now := acceptTestTime()
	correctPassphrase := []byte("synthetic-correct-passphrase")
	validPath := writeAcceptTestBundle(t, now, correctPassphrase, []ksx.Secret{
		{Name: "PROJECT_KEY", Value: "synthetic-value", Scope: store.ScopeProject, Project: "shared-app"},
	})
	validBundle, err := os.ReadFile(validPath)
	if err != nil {
		t.Fatal(err)
	}
	tampered := append([]byte(nil), validBundle...)
	tampered[len(tampered)-8] ^= 1
	tamperedPath := filepath.Join(t.TempDir(), "tampered.keysync.ksx")
	if err := os.WriteFile(tamperedPath, tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	malformedPath := filepath.Join(t.TempDir(), "malformed.keysync.ksx")
	if err := os.WriteFile(malformedPath, []byte("not a bundle"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		path       string
		passphrase []byte
	}{
		{name: "wrong passphrase", path: validPath, passphrase: []byte("synthetic-wrong-passphrase")},
		{name: "tampered", path: tamperedPath, passphrase: correctPassphrase},
		{name: "malformed", path: malformedPath, passphrase: correctPassphrase},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := stubAcceptDependencies(t, now, tt.passphrase)
			defer restore()
			cmd := newAcceptCmd()
			var output bytes.Buffer
			cmd.SetIn(strings.NewReader("ACCEPT\n"))
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs([]string{tt.path})
			if err := cmd.Execute(); err == nil {
				t.Fatal("expected accept failure")
			}
			if strings.Contains(output.String(), "synthetic-value") {
				t.Fatalf("output leaks secret: %s", output.String())
			}
			assertSecretMissing(t, secretSt, store.ScopeProject, "shared-app", "PROJECT_KEY")
		})
	}
}

func TestAcceptRequiresExactConfirmationAndValidatesAllKeysBeforeWriting(t *testing.T) {
	defer setupTest(t)()
	now := acceptTestTime()
	passphrase := []byte("synthetic-passphrase")

	t.Run("confirmation", func(t *testing.T) {
		bundlePath := writeAcceptTestBundle(t, now, passphrase, []ksx.Secret{
			{Name: "PROJECT_KEY", Value: "synthetic-value", Scope: store.ScopeProject, Project: "shared-app"},
		})
		restore := stubAcceptDependencies(t, now, passphrase)
		defer restore()
		cmd := newAcceptCmd()
		cmd.SetIn(strings.NewReader("accept\n"))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{bundlePath})
		if err := cmd.Execute(); err == nil {
			t.Fatal("lowercase confirmation unexpectedly succeeded")
		}
		assertSecretMissing(t, secretSt, store.ScopeProject, "shared-app", "PROJECT_KEY")
	})

	t.Run("mismatched project", func(t *testing.T) {
		bundlePath := writeAcceptTestBundle(t, now, passphrase, []ksx.Secret{
			{Name: "FIRST_KEY", Value: "synthetic-first", Scope: store.ScopeProject, Project: "shared-app"},
			{Name: "INVALID_KEY", Value: "synthetic-invalid", Scope: store.ScopeProject, Project: "other-app"},
		})
		restore := stubAcceptDependencies(t, now, passphrase)
		defer restore()
		cmd := newAcceptCmd()
		cmd.SetIn(strings.NewReader("ACCEPT\n"))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{bundlePath})
		if err := cmd.Execute(); err == nil {
			t.Fatal("invalid payload unexpectedly succeeded")
		}
		assertSecretMissing(t, secretSt, store.ScopeProject, "shared-app", "FIRST_KEY")
		assertSecretMissing(t, secretSt, store.ScopeProject, "other-app", "INVALID_KEY")
	})
}

func TestAcceptRollsBackPriorWritesWhenStoreSetFails(t *testing.T) {
	defer setupTest(t)()
	now := acceptTestTime()
	passphrase := []byte("synthetic-passphrase")
	bundlePath := writeAcceptTestBundle(t, now, passphrase, []ksx.Secret{
		{Name: "FIRST_KEY", Value: "synthetic-first", Scope: store.ScopeProject, Project: "shared-app"},
		{Name: "SECOND_KEY", Value: "synthetic-second", Scope: store.ScopeProject, Project: "shared-app"},
	})
	baseStore := secretSt
	secretSt = &failingSetStore{Store: baseStore, failKey: "SECOND_KEY"}
	restore := stubAcceptDependencies(t, now, passphrase)
	defer restore()

	cmd := newAcceptCmd()
	var output bytes.Buffer
	cmd.SetIn(strings.NewReader("ACCEPT\n"))
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{bundlePath})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "rolled back 1 prior writes") {
		t.Fatalf("error = %v", err)
	}
	for _, value := range []string{"synthetic-first", "synthetic-second"} {
		if strings.Contains(err.Error(), value) || strings.Contains(output.String(), value) {
			t.Fatalf("failure leaks value %q", value)
		}
	}
	assertSecretMissing(t, baseStore, store.ScopeProject, "shared-app", "FIRST_KEY")
	assertSecretMissing(t, baseStore, store.ScopeProject, "shared-app", "SECOND_KEY")
}

func TestAcceptRejectsMissingAndNonRegularFiles(t *testing.T) {
	defer setupTest(t)()
	for _, path := range []string{filepath.Join(t.TempDir(), "missing.keysync.ksx"), t.TempDir()} {
		cmd := newAcceptCmd()
		cmd.SetArgs([]string{path})
		if err := cmd.Execute(); err == nil {
			t.Fatalf("path %q unexpectedly succeeded", path)
		}
	}
}

func acceptTestTime() time.Time {
	return time.Date(2026, 6, 23, 11, 0, 0, 0, time.UTC)
}

func writeAcceptTestBundle(t *testing.T, now time.Time, passphrase []byte, secrets []ksx.Secret) string {
	t.Helper()
	bundle, err := ksx.Encrypt(ksx.Payload{
		Project:   "shared-app",
		CreatedAt: now,
		ExpiresAt: now.Add(ksx.FileTTL),
		Secrets:   secrets,
	}, passphrase)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "shared-app.keysync.ksx")
	if err := os.WriteFile(path, bundle, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func stubAcceptDependencies(t *testing.T, now time.Time, passphrase []byte) func() {
	t.Helper()
	originalNow := acceptNow
	originalPassphrase := readAcceptPassphrase
	acceptNow = func() time.Time { return now }
	readAcceptPassphrase = func() ([]byte, error) { return append([]byte(nil), passphrase...), nil }
	return func() {
		acceptNow = originalNow
		readAcceptPassphrase = originalPassphrase
	}
}

type failingSetStore struct {
	store.Store
	failKey string
}

func (s *failingSetStore) Set(ctx context.Context, scope store.Scope, project, environment, key, value string) error {
	if key == s.failKey {
		return fmt.Errorf("synthetic store failure")
	}
	return s.Store.Set(ctx, scope, project, environment, key, value)
}

func assertSecretMissing(t *testing.T, secretStore store.Store, scope store.Scope, project, key string) {
	t.Helper()
	_, err := secretStore.Get(context.Background(), scope, project, "", key)
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("secret %s unexpectedly exists: %v", key, err)
	}
}
