package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCmd_CreatesConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cmd := newInitCmd()
	stdout, stderr, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, ".keysync.json") {
		t.Errorf("stdout = %q, want '.keysync.json'", stdout)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}

	// Verify file was created
	cfgPath := filepath.Join(dir, ".keysync.json")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatal(".keysync.json was not created")
	}

	// Verify it's valid JSON with empty repos
	data, _ := os.ReadFile(cfgPath)
	if !strings.Contains(string(data), `"repos"`) {
		t.Errorf("config missing repos field: %s", string(data))
	}
}

func TestInitCmd_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Create config first
	cfgPath := filepath.Join(dir, ".keysync.json")
	os.WriteFile(cfgPath, []byte("{}"), 0644)

	cmd := newInitCmd()
	_, _, err := captureCommand(cmd, []string{})
	if err == nil {
		t.Fatal("expected error for existing .keysync.json, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q, want 'already exists'", err.Error())
	}
}

func TestInitCmd_WithProjectAndRepo(t *testing.T) {
	defer setupTest(t)()
	project = "my-app"
	repoFlag = "org/my-app"

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	cmd := newInitCmd()
	stdout, stderr, err := captureCommand(cmd, []string{})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, ".keysync.json") {
		t.Errorf("stdout = %q, want '.keysync.json'", stdout)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}

	// Verify config has the repo entry
	cfgPath := filepath.Join(dir, ".keysync.json")
	data, _ := os.ReadFile(cfgPath)
	if !strings.Contains(string(data), "org/my-app") {
		t.Errorf("config missing repo entry: %s", string(data))
	}
	if !strings.Contains(string(data), "my-app") {
		t.Errorf("config missing project name: %s", string(data))
	}
}
