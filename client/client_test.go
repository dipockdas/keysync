package client

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestGetSecret_NoProject(t *testing.T) {
	defer setHelperProcess(t, `echo 'MY_KEY=my-value'`)()

	val, err := GetSecret("", "MY_KEY")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if val != "my-value" {
		t.Errorf("val = %q, want %q", val, "my-value")
	}
}

func TestGetSecret_WithProject(t *testing.T) {
	defer setHelperProcess(t, `echo 'PROJ_KEY=proj-val'`)()

	val, err := GetSecret("my-app", "PROJ_KEY")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if val != "proj-val" {
		t.Errorf("val = %q, want %q", val, "proj-val")
	}
}

func TestGetSecret_NotFound(t *testing.T) {
	defer setHelperProcess(t, `echo ''`)()

	_, err := GetSecret("", "NONEXISTENT")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetSecret_CommandError(t *testing.T) {
	defer setHelperProcess(t, `echo 'error message' >&2; exit 1`)()

	_, err := GetSecret("", "MY_KEY")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "keysync get") {
		t.Errorf("error = %q, want 'keysync get'", err.Error())
	}
}

func TestGetSecret_MalformedOutput(t *testing.T) {
	defer setHelperProcess(t, `echo 'no-equals-sign'`)()

	val, err := GetSecret("", "KEY")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	// No '=' means the output is returned as-is
	if val != "no-equals-sign" {
		t.Errorf("val = %q, want %q", val, "no-equals-sign")
	}
}

func TestGetSecretContext(t *testing.T) {
	defer setHelperProcess(t, `echo 'MY_KEY=ctx-value'`)()

	val, err := GetSecretContext(context.TODO(), "", "MY_KEY")
	if err != nil {
		t.Fatalf("GetSecretContext failed: %v", err)
	}
	if val != "ctx-value" {
		t.Errorf("val = %q, want %q", val, "ctx-value")
	}
}

// setHelperProcess overrides exec.Command so keysync CLI invocations go through
// a cross-platform shell script. Returns a cleanup function.
func setHelperProcess(t *testing.T, script string) func() {
	t.Helper()

	orig := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		return shellCommand(script)
	}
	return func() { execCommand = orig }
}

// shellCommand creates a cross-platform command from a POSIX sh script.
// Translates echo, stderr redirect, and exit for Windows cmd compatibility.
func shellCommand(script string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		cmd := script
		// cmd echo with no args outputs "ECHO is on." — use echo. for blank line
		cmd = strings.ReplaceAll(cmd, "echo ''", "echo.")
		// Strip single quotes (sh uses them for grouping, cmd treats them literally)
		cmd = strings.ReplaceAll(cmd, "'", "")
		// Translate sh redirect/stderr/command separator syntax
		cmd = strings.ReplaceAll(cmd, ">&2", "1>&2")
		cmd = strings.ReplaceAll(cmd, ";", " & ")
		return exec.Command("cmd", "/C", cmd)
	}
	return exec.Command("/bin/sh", "-c", script)
}
