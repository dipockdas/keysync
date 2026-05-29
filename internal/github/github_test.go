package github

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestHelperProcess is not a real test — it is invoked as a subprocess by tests
// that override execCommand. It inspects its arguments and emulates git/gh CLI
// behavior so that the github package can be tested without real dependencies.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	// Find arguments after the "--" separator inserted by the test.
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "no command")
		os.Exit(2)
	}

	failMode := os.Getenv("GH_TEST_FAIL_MODE")

	switch args[0] {
	case "git":
		if failMode == "git" {
			fmt.Fprintln(os.Stderr, "not a git repository")
			os.Exit(1)
		}
		os.Exit(0)

	case "gh":
		if failMode == "gh" {
			fmt.Fprintln(os.Stderr, "gh not installed")
			os.Exit(1)
		}

		// gh repo view --json nameWithOwner
		if len(args) >= 3 && args[1] == "repo" {
			switch failMode {
			case "empty-repo":
				fmt.Println(`{"nameWithOwner":""}`)
			case "bad-json":
				fmt.Println(`not valid json`)
			default:
				fmt.Println(`{"nameWithOwner":"test/repo"}`)
			}
			return
		}

		// gh secret set <name> --repo <repo>
		if len(args) >= 3 && args[1] == "secret" && args[2] == "set" {
			io.Copy(io.Discard, os.Stdin)
			if failMode == "gh-secret-set" {
				fmt.Fprintln(os.Stderr, "X Failed to set secret")
				os.Exit(1)
			}
			return
		}

		// gh secret list --repo <repo> --json name
		if len(args) >= 3 && args[1] == "secret" && args[2] == "list" {
			if failMode == "gh-secret-list" {
				fmt.Fprintln(os.Stderr, "X Failed to list secrets")
				os.Exit(1)
			}
			if failMode == "gh-bad-json" {
				fmt.Println(`not valid json`)
				return
			}
			if failMode == "empty-json-list" {
				fmt.Println(`[]`)
				return
			}
			fmt.Println(`[{"name":"SECRET_1"},{"name":"SECRET_2"}]`)
			return
		}

		// gh secret delete <name> --repo <repo>
		if len(args) >= 3 && args[1] == "secret" && args[2] == "delete" {
			if failMode == "gh-secret-delete" {
				fmt.Fprintln(os.Stderr, "X Failed to delete secret")
				os.Exit(1)
			}
			return
		}

		// gh variable set <name> --repo <repo>
		if len(args) >= 3 && args[1] == "variable" && args[2] == "set" {
			io.Copy(io.Discard, os.Stdin)
			if failMode == "gh-variable-set" {
				fmt.Fprintln(os.Stderr, "X Failed to set variable")
				os.Exit(1)
			}
			return
		}

		// Unknown gh subcommand — should not happen in tests, but handle it.
		fmt.Fprintf(os.Stderr, "unknown gh subcommand: %v\n", args[1:])
		os.Exit(1)
	}
}

// setHelperProcess overrides execCommand so that all git/gh invocations go
// through TestHelperProcess. Returns a cleanup function.
func setHelperProcess(t *testing.T) func() {
	t.Helper()

	orig := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		return cmd
	}
	return func() { execCommand = orig }
}

// setFailMode overrides execCommand and sets GH_TEST_FAIL_MODE so the helper
// process simulates specific failure scenarios.
func setFailMode(t *testing.T, mode string) func() {
	t.Helper()

	orig := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1", "GH_TEST_FAIL_MODE=" + mode}
		return cmd
	}
	return func() { execCommand = orig }
}

// ---------------------------------------------------------------------------
// NewClient tests
// ---------------------------------------------------------------------------

func TestNewClient_WithRepo(t *testing.T) {
	client, err := NewClient("myorg/my-repo")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if client.Repo() != "myorg/my-repo" {
		t.Errorf("Repo() = %q, want %q", client.Repo(), "myorg/my-repo")
	}
}

func TestNewClient_AutoDetect(t *testing.T) {
	defer setHelperProcess(t)()

	client, err := NewClient("")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if client.Repo() != "test/repo" {
		t.Errorf("Repo() = %q, want %q", client.Repo(), "test/repo")
	}
}

func TestNewClient_NoGitRepo(t *testing.T) {
	defer setFailMode(t, "git")()

	_, err := NewClient("")
	if err == nil {
		t.Fatal("expected error for missing git repo, got nil")
	}
	if !strings.Contains(err.Error(), "not inside a git repository") {
		t.Errorf("error = %q, want 'not inside a git repository'", err.Error())
	}
}

func TestNewClient_GhNotInstalled(t *testing.T) {
	defer setFailMode(t, "gh")()

	_, err := NewClient("")
	if err == nil {
		t.Fatal("expected error for missing gh, got nil")
	}
	if !strings.Contains(err.Error(), "gh installed") {
		t.Errorf("error = %q, want 'gh installed'", err.Error())
	}
}

func TestNewClient_EmptyRepoResponse(t *testing.T) {
	defer setFailMode(t, "empty-repo")()

	_, err := NewClient("")
	if err == nil {
		t.Fatal("expected error for empty repo response, got nil")
	}
	if !strings.Contains(err.Error(), "returned empty") {
		t.Errorf("error = %q, want 'returned empty'", err.Error())
	}
}

func TestNewClient_BadJSONResponse(t *testing.T) {
	defer setFailMode(t, "bad-json")()

	_, err := NewClient("")
	if err == nil {
		t.Fatal("expected error for bad JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("error = %q, want 'failed to parse'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Set tests
// ---------------------------------------------------------------------------

func TestSet_Success(t *testing.T) {
	defer setHelperProcess(t)()
	client := &Client{repo: "test/repo"}

	err := client.Set("MY_KEY", "my-value")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
}

func TestSet_Error(t *testing.T) {
	defer setFailMode(t, "gh-secret-set")()
	client := &Client{repo: "test/repo"}

	err := client.Set("MY_KEY", "val")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "gh secret set") {
		t.Errorf("error = %q, want 'gh secret set'", err.Error())
	}
}

func TestSetVariable_Success(t *testing.T) {
	defer setHelperProcess(t)()
	client := &Client{repo: "test/repo"}

	err := client.SetVariable("AUTH_ISSUER_URL", "https://auth.example.com")
	if err != nil {
		t.Fatalf("SetVariable failed: %v", err)
	}
}

func TestSetVariable_Error(t *testing.T) {
	defer setFailMode(t, "gh-variable-set")()
	client := &Client{repo: "test/repo"}

	err := client.SetVariable("AUTH_ISSUER_URL", "https://auth.example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "gh variable set") {
		t.Errorf("error = %q, want 'gh variable set'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// List tests
// ---------------------------------------------------------------------------

func TestList_Success(t *testing.T) {
	defer setHelperProcess(t)()
	client := &Client{repo: "test/repo"}

	secrets, err := client.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	want := []string{"SECRET_1", "SECRET_2"}
	if len(secrets) != len(want) {
		t.Fatalf("List() = %v, want %v", secrets, want)
	}
	for i, s := range secrets {
		if s != want[i] {
			t.Errorf("secrets[%d] = %q, want %q", i, s, want[i])
		}
	}
}

func TestList_EmptyJSON(t *testing.T) {
	orig := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{
			"GO_WANT_HELPER_PROCESS=1",
			"GH_TEST_FAIL_MODE=empty-json-list",
		}
		return cmd
	}
	defer func() { execCommand = orig }()

	client := &Client{repo: "test/repo"}
	secrets, err := client.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(secrets) != 0 {
		t.Errorf("expected 0 secrets, got %d", len(secrets))
	}
}

func TestList_CommandError(t *testing.T) {
	defer setFailMode(t, "gh-secret-list")()
	client := &Client{repo: "test/repo"}

	_, err := client.List()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "gh secret list") {
		t.Errorf("error = %q, want 'gh secret list'", err.Error())
	}
}

func TestList_InvalidJSON(t *testing.T) {
	defer setFailMode(t, "gh-bad-json")()
	client := &Client{repo: "test/repo"}

	_, err := client.List()
	if err == nil {
		t.Fatal("expected error for bad JSON, got nil")
	}
	if !strings.Contains(err.Error(), "parse gh secret list") {
		t.Errorf("error = %q, want 'parse gh secret list'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Delete tests
// ---------------------------------------------------------------------------

func TestDelete_Success(t *testing.T) {
	defer setHelperProcess(t)()
	client := &Client{repo: "test/repo"}

	err := client.Delete("MY_KEY")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestDelete_Error(t *testing.T) {
	defer setFailMode(t, "gh-secret-delete")()
	client := &Client{repo: "test/repo"}

	err := client.Delete("MY_KEY")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "gh secret delete") {
		t.Errorf("error = %q, want 'gh secret delete'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Get tests
// ---------------------------------------------------------------------------

func TestGet_ReturnsError(t *testing.T) {
	client := &Client{repo: "test/repo"}

	val, err := client.Get("ANY_KEY")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if val != "" {
		t.Errorf("val = %q, want empty", val)
	}
	if !strings.Contains(err.Error(), "does not expose secret values") {
		t.Errorf("error = %q, want 'does not expose secret values'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Repo accessor test
// ---------------------------------------------------------------------------

func TestClient_Repo(t *testing.T) {
	client := &Client{repo: "org/repo"}
	if client.Repo() != "org/repo" {
		t.Errorf("Repo() = %q, want %q", client.Repo(), "org/repo")
	}
}
