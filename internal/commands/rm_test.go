package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/dipockdas/keysync/internal/store"
)

func TestRmCmd_Global(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "MY_KEY", "val")

	cmd := newRmCmd()
	stdout, stderr, err := captureCommand(cmd, []string{"MY_KEY"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "Deleted global/MY_KEY") {
		t.Errorf("stdout = %q, want 'Deleted global/MY_KEY'", stdout)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}

	// Verify it's gone
	_, err = secretSt.Get(ctx, store.ScopeGlobal, "", "", "MY_KEY")
	if err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestRmCmd_ProjectScope(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	envFlag = "" // no env — delete at project scope
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "PROJ_KEY", "proj-val")

	cmd := newRmCmd()
	stdout, stderr, err := captureCommand(cmd, []string{"PROJ_KEY", "--env", ""})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "Deleted project/test-app/PROJ_KEY") {
		t.Errorf("stdout = %q, want 'Deleted project/test-app/PROJ_KEY'", stdout)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}

	_, err = secretSt.Get(ctx, store.ScopeProject, "test-app", "", "PROJ_KEY")
	if err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestRmCmd_ProjectEnvScope(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	envFlag = "staging"
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeProject, "test-app", "staging", "ENV_KEY", "env-val")

	cmd := newRmCmd()
	stdout, stderr, err := captureCommand(cmd, []string{"ENV_KEY", "--env", "staging"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "Deleted project/test-app/ENV_KEY") {
		t.Errorf("stdout = %q, want 'Deleted project/test-app/ENV_KEY'", stdout)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}

	_, err = secretSt.Get(ctx, store.ScopeProject, "test-app", "staging", "ENV_KEY")
	if err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestRmCmd_NotFound(t *testing.T) {
	defer setupTest(t)()

	cmd := newRmCmd()
	_, _, err := captureCommand(cmd, []string{"NONEXISTENT"})
	if err == nil {
		t.Fatal("expected error for nonexistent key, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want 'not found'", err.Error())
	}
}

func TestRmCmd_InvalidKeyName(t *testing.T) {
	defer setupTest(t)()

	cmd := newRmCmd()
	_, _, err := captureCommand(cmd, []string{"123invalid"})
	if err == nil {
		t.Fatal("expected error for invalid key name, got nil")
	}
	if !strings.Contains(err.Error(), "cannot start with a digit") {
		t.Errorf("error = %q, want 'cannot start with a digit'", err.Error())
	}
}

func TestRmCmd_EmptyKeyName(t *testing.T) {
	defer setupTest(t)()

	cmd := newRmCmd()
	_, _, err := captureCommand(cmd, []string{""})
	if err == nil {
		t.Fatal("expected error for missing key, got nil")
	}
}
