package commands

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/dipockdas/keysync/internal/github"
	"github.com/dipockdas/keysync/internal/store"
)

func TestPullCmd_AllFound(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "SECRET_1", "val1")
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "SECRET_2", "val2")

	// Mock github to return matching secrets
	defer github.SetExecCommandForTesting(func(name string, arg ...string) *exec.Cmd {
		return mockOutput(t, `[{"name":"SECRET_1"},{"name":"SECRET_2"}]`)
	})()

	cmd := newPullCmd()
	stdout, stderr, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "All secrets are present") {
		t.Errorf("stdout = %q, want 'All secrets are present'", stdout)
	}
	if !strings.Contains(stdout, "SECRET_1") {
		t.Errorf("stdout missing SECRET_1: %s", stdout)
	}
	if !strings.Contains(stdout, "SECRET_2") {
		t.Errorf("stdout missing SECRET_2: %s", stdout)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
}

func TestPullCmd_SomeMissing(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	// Only one of the two github secrets exists locally
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "SECRET_1", "val1")

	defer github.SetExecCommandForTesting(func(name string, arg ...string) *exec.Cmd {
		return mockOutput(t, `[{"name":"SECRET_1"},{"name":"SECRET_2"}]`)
	})()

	cmd := newPullCmd()
	stdout, stderr, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "MISSING") {
		t.Errorf("stdout missing MISSING: %s", stdout)
	}
	if !strings.Contains(stderr, "missing") {
		t.Errorf("stderr missing count: %s", stderr)
	}
}

func TestPullCmd_ProjectScopeCheck(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	envFlag = ""
	ctx := context.Background()
	// Secret is in project scope, not global
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "SECRET_1", "val1")

	defer github.SetExecCommandForTesting(func(name string, arg ...string) *exec.Cmd {
		return mockOutput(t, `[{"name":"SECRET_1"}]`)
	})()

	cmd := newPullCmd()
	stdout, stderr, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "OK") {
		t.Errorf("stdout missing OK: %s", stdout)
	}
	if !strings.Contains(stdout, "project/test-app") {
		t.Errorf("stdout missing scope: %s", stdout)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
}

func TestPullCmd_EnvScopeCheck(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	envFlag = "production"
	ctx := context.Background()
	// Secret is in env scope, not global or project
	secretSt.Set(ctx, store.ScopeProject, "test-app", "production", "SECRET_1", "val1")

	defer github.SetExecCommandForTesting(func(name string, arg ...string) *exec.Cmd {
		return mockOutput(t, `[{"name":"SECRET_1"}]`)
	})()

	cmd := newPullCmd()
	stdout, stderr, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "OK") {
		t.Errorf("stdout missing OK: %s", stdout)
	}
	if !strings.Contains(stdout, "project/test-app/production") {
		t.Errorf("stdout missing scope: %s", stdout)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
}

func TestPullCmd_EmptyGithub(t *testing.T) {
	defer setupTest(t)()

	defer github.SetExecCommandForTesting(func(name string, arg ...string) *exec.Cmd {
		return mockOutput(t, `[]`)
	})()

	cmd := newPullCmd()
	stdout, stderr, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "No secrets found in GitHub") {
		t.Errorf("stdout = %q, want 'No secrets found in GitHub'", stdout)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
}
