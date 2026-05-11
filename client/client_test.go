package client

import (
	"context"
	"os/exec"
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
// a /bin/sh -c script. Returns a cleanup function.
func setHelperProcess(t *testing.T, script string) func() {
	t.Helper()

	orig := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		// Redirect to shell script
		allArgs := strings.Join(append([]string{name}, arg...), " ")
		return exec.Command("/bin/sh", "-c", script+" # "+allArgs)
	}
	return func() { execCommand = orig }
}
