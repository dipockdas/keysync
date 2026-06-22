package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dipockdas/keysync/internal/share/ksx"
	wormholepkg "github.com/dipockdas/keysync/internal/share/wormhole"
	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	safetyProjectName = "safety-app"
	safetySecretName  = "SAFETY_SECRET"
	safetySecretValue = "synthetic-safety-secret-value"
)

func TestShareSafetyRegressionCoversFileAndWormholeTransports(t *testing.T) {
	defer setupTest(t)()
	project = safetyProjectName
	if err := secretSt.Set(context.Background(), store.ScopeProject, project, "", safetySecretName, safetySecretValue); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 6, 23, 14, 0, 0, 0, time.UTC)
	passphrase := []byte("synthetic-safety-passphrase")
	restoreShare := stubShareDependencies(t, now, passphrase)
	defer restoreShare()

	t.Run("file", func(t *testing.T) {
		outPath := filepath.Join(t.TempDir(), "safety.keysync.ksx")
		cmd := newShareCmd()
		var output bytes.Buffer
		cmd.SetIn(strings.NewReader("SHARE\n"))
		cmd.SetOut(&output)
		cmd.SetErr(&output)
		cmd.SetArgs([]string{"--file", "--out", outPath})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("file share error = %v", err)
		}
		bundle, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatal(err)
		}
		assertShareSafetyBundle(t, bundle, output.String(), nil)
	})

	t.Run("wormhole", func(t *testing.T) {
		transport := &fakeCommandWormhole{sendCode: "7-purple-dolphin", sendResult: wormholepkg.Result{OK: true}}
		restoreTransport := stubCommandWormhole(t, transport)
		defer restoreTransport()
		cmd := newShareCmd()
		var output bytes.Buffer
		cmd.SetIn(strings.NewReader("SHARE\n"))
		cmd.SetOut(&output)
		cmd.SetErr(&output)
		cmd.SetArgs([]string{"--wormhole"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("wormhole share error = %v", err)
		}
		assertShareSafetyBundle(t, transport.sendData, output.String(), nil)
	})
}

func TestShareSafetyErrorsNeverContainSecretValues(t *testing.T) {
	defer setupTest(t)()
	project = safetyProjectName
	if err := secretSt.Set(context.Background(), store.ScopeProject, project, "", safetySecretName, safetySecretValue); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 6, 23, 14, 0, 0, 0, time.UTC)
	passphrase := []byte("synthetic-safety-passphrase")
	bundlePath := writeAcceptTestBundle(t, now, passphrase, []ksx.Secret{
		{Name: safetySecretName, Value: safetySecretValue, Scope: store.ScopeProject, Project: safetyProjectName},
	})

	t.Run("share cancellation", func(t *testing.T) {
		cmd := newShareCmd()
		cmd.SetIn(strings.NewReader("share\n"))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{"--out", filepath.Join(t.TempDir(), "cancelled.keysync.ksx")})
		err := cmd.Execute()
		assertNoSafetySecretLeak(t, err, "")
	})

	t.Run("accept wrong passphrase", func(t *testing.T) {
		restore := stubAcceptDependencies(t, now, []byte("synthetic-wrong-passphrase"))
		defer restore()
		cmd := newAcceptCmd()
		cmd.SetIn(strings.NewReader("ACCEPT\n"))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{bundlePath})
		err := cmd.Execute()
		assertNoSafetySecretLeak(t, err, "")
	})

	t.Run("wormhole timeout", func(t *testing.T) {
		restoreShare := stubShareDependencies(t, now, passphrase)
		defer restoreShare()
		transport := &fakeCommandWormhole{sendCode: "7-purple-dolphin", sendResult: wormholepkg.Result{Err: wormholepkg.ErrTimeout}}
		restoreTransport := stubCommandWormhole(t, transport)
		defer restoreTransport()
		cmd := newShareCmd()
		cmd.SetIn(strings.NewReader("SHARE\n"))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{"--wormhole"})
		err := cmd.Execute()
		assertNoSafetySecretLeak(t, err, "")
	})
}

func TestWriteBundleAtomicRemovesTempFileOnFailure(t *testing.T) {
	dir := t.TempDir()
	blockingPath := filepath.Join(dir, "bundle.keysync.ksx")
	if err := os.Mkdir(blockingPath, 0o755); err != nil {
		t.Fatal(err)
	}
	err := writeBundleAtomic(blockingPath, []byte("synthetic-encrypted-bundle"))
	if err == nil {
		t.Fatal("expected writeBundleAtomic failure")
	}
	matches, globErr := filepath.Glob(filepath.Join(dir, ".keysync-share-*.tmp"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}
}

func TestShareAcceptCommandsHaveNoPassphraseFlags(t *testing.T) {
	for _, cmd := range []*cobra.Command{newShareCmd(), newAcceptCmd()} {
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			name := strings.ToLower(flag.Name)
			if strings.Contains(name, "pass") || strings.Contains(name, "password") {
				t.Fatalf("command %q exposes passphrase flag %q", cmd.Name(), flag.Name)
			}
		})
	}
}

func TestShareWormholeAsyncCompletionIsRaceSafe(t *testing.T) {
	defer setupTest(t)()
	project = safetyProjectName
	if err := secretSt.Set(context.Background(), store.ScopeProject, project, "", safetySecretName, safetySecretValue); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 6, 23, 14, 0, 0, 0, time.UTC)
	restoreShare := stubShareDependencies(t, now, []byte("synthetic-safety-passphrase"))
	defer restoreShare()

	done := make(chan wormholepkg.Result, 1)
	transport := &delayedFakeCommandWormhole{
		sendCode: "7-purple-dolphin",
		result:   done,
	}
	restoreTransport := stubCommandWormhole(t, transport)
	defer restoreTransport()

	cmd := newShareCmd()
	cmd.SetIn(strings.NewReader("SHARE\n"))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{"--wormhole"})
	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Execute()
	}()
	done <- wormholepkg.Result{OK: true}
	if err := <-errCh; err != nil {
		t.Fatalf("async wormhole share error = %v", err)
	}
}

type delayedFakeCommandWormhole struct {
	sendCode string
	result   chan wormholepkg.Result
}

func (f *delayedFakeCommandWormhole) Send(_ context.Context, _ string, _ []byte) (string, <-chan wormholepkg.Result, error) {
	result := make(chan wormholepkg.Result, 1)
	go func() {
		result <- <-f.result
	}()
	return f.sendCode, result, nil
}

func (f *delayedFakeCommandWormhole) Receive(_ context.Context, _ string) (string, []byte, error) {
	return "", nil, wormholepkg.ErrTimeout
}

func assertShareSafetyBundle(t *testing.T, bundle []byte, output string, shareErr error) {
	t.Helper()
	for _, sensitive := range []string{safetySecretName, safetySecretValue} {
		if bytes.Contains(bundle, []byte(sensitive)) {
			t.Fatalf("bundle contains plaintext %q", sensitive)
		}
	}
	assertNoSafetySecretLeak(t, shareErr, output)
	if !strings.Contains(output, "Values: hidden") {
		t.Fatalf("output missing hidden-values marker: %s", output)
	}
	if strings.Contains(output, "Type ACCEPT") || strings.Contains(output, "IMPORT") {
		t.Fatalf("share output uses wrong confirmation vocabulary: %s", output)
	}
}

func assertNoSafetySecretLeak(t *testing.T, err error, output string) {
	t.Helper()
	for _, sensitive := range []string{safetySecretValue, "synthetic-safety-passphrase", "synthetic-wrong-passphrase"} {
		if strings.Contains(output, sensitive) {
			t.Fatalf("output leaks %q: %s", sensitive, output)
		}
		if err != nil && strings.Contains(err.Error(), sensitive) {
			t.Fatalf("error leaks %q: %v", sensitive, err)
		}
	}
}
