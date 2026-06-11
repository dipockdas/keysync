//go:build integration

package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}
	mod := strings.TrimSpace(string(out))
	if mod == "" || mod == "/dev/null" {
		t.Fatal("GOMOD not set")
	}
	return filepath.Dir(mod)
}

func buildKeysyncBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "keysync")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build",
		"-ldflags", "-X github.com/dipockdas/keysync/internal/commands.Version=integration-test",
		"-o", bin, "./cmd/keysync")
	cmd.Dir = moduleRoot(t)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build: %v\n%s", err, stderr.String())
	}
	return bin
}

func runKeysyncBinary(t *testing.T, bin, home, workDir string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	fullArgs := append([]string{"--store", "fallback"}, args...)
	cmd := exec.Command(bin, fullArgs...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "HOME="+home)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

func TestBinaryUserPath_SetGetProjectFlag(t *testing.T) {
	bin := buildKeysyncBinary(t)
	home := t.TempDir()
	workDir := t.TempDir()

	cfg := `{"repos":{"test-org/test-repo":{"project":"test-app"}}}`
	if err := os.WriteFile(filepath.Join(workDir, ".keysync.json"), []byte(cfg), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	stdout, stderr, err := runKeysyncBinary(t, bin, home, workDir,
		"set", "LANGCHAIN_API_KEY=lsv2_bin", "-p", "geo")
	if err != nil {
		t.Fatalf("set: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "Set project/geo/LANGCHAIN_API_KEY") {
		t.Fatalf("set stdout = %q", stdout)
	}

	stdout, stderr, err = runKeysyncBinary(t, bin, home, workDir,
		"get", "LANGCHAIN_API_KEY", "-p", "geo", "-u")
	if err != nil {
		t.Fatalf("get: %v\nstderr: %s", err, stderr)
	}
	if stdout != "LANGCHAIN_API_KEY=lsv2_bin" {
		t.Fatalf("get stdout = %q", stdout)
	}
}

func TestBinaryUserPath_ListProjectFlag(t *testing.T) {
	bin := buildKeysyncBinary(t)
	home := t.TempDir()
	workDir := t.TempDir()

	cfg := `{"repos":{"test-org/test-repo":{"project":"test-app"}}}`
	if err := os.WriteFile(filepath.Join(workDir, ".keysync.json"), []byte(cfg), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, stderr, err := runKeysyncBinary(t, bin, home, workDir, "set", "H_KEY=1", "-p", "hyperdx"); err != nil {
		t.Fatalf("seed: %v\nstderr: %s", err, stderr)
	}
	if _, stderr, err := runKeysyncBinary(t, bin, home, workDir, "set", "O_KEY=1", "-p", "other"); err != nil {
		t.Fatalf("seed: %v\nstderr: %s", err, stderr)
	}

	stdout, stderr, err := runKeysyncBinary(t, bin, home, workDir, "list", "--project", "hyperdx")
	if err != nil {
		t.Fatalf("list: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "H_KEY") {
		t.Fatalf("list stdout = %q, want H_KEY", stdout)
	}
	if strings.Contains(stdout, "O_KEY") {
		t.Fatalf("list stdout should not contain O_KEY: %s", stdout)
	}
}
