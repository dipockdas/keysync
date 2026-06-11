package commands

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// userPathEnv isolates HOME and working directory so root CLI tests use the
// fallback file store instead of the OS keychain (same code path users hit).
type userPathEnv struct {
	home    string
	workDir string
}

func newUserPathEnv(t *testing.T) *userPathEnv {
	t.Helper()

	home := t.TempDir()
	workDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(workDir)

	cfg := `{
  "repos": {
    "test-org/test-repo": {
      "project": "test-app"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(workDir, ".keysync.json"), []byte(cfg), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return &userPathEnv{home: home, workDir: workDir}
}

func (e *userPathEnv) resetStore(t *testing.T) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(e.home, ".config", "keysync")); err != nil {
		t.Fatalf("reset store: %v", err)
	}
}

func resetCLIGlobals() {
	project = ""
	envFlag = ""
	effectiveEnv = ""
	repoFlag = ""
	storeFlag = ""
	cfgFile = ""
	configPath = ""
	cfg = nil
	secretSt = nil
	listGlobal = false
	listUnmask = false
	getUnmask = false
}

// runUserCLI executes keysync through newRootCmd with real Cobra flag parsing.
// Always uses --store fallback and an isolated HOME from userPathEnv.
func runUserCLI(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	resetCLIGlobals()

	fullArgs := append([]string{"--store", "fallback"}, args...)

	origStdout := os.Stdout
	origStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	root := newRootCmd()
	root.SetArgs(fullArgs)
	err = root.Execute()

	wOut.Close()
	wErr.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr

	var outBuf, errBuf bytes.Buffer
	_, _ = io.Copy(&outBuf, rOut)
	_, _ = io.Copy(&errBuf, rErr)
	return outBuf.String(), errBuf.String(), err
}

func TestUserPath_SetProjectFlagVariants(t *testing.T) {
	env := newUserPathEnv(t)

	cases := []struct {
		name string
		args []string
	}{
		{"short_flag", []string{"set", "LANGCHAIN_API_KEY=lsv2_test", "-p", "geo"}},
		{"long_flag_space", []string{"set", "LANGCHAIN_API_KEY=lsv2_test", "--project", "geo"}},
		{"long_flag_equals", []string{"set", "LANGCHAIN_API_KEY=lsv2_test", "--project=geo"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env.resetStore(t)

			stdout, stderr, err := runUserCLI(t, tc.args...)
			if err != nil {
				t.Fatalf("set failed: %v\nstderr: %s", err, stderr)
			}
			if !strings.Contains(stdout, "Set project/geo/LANGCHAIN_API_KEY") {
				t.Fatalf("stdout = %q, want set confirmation", stdout)
			}

			stdout, stderr, err = runUserCLI(t, "get", "LANGCHAIN_API_KEY", "-p", "geo", "-u")
			if err != nil {
				t.Fatalf("get failed: %v\nstderr: %s", err, stderr)
			}
			if stdout != "LANGCHAIN_API_KEY=lsv2_test" {
				t.Fatalf("get stdout = %q, want LANGCHAIN_API_KEY=lsv2_test", stdout)
			}
		})
	}
}

func TestUserPath_SetBareProjectFlagFails(t *testing.T) {
	env := newUserPathEnv(t)
	env.resetStore(t)

	_, stderr, err := runUserCLI(t, "set", "API_KEY=secret", "-p")
	if err == nil {
		t.Fatal("expected error for bare -p on set")
	}
	if !strings.Contains(err.Error(), "project") && !strings.Contains(stderr, "project") {
		t.Fatalf("error = %v stderr = %q, want project name required", err, stderr)
	}
}

func TestUserPath_ListProjectFlagVariants(t *testing.T) {
	env := newUserPathEnv(t)
	env.resetStore(t)

	if _, _, err := runUserCLI(t, "set", "A_KEY=1", "-p", "alpha"); err != nil {
		t.Fatalf("seed alpha: %v", err)
	}
	if _, _, err := runUserCLI(t, "set", "Z_KEY=1", "-p", "zebra"); err != nil {
		t.Fatalf("seed zebra: %v", err)
	}
	if _, _, err := runUserCLI(t, "set", "O_KEY=1", "-p", "other"); err != nil {
		t.Fatalf("seed other: %v", err)
	}

	stdout, _, err := runUserCLI(t, "list", "-p")
	if err != nil {
		t.Fatalf("list -p: %v", err)
	}
	for _, want := range []string{"Projects", "alpha", "zebra", "other"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("list -p stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "A_KEY") {
		t.Errorf("list -p should not show secret keys: %s", stdout)
	}

	for _, args := range [][]string{
		{"list", "-p", "alpha"},
		{"list", "--project", "alpha"},
		{"ls", "-p", "alpha"},
	} {
		name := strings.Join(args, " ")
		t.Run(name, func(t *testing.T) {
			stdout, _, err := runUserCLI(t, args...)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if !strings.Contains(stdout, "A_KEY") {
				t.Errorf("%s stdout missing A_KEY: %s", name, stdout)
			}
			if strings.Contains(stdout, "Z_KEY") || strings.Contains(stdout, "O_KEY") {
				t.Errorf("%s stdout should only show alpha keys: %s", name, stdout)
			}
		})
	}
}

func TestUserPath_ExportProjectFlag(t *testing.T) {
	env := newUserPathEnv(t)
	env.resetStore(t)

	if _, _, err := runUserCLI(t, "set", "EXPORT_ME=hello", "-p", "geo"); err != nil {
		t.Fatalf("set: %v", err)
	}

	stdout, _, err := runUserCLI(t, "export", "EXPORT_ME", "-p", "geo")
	if err != nil {
		t.Fatalf("export one: %v", err)
	}
	if !strings.Contains(stdout, "export EXPORT_ME=") {
		t.Fatalf("export stdout = %q", stdout)
	}

	stdout, _, err = runUserCLI(t, "export", "-p", "geo")
	if err != nil {
		t.Fatalf("export all: %v", err)
	}
	if !strings.Contains(stdout, "export EXPORT_ME=") {
		t.Fatalf("export -p geo stdout = %q", stdout)
	}
}

func TestUserPath_RmProjectFlag(t *testing.T) {
	env := newUserPathEnv(t)
	env.resetStore(t)

	if _, _, err := runUserCLI(t, "set", "DELETE_ME=bye", "-p", "geo"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if _, _, err := runUserCLI(t, "rm", "DELETE_ME", "-p", "geo"); err != nil {
		t.Fatalf("rm: %v", err)
	}

	// Do not call get here — it os.Exit(1) on missing keys. List instead.
	stdout, _, err := runUserCLI(t, "list", "-p", "geo")
	if err != nil {
		t.Fatalf("list after rm: %v", err)
	}
	if strings.Contains(stdout, "DELETE_ME") {
		t.Fatalf("list still shows deleted key: %s", stdout)
	}
}

func TestUserPath_GetProjectFlagSpace(t *testing.T) {
	env := newUserPathEnv(t)
	env.resetStore(t)

	if _, _, err := runUserCLI(t, "set", "SPACE_KEY=value", "--project", "hyperdx"); err != nil {
		t.Fatalf("set: %v", err)
	}

	stdout, _, err := runUserCLI(t, "get", "SPACE_KEY", "--project", "hyperdx", "-u")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stdout != "SPACE_KEY=value" {
		t.Fatalf("stdout = %q", stdout)
	}
}
