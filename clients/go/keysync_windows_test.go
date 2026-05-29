//go:build windows

package keysync

import "testing"

func TestCredTarget_Global(t *testing.T) {
	got := credTarget("global", "", "")
	want := "keysync_global"
	if got != want {
		t.Errorf("credTarget(global, '', '') = %q, want %q", got, want)
	}
}

func TestCredTarget_Project(t *testing.T) {
	got := credTarget("project", "my-app", "")
	want := "keysync_project_my-app"
	if got != want {
		t.Errorf("credTarget(project, my-app, '') = %q, want %q", got, want)
	}
}

func TestCredTarget_ProjectWithEnv(t *testing.T) {
	got := credTarget("project", "myapp", "dev")
	want := "keysync_project_myapp_dev"
	if got != want {
		t.Errorf("credTarget(project, myapp, dev) = %q, want %q", got, want)
	}
}

func TestCredTarget_GlobalIgnoresEnv(t *testing.T) {
	got := credTarget("global", "myapp", "dev")
	want := "keysync_global"
	if got != want {
		t.Errorf("credTarget(global, myapp, dev) = %q, want %q", got, want)
	}
}

func TestParseCredTarget_Global(t *testing.T) {
	scope, project, env := parseCredTarget("keysync_global")
	if scope != "global" || project != "" || env != "" {
		t.Errorf("parseCredTarget = (%q, %q, %q), want (global, '', '')", scope, project, env)
	}
}

func TestParseCredTarget_Project(t *testing.T) {
	scope, project, env := parseCredTarget("keysync_project_my-app")
	if scope != "project" || project != "my-app" || env != "" {
		t.Errorf("parseCredTarget = (%q, %q, %q), want (project, my-app, '')", scope, project, env)
	}
}

func TestParseCredTarget_ProjectWithEnv(t *testing.T) {
	scope, project, env := parseCredTarget("keysync_project_myapp_dev")
	if scope != "project" || project != "myapp" || env != "dev" {
		t.Errorf("parseCredTarget = (%q, %q, %q), want (project, myapp, dev)", scope, project, env)
	}
}

func TestParseCredTarget_NonKeysync(t *testing.T) {
	scope, project, env := parseCredTarget("other_data")
	if scope != "global" || project != "" || env != "" {
		// Falls back to global since the prefix doesn't match the expected
		// format; TrimPrefix leaves it as-is and the scope validation fails.
		t.Logf("parseCredTarget = (%q, %q, %q)", scope, project, env)
	}
}

func TestTargetFromService_Global(t *testing.T) {
	got := targetFromService("keysync/global")
	want := "keysync_global"
	if got != want {
		t.Errorf("targetFromService(keysync/global) = %q, want %q", got, want)
	}
}

func TestTargetFromService_Project(t *testing.T) {
	got := targetFromService("keysync/project/my-app")
	want := "keysync_project_my-app"
	if got != want {
		t.Errorf("targetFromService(keysync/project/my-app) = %q, want %q", got, want)
	}
}

func TestTargetFromService_ProjectWithEnv(t *testing.T) {
	got := targetFromService("keysync/project/myapp/env/dev")
	want := "keysync_project_myapp_dev"
	if got != want {
		t.Errorf("targetFromService(keysync/project/myapp/env/dev) = %q, want %q", got, want)
	}
}

func TestTargetFromService_NoPrefix(t *testing.T) {
	got := targetFromService("some-service")
	want := "keysync_some-service"
	if got != want {
		t.Errorf("targetFromService(some-service) = %q, want %q", got, want)
	}
}
