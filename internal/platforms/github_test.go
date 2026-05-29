package platforms

import (
	"testing"
)

func TestGitHubKeyTarget(t *testing.T) {
	cfg := `{"repo":"o/r","variables":["A"],"secrets":["B"]}`
	if got := GitHubKeyTarget(cfg, "A"); got != "variable" {
		t.Errorf("A = %q, want variable", got)
	}
	if got := GitHubKeyTarget(cfg, "B"); got != "secret" {
		t.Errorf("B = %q, want secret", got)
	}
	if got := GitHubKeyTarget(cfg, "OTHER"); got != "secret" {
		t.Errorf("OTHER = %q, want secret", got)
	}
}

func TestValidateGitHubConfig_Duplicate(t *testing.T) {
	cfg := `{"repo":"o/r","secrets":["X"],"variables":["X"]}`
	if err := ValidateGitHubConfig(cfg); err == nil {
		t.Fatal("expected error for duplicate key")
	}
}

func TestValidateGitHubConfig_OK(t *testing.T) {
	cfg := `{"repo":"o/r","secrets":["A"],"variables":["B"]}`
	if err := ValidateGitHubConfig(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
