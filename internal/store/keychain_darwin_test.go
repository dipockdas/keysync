//go:build darwin && cgo

package store

import (
	"os"
	"strings"
	"testing"
)

func TestParseTeamIdentifier(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "signed binary",
			in:   "TeamIdentifier=W69M8X9AXT\nIdentifier=keysync\n",
			want: "W69M8X9AXT",
		},
		{name: "unsigned", in: "TeamIdentifier=not set\n", want: ""},
		{name: "missing", in: "Identifier=keysync\n", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseTeamIdentifier(tt.in); got != tt.want {
				t.Errorf("parseTeamIdentifier() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoginKeychainPath_Exists(t *testing.T) {
	if path := loginKeychainPath(); path == "" {
		t.Fatal("login keychain not found")
	}
}

func TestKeychainPartitionList_IncludesTeamIDWhenSigned(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	list := keychainPartitionList(exe)
	if !strings.Contains(list, "apple-tool:") {
		t.Fatalf("partition list = %q, want apple-tool:", list)
	}
	if tid := teamIDFromExecutable(exe); tid != "" && !strings.Contains(list, "teamid:"+tid) {
		t.Fatalf("signed binary: partition list = %q, want teamid:%s", list, tid)
	}
}
