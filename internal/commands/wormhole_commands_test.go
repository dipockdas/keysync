package commands

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dipockdas/keysync/internal/share/ksx"
	wormholepkg "github.com/dipockdas/keysync/internal/share/wormhole"
	"github.com/dipockdas/keysync/internal/store"
)

func TestShareWormholeUsesSharedEncryptedPipeline(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	secretSt.Set(context.Background(), store.ScopeProject, project, "", "SYNTHETIC_KEY", "synthetic-value")
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	restoreShare := stubShareDependencies(t, now, []byte("synthetic-passphrase"))
	defer restoreShare()
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
		t.Fatalf("share --wormhole error = %v", err)
	}
	if transport.sendFilename != "test-app.keysync.ksx" {
		t.Fatalf("filename = %q", transport.sendFilename)
	}
	payload, err := ksx.Decrypt(transport.sendData, []byte("synthetic-passphrase"), now)
	if err != nil {
		t.Fatalf("sent data is not a KSX bundle: %v", err)
	}
	if len(payload.Secrets) != 1 || payload.Secrets[0].Value != "synthetic-value" {
		t.Fatalf("payload = %#v", payload)
	}
	for _, expected := range []string{"Values: hidden", "7-purple-dolphin", "Waiting for recipient", "Transfer complete"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("output missing %q: %s", expected, output.String())
		}
	}
	if strings.Contains(output.String(), "synthetic-value") || strings.Contains(output.String(), "synthetic-passphrase") {
		t.Fatalf("output exposes sensitive data: %s", output.String())
	}
}

func TestShareWormholeFailureSuggestsFileModeWithoutLeakingValues(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	secretSt.Set(context.Background(), store.ScopeProject, project, "", "SYNTHETIC_KEY", "synthetic-value")
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	restoreShare := stubShareDependencies(t, now, []byte("synthetic-passphrase"))
	defer restoreShare()
	transport := &fakeCommandWormhole{sendCode: "7-purple-dolphin", sendResult: wormholepkg.Result{Err: wormholepkg.ErrTimeout}}
	restoreTransport := stubCommandWormhole(t, transport)
	defer restoreTransport()

	cmd := newShareCmd()
	var output bytes.Buffer
	cmd.SetIn(strings.NewReader("SHARE\n"))
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"--wormhole"})
	err := cmd.Execute()
	if !errors.Is(err, wormholepkg.ErrTimeout) {
		t.Fatalf("error = %v, want ErrTimeout", err)
	}
	if !strings.Contains(err.Error(), "keysync share -p test-app --file") {
		t.Fatalf("error lacks file fallback: %v", err)
	}
	if strings.Contains(err.Error(), "synthetic-value") || strings.Contains(output.String(), "synthetic-value") {
		t.Fatal("Wormhole failure exposes secret value")
	}
}

func TestAcceptWormholeCodeUsesSharedAcceptPipeline(t *testing.T) {
	defer setupTest(t)()
	now := acceptTestTime()
	passphrase := []byte("synthetic-accept-passphrase")
	bundlePath := writeAcceptTestBundle(t, now, passphrase, []ksx.Secret{
		{Name: "PROJECT_KEY", Value: "synthetic-project-value", Scope: store.ScopeProject, Project: "shared-app"},
	})
	bundle, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	transport := &fakeCommandWormhole{receiveFilename: "shared-app.keysync.ksx", receiveData: bundle}
	restoreTransport := stubCommandWormhole(t, transport)
	defer restoreTransport()
	restoreAccept := stubAcceptDependencies(t, now, passphrase)
	defer restoreAccept()

	cmd := newAcceptCmd()
	var output bytes.Buffer
	cmd.SetIn(strings.NewReader("ACCEPT\n"))
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"7-purple-dolphin"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("accept code error = %v", err)
	}
	got, err := secretSt.Get(context.Background(), store.ScopeProject, "shared-app", "", "PROJECT_KEY")
	if err != nil || got != "synthetic-project-value" {
		t.Fatalf("imported value = %q, %v", got, err)
	}
	if transport.receiveCode != "7-purple-dolphin" {
		t.Fatalf("receive code = %q", transport.receiveCode)
	}
	if !strings.Contains(output.String(), "Received encrypted share via Wormhole") || strings.Contains(output.String(), "synthetic-project-value") {
		t.Fatalf("unexpected output: %s", output.String())
	}
}

func TestAcceptWormholeValidatesKSXBeforePassphrase(t *testing.T) {
	defer setupTest(t)()
	transport := &fakeCommandWormhole{receiveFilename: "bad.keysync.ksx", receiveData: []byte("not-a-ksx-bundle")}
	restoreTransport := stubCommandWormhole(t, transport)
	defer restoreTransport()
	originalReader := readAcceptPassphrase
	passphraseCalls := 0
	readAcceptPassphrase = func() ([]byte, error) {
		passphraseCalls++
		return nil, errors.New("must not be called")
	}
	defer func() { readAcceptPassphrase = originalReader }()

	cmd := newAcceptCmd()
	cmd.SetArgs([]string{"7-purple-dolphin"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("invalid received KSX unexpectedly succeeded")
	}
	if passphraseCalls != 0 {
		t.Fatalf("passphrase reader called %d times", passphraseCalls)
	}
}

func TestAcceptWormholeFailuresImportNothingAndSuggestFileMode(t *testing.T) {
	for _, transportErr := range []error{wormholepkg.ErrTimeout, wormholepkg.ErrRejected, wormholepkg.ErrInterrupted, wormholepkg.ErrUnavailable, wormholepkg.ErrInvalidCode} {
		t.Run(transportErr.Error(), func(t *testing.T) {
			defer setupTest(t)()
			transport := &fakeCommandWormhole{receiveErr: transportErr}
			restoreTransport := stubCommandWormhole(t, transport)
			defer restoreTransport()
			cmd := newAcceptCmd()
			cmd.SetArgs([]string{"7-purple-dolphin"})
			err := cmd.Execute()
			if !errors.Is(err, transportErr) {
				t.Fatalf("error = %v, want %v", err, transportErr)
			}
			if !strings.Contains(err.Error(), "--file") {
				t.Fatalf("error lacks file fallback: %v", err)
			}
			assertSecretMissing(t, secretSt, store.ScopeProject, "shared-app", "PROJECT_KEY")
		})
	}
}

func TestAcceptExistingFileDoesNotCallWormhole(t *testing.T) {
	defer setupTest(t)()
	now := acceptTestTime()
	passphrase := []byte("synthetic-passphrase")
	bundlePath := writeAcceptTestBundle(t, now, passphrase, []ksx.Secret{
		{Name: "PROJECT_KEY", Value: "synthetic-value", Scope: store.ScopeProject, Project: "shared-app"},
	})
	transport := &fakeCommandWormhole{receiveErr: errors.New("must not be called")}
	restoreTransport := stubCommandWormhole(t, transport)
	defer restoreTransport()
	restoreAccept := stubAcceptDependencies(t, now, passphrase)
	defer restoreAccept()
	cmd := newAcceptCmd()
	cmd.SetIn(strings.NewReader("ACCEPT\n"))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{bundlePath})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if transport.receiveCalls != 0 {
		t.Fatalf("Wormhole receive called %d times", transport.receiveCalls)
	}
}

type fakeCommandWormhole struct {
	sendCode        string
	sendResult      wormholepkg.Result
	sendErr         error
	sendFilename    string
	sendData        []byte
	receiveCode     string
	receiveFilename string
	receiveData     []byte
	receiveErr      error
	receiveCalls    int
}

func (f *fakeCommandWormhole) Send(_ context.Context, filename string, data []byte) (string, <-chan wormholepkg.Result, error) {
	f.sendFilename = filename
	f.sendData = append([]byte(nil), data...)
	if f.sendErr != nil {
		return "", nil, f.sendErr
	}
	result := make(chan wormholepkg.Result, 1)
	result <- f.sendResult
	close(result)
	return f.sendCode, result, nil
}

func (f *fakeCommandWormhole) Receive(_ context.Context, code string) (string, []byte, error) {
	f.receiveCalls++
	f.receiveCode = code
	return f.receiveFilename, append([]byte(nil), f.receiveData...), f.receiveErr
}

func stubCommandWormhole(t *testing.T, transport wormholepkg.Transport) func() {
	t.Helper()
	original := commandWormhole
	commandWormhole = transport
	return func() { commandWormhole = original }
}
