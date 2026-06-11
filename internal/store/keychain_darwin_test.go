//go:build darwin && cgo

package store

import (
	"fmt"
	"os"
	"path/filepath"
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
	exe, err := resolveExecutablePath()
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

func TestFormatTrustProgress(t *testing.T) {
	tests := []struct {
		done, total int
		wantFilled  int
	}{
		{0, 10, 0},
		{5, 10, 12},
		{10, 10, 24},
		{3, 4, 18},
	}
	for _, tt := range tests {
		out := formatTrustProgress(tt.done, tt.total)
		if !strings.Contains(out, fmt.Sprintf("%d/%d trusted", tt.done, tt.total)) {
			t.Errorf("formatTrustProgress(%d,%d) = %q, missing counter", tt.done, tt.total, out)
		}
		filled := strings.Count(out, "█")
		if filled != tt.wantFilled {
			t.Errorf("formatTrustProgress(%d,%d) filled = %d, want %d (%q)", tt.done, tt.total, filled, tt.wantFilled, out)
		}
	}
}

func TestResolveExecutablePath_Absolute(t *testing.T) {
	exe, err := resolveExecutablePath()
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(exe) {
		t.Fatalf("resolveExecutablePath() = %q, want absolute path", exe)
	}
	if _, err := os.Stat(exe); err != nil {
		t.Fatalf("resolved executable missing: %v", err)
	}
}
