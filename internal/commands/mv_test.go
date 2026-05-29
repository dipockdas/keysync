package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/dipockdas/keysync/internal/store"
)

func TestMvCmd_GlobalToProject(t *testing.T) {
	defer setupTest(t)()
	project = ""
	envFlag = ""
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "MY_KEY", "val")

	cmd := newMvCmd()
	stdout, _, err := captureCommand(cmd, []string{"MY_KEY", "--to-project", "my-app"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "Moved global/MY_KEY → project/my-app/MY_KEY") {
		t.Errorf("stdout = %q", stdout)
	}

	if _, err := secretSt.Get(ctx, store.ScopeGlobal, "", "", "MY_KEY"); err != store.ErrNotFound {
		t.Error("source should be deleted")
	}
	if val, err := secretSt.Get(ctx, store.ScopeProject, "my-app", "", "MY_KEY"); err != nil || val != "val" {
		t.Errorf("destination = %q, err = %v", val, err)
	}
}

func TestMvCmd_ProjectToGlobal(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	envFlag = ""
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "PROJ_KEY", "proj-val")

	cmd := newMvCmd()
	stdout, _, err := captureCommand(cmd, []string{"PROJ_KEY", "--env", "", "--to-global"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "Moved project/test-app/PROJ_KEY → global/PROJ_KEY") {
		t.Errorf("stdout = %q", stdout)
	}

	if _, err := secretSt.Get(ctx, store.ScopeProject, "test-app", "", "PROJ_KEY"); err != store.ErrNotFound {
		t.Error("source should be deleted")
	}
	if val, err := secretSt.Get(ctx, store.ScopeGlobal, "", "", "PROJ_KEY"); err != nil || val != "proj-val" {
		t.Errorf("destination = %q, err = %v", val, err)
	}
}

func TestMvCmd_EnvToEnv_WithAliases(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	envFlag = "staging"
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeProject, "test-app", "staging", "DB_URL", "staging-val")

	cmd := newMvCmd()
	stdout, _, err := captureCommand(cmd, []string{"DB_URL", "--env", "staging", "--to-p", "test-app", "--to-e", "production"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if !strings.Contains(stdout, "Moved project/test-app/staging/DB_URL → project/test-app/production/DB_URL") {
		t.Errorf("stdout = %q", stdout)
	}

	if _, err := secretSt.Get(ctx, store.ScopeProject, "test-app", "staging", "DB_URL"); err != store.ErrNotFound {
		t.Error("source should be deleted")
	}
	if val, err := secretSt.Get(ctx, store.ScopeProject, "test-app", "production", "DB_URL"); err != nil || val != "staging-val" {
		t.Errorf("destination = %q, err = %v", val, err)
	}
}

func TestMvCmd_ToGlobalAlias(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	envFlag = ""
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "K", "v")

	cmd := newMvCmd()
	_, _, err := captureCommand(cmd, []string{"K", "--env", "", "--to-g"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if _, err := secretSt.Get(ctx, store.ScopeGlobal, "", "", "K"); err != nil {
		t.Errorf("expected key at global scope: %v", err)
	}
}

func TestMvCmd_NotFound(t *testing.T) {
	defer setupTest(t)()
	project = ""
	envFlag = ""

	cmd := newMvCmd()
	_, _, err := captureCommand(cmd, []string{"MISSING", "--to-project", "my-app"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestMvCmd_SameScope(t *testing.T) {
	defer setupTest(t)()
	project = "test-app"
	envFlag = ""
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeProject, "test-app", "", "K", "v")

	cmd := newMvCmd()
	_, _, err := captureCommand(cmd, []string{"K", "--env", "", "--to-project", "test-app"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "same") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestMvCmd_DestExists(t *testing.T) {
	defer setupTest(t)()
	project = ""
	envFlag = ""
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "K", "src")
	secretSt.Set(ctx, store.ScopeProject, "my-app", "", "K", "existing")

	cmd := newMvCmd()
	_, _, err := captureCommand(cmd, []string{"K", "--to-project", "my-app"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestMvCmd_ForceOverwrite(t *testing.T) {
	defer setupTest(t)()
	project = ""
	envFlag = ""
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "K", "new-val")
	secretSt.Set(ctx, store.ScopeProject, "my-app", "", "K", "old-val")

	cmd := newMvCmd()
	_, _, err := captureCommand(cmd, []string{"K", "--to-project", "my-app", "--force"})
	if err != nil {
		t.Fatalf("RunE failed: %v", err)
	}
	if val, _ := secretSt.Get(ctx, store.ScopeProject, "my-app", "", "K"); val != "new-val" {
		t.Errorf("destination = %q, want new-val", val)
	}
	if _, err := secretSt.Get(ctx, store.ScopeGlobal, "", "", "K"); err != store.ErrNotFound {
		t.Error("source should be deleted")
	}
}

func TestMvCmd_NoDestination(t *testing.T) {
	defer setupTest(t)()

	cmd := newMvCmd()
	_, _, err := captureCommand(cmd, []string{"K"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "destination required") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestMvCmd_ConflictingToProjectFlags(t *testing.T) {
	defer setupTest(t)()
	project = ""
	envFlag = ""
	ctx := context.Background()
	secretSt.Set(ctx, store.ScopeGlobal, "", "", "K", "v")

	cmd := newMvCmd()
	_, _, err := captureCommand(cmd, []string{"K", "--to-project", "a", "--to-p", "b"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "conflicting") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestResolveMvDestFlags(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantGlob  bool
		wantProj  string
		wantEnv   string
		wantError string
	}{
		{"to-global", []string{"--to-global"}, true, "", "", ""},
		{"to-g alias", []string{"--to-g"}, true, "", "", ""},
		{"to-project", []string{"--to-project", "app"}, false, "app", "", ""},
		{"to-p alias", []string{"--to-p", "app"}, false, "app", "", ""},
		{"to-env", []string{"--to-project", "app", "--to-env", "prod"}, false, "app", "prod", ""},
		{"to-e alias", []string{"--to-p", "app", "--to-e", "prod"}, false, "app", "prod", ""},
		{"missing dest", []string{}, false, "", "", "destination required"},
		{"to-env without project", []string{"--to-e", "prod"}, false, "", "", "requires --to-project"},
		{"both global and project", []string{"--to-g", "--to-p", "app"}, false, "", "", "cannot use"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := newMvCmd()
			sub.SetArgs(append([]string{"KEY"}, tt.args...))
			if err := sub.ParseFlags(append([]string{"KEY"}, tt.args...)); err != nil {
				t.Fatalf("ParseFlags: %v", err)
			}

			gotGlob, gotProj, gotEnv, err := resolveMvDestFlags(sub)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if gotGlob != tt.wantGlob || gotProj != tt.wantProj || gotEnv != tt.wantEnv {
				t.Errorf("got (%v, %q, %q), want (%v, %q, %q)", gotGlob, gotProj, gotEnv, tt.wantGlob, tt.wantProj, tt.wantEnv)
			}
		})
	}
}
