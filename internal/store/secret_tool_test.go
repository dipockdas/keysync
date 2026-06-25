package store

import "testing"

func TestParseSecretToolSearchOutput(t *testing.T) {
	sample := `
attribute.service = keysync/global
attribute.account = GLOBAL_KEY

attribute.service = keysync/project/my-app
attribute.account = DATABASE_URL

attribute.service = keysync/project/my-app/env/production
attribute.account = DATABASE_URL

attribute.service = other-app/credentials
attribute.account = UNRELATED

service = keysync/project/geo
account = RAILWAY_TOKEN
`

	got := parseSecretToolSearchOutput(sample)
	want := []SecretEntry{
		{Scope: ScopeGlobal, Key: "GLOBAL_KEY"},
		{Scope: ScopeProject, Project: "my-app", Key: "DATABASE_URL"},
		{Scope: ScopeProject, Project: "my-app", Environment: "production", Key: "DATABASE_URL"},
		{Scope: ScopeProject, Project: "geo", Key: "RAILWAY_TOKEN"},
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("entry[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestParseSecretToolSearchOutputIgnoresNonKeysync(t *testing.T) {
	got := parseSecretToolSearchOutput(`
attribute.service = github.com
attribute.account = token
`)
	if len(got) != 0 {
		t.Fatalf("expected no entries, got %#v", got)
	}
}
