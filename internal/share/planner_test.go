package share

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/dipockdas/keysync/internal/store"
)

func TestBuildPlanSelectsProjectWideSecretsInDeterministicOrder(t *testing.T) {
	ctx := context.Background()
	secretStore := store.NewMemoryStore()
	mustSet(t, secretStore, store.ScopeProject, "example-app", "", "Z_KEY", "synthetic-z")
	mustSet(t, secretStore, store.ScopeProject, "example-app", "", "A_KEY", "synthetic-a")
	mustSet(t, secretStore, store.ScopeProject, "example-app", "production", "ENV_KEY", "synthetic-env")
	mustSet(t, secretStore, store.ScopeProject, "other-app", "", "OTHER_KEY", "synthetic-other")
	mustSet(t, secretStore, store.ScopeGlobal, "", "", "GLOBAL_KEY", "synthetic-global")

	plan, err := BuildPlan(ctx, secretStore, "example-app", "")
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if got := []string{plan.Secrets[0].Name, plan.Secrets[1].Name}; !reflect.DeepEqual(got, []string{"A_KEY", "Z_KEY"}) {
		t.Fatalf("selected keys = %v", got)
	}
	if plan.Secrets[0].Value != "synthetic-a" || plan.Secrets[1].Value != "synthetic-z" {
		t.Fatal("selected values do not match store")
	}

	preview := plan.Preview()
	if preview.Project != "example-app" || preview.Count != 2 || preview.Scope != "project" {
		t.Fatalf("preview = %#v", preview)
	}
	if !reflect.DeepEqual(preview.Keys, []string{"A_KEY", "Z_KEY"}) {
		t.Fatalf("preview keys = %v", preview.Keys)
	}
	if strings.Contains(preview.String(), "synthetic-") {
		t.Fatalf("preview leaks values: %s", preview.String())
	}
}

func TestBuildPlanSingleKeyUsesProjectThenGlobalFallback(t *testing.T) {
	ctx := context.Background()
	secretStore := store.NewMemoryStore()
	mustSet(t, secretStore, store.ScopeGlobal, "", "", "SHARED_KEY", "synthetic-global")
	mustSet(t, secretStore, store.ScopeProject, "example-app", "", "SHARED_KEY", "synthetic-project")
	mustSet(t, secretStore, store.ScopeGlobal, "", "", "GLOBAL_ONLY", "synthetic-global-only")

	projectPlan, err := BuildPlan(ctx, secretStore, "example-app", "SHARED_KEY")
	if err != nil {
		t.Fatal(err)
	}
	if got := projectPlan.Secrets[0]; got.Scope != store.ScopeProject || got.Value != "synthetic-project" {
		t.Fatalf("project secret = %#v", got)
	}

	globalPlan, err := BuildPlan(ctx, secretStore, "example-app", "GLOBAL_ONLY")
	if err != nil {
		t.Fatal(err)
	}
	if got := globalPlan.Secrets[0]; got.Scope != store.ScopeGlobal || got.Value != "synthetic-global-only" {
		t.Fatalf("global fallback = %#v", got)
	}
	if globalPlan.Preview().Scope != "global" {
		t.Fatalf("global preview = %#v", globalPlan.Preview())
	}
}

func TestBuildPlanRejectsMissingInputsWithoutLeakingValues(t *testing.T) {
	ctx := context.Background()
	secretStore := store.NewMemoryStore()
	mustSet(t, secretStore, store.ScopeProject, "example-app", "", "KNOWN_KEY", "synthetic-value-must-not-leak")

	tests := []struct {
		name    string
		project string
		key     string
		want    error
	}{
		{name: "empty project", project: "", want: ErrProjectRequired},
		{name: "empty project store", project: "missing-app", want: ErrNoSecrets},
		{name: "missing key", project: "example-app", key: "MISSING_KEY", want: ErrSecretNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildPlan(ctx, secretStore, tt.project, tt.key)
			if !errors.Is(err, tt.want) {
				t.Fatalf("BuildPlan() error = %v, want %v", err, tt.want)
			}
			if strings.Contains(err.Error(), "synthetic-value-must-not-leak") {
				t.Fatalf("error leaks secret value: %v", err)
			}
		})
	}
}

func mustSet(t *testing.T, secretStore store.Store, scope store.Scope, project, environment, key, value string) {
	t.Helper()
	if err := secretStore.Set(context.Background(), scope, project, environment, key, value); err != nil {
		t.Fatal(err)
	}
}
