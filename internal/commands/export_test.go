package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/dipockdas/keysync/internal/store"
)

// ---------------------------------------------------------------------------
// shellQuote tests
// ---------------------------------------------------------------------------

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal", "hello", "'hello'"},
		{"empty", "", "''"},
		{"with single quote", "it's", "'it'\\''s'"},
		{"multiple single quotes", "'a' 'b'", "''\\''a'\\'' '\\''b'\\'''"},
		{"special chars", "val=$HOME;echo hi", "'val=$HOME;echo hi'"},
		{"newlines", "line1\nline2", "'line1\nline2'"},
		{"only single quote", "'", "''\\'''"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellQuote(tt.input)
			if got != tt.want {
				t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Export command tests
// ---------------------------------------------------------------------------

func TestExportCmd_GlobalOnly(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "GLOBAL_KEY", "global-val")
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "ANOTHER_KEY", "another-val")

	cmd := newExportCmd()
	stdout, stderr, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %s", len(lines), stdout)
	}
	if !strings.Contains(stdout, "export GLOBAL_KEY=") {
		t.Errorf("stdout missing GLOBAL_KEY: %s", stdout)
	}
	if !strings.Contains(stdout, "export ANOTHER_KEY=") {
		t.Errorf("stdout missing ANOTHER_KEY: %s", stdout)
	}
}

func TestExportCmd_ProjectAndGlobal(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "GLOBAL_KEY", "global-val")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "PROJ_KEY", "proj-val")
	project = "test-app"

	cmd := newExportCmd()
	stdout, stderr, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}

	if !strings.Contains(stdout, "export GLOBAL_KEY=") {
		t.Errorf("stdout missing GLOBAL_KEY: %s", stdout)
	}
	if !strings.Contains(stdout, "export PROJ_KEY=") {
		t.Errorf("stdout missing PROJ_KEY: %s", stdout)
	}
}

func TestExportCmd_ProjectOverridesGlobal(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "SHARED_KEY", "global-val")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "SHARED_KEY", "proj-val")
	project = "test-app"

	cmd := newExportCmd()
	stdout, stderr, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}

	// Should only appear once with the project value
	if !strings.Contains(stdout, "SHARED_KEY=") {
		t.Errorf("stdout missing SHARED_KEY: %s", stdout)
	}
	if !strings.Contains(stdout, "proj-val") {
		t.Errorf("stdout should contain project value, got: %s", stdout)
	}
	if strings.Contains(stdout, "global-val") {
		t.Errorf("stdout should NOT contain global value, got: %s", stdout)
	}
}

func TestExportCmd_EnvOverridesProject(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "SHARED_KEY", "proj-val")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "staging", "SHARED_KEY", "staging-val")
	project = "test-app"
	envFlag = "staging"

	cmd := newExportCmd()
	stdout, stderr, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}

	if !strings.Contains(stdout, "staging-val") {
		t.Errorf("stdout should contain staging value, got: %s", stdout)
	}
	if strings.Contains(stdout, "proj-val") {
		t.Errorf("stdout should NOT contain project value, got: %s", stdout)
	}
}

func TestExportCmd_EmptyStore(t *testing.T) {
	defer setupTest(t)()

	cmd := newExportCmd()
	stdout, stderr, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if stdout != "" {
		t.Errorf("expected empty stdout, got: %s", stdout)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr, got: %s", stderr)
	}
}

func TestExportCmd_ShellQuoteApplied(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "ITEMS", "a'b'c")

	cmd := newExportCmd()
	stdout, stderr, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}

	// The value with single quotes should be shell-escaped
	if !strings.Contains(stdout, shellQuote("a'b'c")) {
		t.Errorf("stdout should contain shell-escaped value, got: %s", stdout)
	}
}
