package commands

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/dipockdas/keysync/internal/config"
	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

// setupTest saves global state and injects a MemoryStore for testing.
func setupTest(t *testing.T) func() {
	t.Helper()

	origSecretSt := secretSt
	origCfg := cfg
	origProject := project
	origEnv := envFlag
	origConfigPath := configPath
	origRepoFlag := repoFlag

	secretSt = store.NewMemoryStore()
	cfg = &config.Config{
		Projects: map[string]config.ProjectConfig{
			"test-app": {},
		},
	}
	project = ""
	envFlag = "production"
	configPath = "/tmp/test/.keysync.json"
	repoFlag = "test/repo"

	return func() {
		secretSt = origSecretSt
		cfg = origCfg
		project = origProject
		envFlag = origEnv
		configPath = origConfigPath
		repoFlag = origRepoFlag
	}
}

// captureCommand runs a command's RunE and captures stdout/stderr.
// Commands use fmt.Print/fmt.Fprintf(os.Stderr) directly (not cmd.OutOrStdout),
// so we redirect os.Stdout and os.Stderr at the OS level.
func captureCommand(cmd *cobra.Command, args []string) (stdout, stderr string, err error) {
	// Save original descriptors
	origStdout := os.Stdout
	origStderr := os.Stderr

	// Create pipes
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	cmd.SetArgs(args)
	cmd.SetContext(context.Background())
	err = cmd.RunE(cmd, args)

	// Close writers and restore
	wOut.Close()
	wErr.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr

	// Read captured output
	outBuf := &bytes.Buffer{}
	io.Copy(outBuf, rOut)
	errBuf := &bytes.Buffer{}
	io.Copy(errBuf, rErr)

	return outBuf.String(), errBuf.String(), err
}

// ---------------------------------------------------------------------------
// Get command tests
// ---------------------------------------------------------------------------

func TestGetCmd_ProjectScope(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	secretSt.Set(context.Background(), store.ScopeProject, "test-app", "production", "MY_KEY", "my-value")

	cmd := newGetCmd()
	// pflag resets the variable when binding — set after newGetCmd
	getUnmask = true
	defer func() { getUnmask = false }()
	stdout, stderr, err := captureCommand(cmd, []string{"MY_KEY"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if stdout != "MY_KEY=my-value" {
		t.Errorf("stdout = %q, want %q", stdout, "MY_KEY=my-value")
	}
}

func TestGetCmd_GlobalScope(t *testing.T) {
	defer setupTest(t)()
	secretSt.Set(context.Background(), store.ScopeGlobal, "", "", "MY_KEY", "global-val")

	cmd := newGetCmd()
	getUnmask = true
	defer func() { getUnmask = false }()
	stdout, stderr, err := captureCommand(cmd, []string{"MY_KEY"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if stdout != "MY_KEY=global-val" {
		t.Errorf("stdout = %q, want %q", stdout, "MY_KEY=global-val")
	}
}

func TestGetCmd_ProjectFallbackToGlobal(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	// Only set global — no project-scoped value
	secretSt.Set(context.Background(), store.ScopeGlobal, "", "", "MY_KEY", "global-val")

	cmd := newGetCmd()
	getUnmask = true
	defer func() { getUnmask = false }()
	stdout, stderr, err := captureCommand(cmd, []string{"MY_KEY"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if stdout != "MY_KEY=global-val" {
		t.Errorf("stdout = %q, want %q", stdout, "MY_KEY=global-val")
	}
}

func TestGetCmd_EnvFallbackToProject(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	envFlag = "staging"
	// Set project-level (no env) but not staging
	secretSt.Set(context.Background(), store.ScopeProject, "test-app", "", "MY_KEY", "proj-val")

	cmd := newGetCmd()
	getUnmask = true
	defer func() { getUnmask = false }()
	stdout, stderr, err := captureCommand(cmd, []string{"MY_KEY"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	// Should fall back to project-level (no env)
	if stdout != "MY_KEY=proj-val" {
		t.Errorf("stdout = %q, want %q", stdout, "MY_KEY=proj-val")
	}
}

// ---------------------------------------------------------------------------
// Set command tests
// ---------------------------------------------------------------------------

func TestSetCmd_Global(t *testing.T) {
	defer setupTest(t)()

	cmd := newSetCmd()
	stdout, stderr, err := captureCommand(cmd, []string{"MY_KEY=my-value"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "Set global/MY_KEY") {
		t.Errorf("stdout = %q, want 'Set global/MY_KEY'", stdout)
	}
	// GitHub call will fail — expect warning on stderr
	if !strings.Contains(stderr, "warning: failed to write to GitHub") {
		t.Errorf("expected GitHub warning on stderr, got: %s", stderr)
	}

	// Verify value in store
	val, err := secretSt.Get(context.Background(), store.ScopeGlobal, "", "", "MY_KEY")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "my-value" {
		t.Errorf("val = %q, want %q", val, "my-value")
	}
}

func TestSetCmd_ProjectScope(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"

	cmd := newSetCmd()
	stdout, _, err := captureCommand(cmd, []string{"PROJ_KEY=proj-val"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "Set project/test-app/PROJ_KEY") {
		t.Errorf("stdout = %q, want 'Set project/test-app/PROJ_KEY'", stdout)
	}

	val, err := secretSt.Get(context.Background(), store.ScopeProject, "test-app", "production", "PROJ_KEY")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "proj-val" {
		t.Errorf("val = %q, want %q", val, "proj-val")
	}
}

func TestSetCmd_InvalidFormat(t *testing.T) {
	defer setupTest(t)()

	cmd := newSetCmd()
	_, _, err := captureCommand(cmd, []string{"NOEQUALS"})
	if err == nil {
		t.Fatal("expected error for missing '=', got nil")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("error = %q, want 'invalid format'", err.Error())
	}
}

func TestSetCmd_EmptyKey(t *testing.T) {
	defer setupTest(t)()

	cmd := newSetCmd()
	_, _, err := captureCommand(cmd, []string{"=value"})
	if err == nil {
		t.Fatal("expected error for empty key, got nil")
	}
}

// ---------------------------------------------------------------------------
// List command tests
// ---------------------------------------------------------------------------

func TestListCmd_Empty(t *testing.T) {
	defer setupTest(t)()

	cmd := newListCmd()
	stdout, _, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "No secrets found.") {
		t.Errorf("stdout = %q, want 'No secrets found.'", stdout)
	}
}

func TestListCmd_WithSecrets(t *testing.T) {
	defer setupTest(t)()
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "GLOBAL_KEY", "gv")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "production", "PROJ_KEY", "pv")

	cmd := newListCmd()
	stdout, _, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "GLOBAL_KEY") {
		t.Errorf("stdout missing GLOBAL_KEY: %s", stdout)
	}
	if !strings.Contains(stdout, "global") {
		t.Errorf("stdout missing 'global' scope: %s", stdout)
	}
}

func TestListCmd_ProjectFilter(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "G_KEY", "gv")
	secretSt.Set(ctx, store.ScopeProject, "test-app", "production", "P_KEY", "pv")
	secretSt.Set(ctx, store.ScopeProject, "other-app", "production", "O_KEY", "ov")

	cmd := newListCmd()
	stdout, _, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "G_KEY") {
		t.Errorf("stdout missing G_KEY: %s", stdout)
	}
	if !strings.Contains(stdout, "P_KEY") {
		t.Errorf("stdout missing P_KEY: %s", stdout)
	}
	if strings.Contains(stdout, "O_KEY") {
		t.Errorf("stdout should not contain O_KEY (different project): %s", stdout)
	}
}

// ---------------------------------------------------------------------------
// Doctor command tests
// ---------------------------------------------------------------------------

func TestDoctorCmd_ValidConfig(t *testing.T) {
	defer setupTest(t)()

	cmd := newDoctorCmd()
	stdout, _, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "Config") {
		t.Errorf("stdout missing Config check: %s", stdout)
	}
	if !strings.Contains(stdout, "Store") {
		t.Errorf("stdout missing Store check: %s", stdout)
	}
	if !strings.Contains(stdout, "✓") {
		t.Errorf("stdout missing checkmarks: %s", stdout)
	}
}

func TestDoctorCmd_NilConfig(t *testing.T) {
	defer setupTest(t)()
	cfg = nil

	cmd := newDoctorCmd()
	stdout, _, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "✗ Config") {
		t.Errorf("stdout should show config error: %s", stdout)
	}
}

// ---------------------------------------------------------------------------
// Rotate command tests
// ---------------------------------------------------------------------------

func TestRotateCmd(t *testing.T) {
	defer setupTest(t)()
	secretSt.Set(context.Background(), store.ScopeGlobal, "", "", "ROTATE_KEY", "old-value")

	cmd := newRotateCmd()
	stdout, stderr, err := captureCommand(cmd, []string{"ROTATE_KEY"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "Rotation complete") {
		t.Errorf("stdout missing completion message: %s", stdout)
	}
	// GitHub call will fail
	if !strings.Contains(stderr, "✗ github") {
		t.Errorf("expected GitHub error on stderr, got: %s", stderr)
	}

	// Verify value changed
	val, err := secretSt.Get(context.Background(), store.ScopeGlobal, "", "", "ROTATE_KEY")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val == "old-value" {
		t.Error("value was not rotated")
	}
	if len(val) == 0 {
		t.Error("rotated value is empty")
	}
}

// ---------------------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------------------

func TestScopeLabel(t *testing.T) {
	tests := []struct {
		name    string
		scope   store.Scope
		project string
		want    string
	}{
		{"global", store.ScopeGlobal, "", "global"},
		{"global with project", store.ScopeGlobal, "my-app", "global"},
		{"project", store.ScopeProject, "my-app", "project/my-app"},
		{"project empty", store.ScopeProject, "", "global"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scopeLabel(tt.scope, tt.project)
			if got != tt.want {
				t.Errorf("scopeLabel(%q, %q) = %q, want %q", tt.scope, tt.project, got, tt.want)
			}
		})
	}
}

func TestParseEnvFile(t *testing.T) {
	// Write test env file
	content := `# Comment
EMPTY=
  SIMPLE=value
export EXPORTED=exported-val
QUOTED="quoted val"
SINGLE_QUOTED='single quoted'
KEY_WITH_EQUALS==equalsval`

	tmpFile, err := os.CreateTemp("", "test-env-*.env")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	tmpFile.Close()

	secrets, err := parseEnvFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("parseEnvFile failed: %v", err)
	}

	expected := map[string]string{
		"EMPTY":            "",
		"SIMPLE":           "value",
		"EXPORTED":         "exported-val",
		"QUOTED":           "quoted val",
		"SINGLE_QUOTED":    "single quoted",
		"KEY_WITH_EQUALS":  "=equalsval",
	}

	if len(secrets) != len(expected) {
		t.Errorf("got %d secrets, want %d", len(secrets), len(expected))
	}

	for _, s := range secrets {
		wantVal, ok := expected[s.key]
		if !ok {
			t.Errorf("unexpected key: %s", s.key)
			continue
		}
		if s.value != wantVal {
			t.Errorf("key %s: value = %q, want %q", s.key, s.value, wantVal)
		}
	}
}

func TestGenerateSecret(t *testing.T) {
	s1, err := generateSecret()
	if err != nil {
		t.Fatalf("generateSecret failed: %v", err)
	}
	s2, err := generateSecret()
	if err != nil {
		t.Fatalf("generateSecret failed: %v", err)
	}
	if len(s1) == 0 {
		t.Error("generated secret is empty")
	}
	if s1 == s2 {
		t.Error("two generated secrets should not be equal")
	}
}
