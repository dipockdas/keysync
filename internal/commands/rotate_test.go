package commands

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/dipockdas/keysync/internal/github"
	"github.com/dipockdas/keysync/internal/store"
)

func TestRotateCmd_ProjectScope(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	envFlag = ""
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "ROTATE_KEY", "old-proj-val")

	// Suppress github error by mocking exec to succeed silently
	defer github.SetExecCommandForTesting(func(name string, arg ...string) *exec.Cmd {
		return shellCommand("exit 0")
	})()

	cmd := newRotateCmd()
	stdout, stderr, err := captureCommand(cmd, []string{"ROTATE_KEY"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "Rotation complete") {
		t.Errorf("stdout missing completion: %s", stdout)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}

	// Verify value changed at project scope
	val, err := secretSt.Get(ctx, store.ScopeProject, "test-app", "", "ROTATE_KEY")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val == "old-proj-val" {
		t.Error("value was not rotated")
	}
	if len(val) == 0 {
		t.Error("rotated value is empty")
	}
}

func TestRotateCmd_EnvScope(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	envFlag = "staging"
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeProject, "test-app", "staging", "ROTATE_KEY", "old-env-val")

	defer github.SetExecCommandForTesting(func(name string, arg ...string) *exec.Cmd {
		return shellCommand("exit 0")
	})()

	cmd := newRotateCmd()
	stdout, stderr, err := captureCommand(cmd, []string{"ROTATE_KEY"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "Rotation complete") {
		t.Errorf("stdout missing completion: %s", stdout)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}

	// Verify value changed at env scope
	val, err := secretSt.Get(ctx, store.ScopeProject, "test-app", "staging", "ROTATE_KEY")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val == "old-env-val" {
		t.Error("value was not rotated")
	}
}

func TestRotateCmd_InvalidKey(t *testing.T) {
	defer setupTest(t)()

	cmd := newRotateCmd()
	_, _, err := captureCommand(cmd, []string{"123invalid"})
	if err == nil {
		t.Fatal("expected error for invalid key, got nil")
	}
	if !strings.Contains(err.Error(), "cannot start with a digit") {
		t.Errorf("error = %q, want 'cannot start with a digit'", err.Error())
	}
}
