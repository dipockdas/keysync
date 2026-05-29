package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// parseSupabaseTable tests
// ---------------------------------------------------------------------------

func TestParseSupabaseTable_Standard(t *testing.T) {
	input := `Name                 │ Value
─────────────────────┼──────────────────────────
MY_KEY               │ my-value
ANOTHER_KEY          │ another-value`
	secrets, err := parseSupabaseTable(input)
	if err != nil {
		t.Fatalf("parseSupabaseTable failed: %v", err)
	}
	if len(secrets) != 2 {
		t.Fatalf("expected 2 secrets, got %d", len(secrets))
	}

	check := map[string]string{
		"MY_KEY":      "my-value",
		"ANOTHER_KEY": "another-value",
	}
	for _, s := range secrets {
		want, ok := check[s.key]
		if !ok {
			t.Errorf("unexpected key: %s", s.key)
			continue
		}
		if s.value != want {
			t.Errorf("%s = %q, want %q", s.key, s.value, want)
		}
	}
}

func TestParseSupabaseTable_Empty(t *testing.T) {
	secrets, err := parseSupabaseTable("")
	if err != nil {
		t.Fatalf("parseSupabaseTable failed: %v", err)
	}
	if len(secrets) != 0 {
		t.Errorf("expected 0 secrets, got %d", len(secrets))
	}
}

func TestParseSupabaseTable_HeaderOnly(t *testing.T) {
	input := `Name  │ Value
──────┼──────`
	secrets, err := parseSupabaseTable(input)
	if err != nil {
		t.Fatalf("parseSupabaseTable failed: %v", err)
	}
	if len(secrets) != 0 {
		t.Errorf("expected 0 secrets, got %d", len(secrets))
	}
}

func TestParseSupabaseTable_MalformedLine(t *testing.T) {
	input := `Name │ Value
─────┼──────
MALFORMED`
	secrets, err := parseSupabaseTable(input)
	if err != nil {
		t.Fatalf("parseSupabaseTable failed: %v", err)
	}
	// Malformed line without "│" should be skipped
	if len(secrets) != 0 {
		t.Errorf("expected 0 secrets, got %d", len(secrets))
	}
}

func TestParseSupabaseTable_EmptyName(t *testing.T) {
	input := `Name │ Value
─────┼──────
     │ some-value`
	secrets, err := parseSupabaseTable(input)
	if err != nil {
		t.Fatalf("parseSupabaseTable failed: %v", err)
	}
	// Empty name should be skipped
	if len(secrets) != 0 {
		t.Errorf("expected 0 secrets, got %d", len(secrets))
	}
}

// ---------------------------------------------------------------------------
// parseGitHubRepo tests
// ---------------------------------------------------------------------------

func TestParseGitHubRepo_SSH(t *testing.T) {
	got := parseGitHubRepo("git@github.com:myorg/my-repo.git")
	want := "myorg/my-repo"
	if got != want {
		t.Errorf("parseGitHubRepo = %q, want %q", got, want)
	}
}

func TestParseGitHubRepo_HTTPS(t *testing.T) {
	got := parseGitHubRepo("https://github.com/myorg/my-repo.git")
	want := "myorg/my-repo"
	if got != want {
		t.Errorf("parseGitHubRepo = %q, want %q", got, want)
	}
}

func TestParseGitHubRepo_HTTPSWithoutSuffix(t *testing.T) {
	got := parseGitHubRepo("https://github.com/myorg/my-repo")
	want := "myorg/my-repo"
	if got != want {
		t.Errorf("parseGitHubRepo = %q, want %q", got, want)
	}
}

func TestParseGitHubRepo_SSHNoGitSuffix(t *testing.T) {
	got := parseGitHubRepo("git@github.com:myorg/my-repo")
	want := "myorg/my-repo"
	if got != want {
		t.Errorf("parseGitHubRepo = %q, want %q", got, want)
	}
}

func TestParseGitHubRepo_AlreadyNormalized(t *testing.T) {
	got := parseGitHubRepo("myorg/my-repo")
	want := "myorg/my-repo"
	if got != want {
		t.Errorf("parseGitHubRepo = %q, want %q", got, want)
	}
}

func TestParseGitHubRepo_HTTP(t *testing.T) {
	got := parseGitHubRepo("http://github.com/org/repo.git")
	want := "org/repo"
	if got != want {
		t.Errorf("parseGitHubRepo = %q, want %q", got, want)
	}
}

func TestParseGitHubRepo_Empty(t *testing.T) {
	got := parseGitHubRepo("")
	if got != "" {
		t.Errorf("parseGitHubRepo = %q, want empty", got)
	}
}

// ---------------------------------------------------------------------------
// isEnvReference tests
// ---------------------------------------------------------------------------

func TestIsEnvReference_ProcessEnv(t *testing.T) {
	if !isEnvReference("const dbUrl = process.env.DATABASE_URL", "DATABASE_URL") {
		t.Error("expected match for process.env.DATABASE_URL")
	}
}

func TestIsEnvReference_OsGetenv(t *testing.T) {
	if !isEnvReference(`dbUrl := os.Getenv("DATABASE_URL")`, "DATABASE_URL") {
		t.Error("expected match for os.Getenv(\"DATABASE_URL\")")
	}
}

func TestIsEnvReference_OsGetenvBacktick(t *testing.T) {
	if !isEnvReference("dbUrl := os.Getenv(`DATABASE_URL`)", "DATABASE_URL") {
		t.Error("expected match for os.Getenv(`DATABASE_URL`)")
	}
}

func TestIsEnvReference_ProcessEnvBracketDoubleQuote(t *testing.T) {
	if !isEnvReference(`const x = process.env["DATABASE_URL"]`, "DATABASE_URL") {
		t.Error("expected match for process.env[\"DATABASE_URL\"]")
	}
}

func TestIsEnvReference_ProcessEnvBracketSingleQuote(t *testing.T) {
	if !isEnvReference("const x = process.env['DATABASE_URL']", "DATABASE_URL") {
		t.Error("expected match for process.env['DATABASE_URL']")
	}
}

func TestIsEnvReference_ENVBracket(t *testing.T) {
	if !isEnvReference(`ENV["DATABASE_URL"]`, "DATABASE_URL") {
		t.Error("expected match for ENV[\"DATABASE_URL\"]")
	}
}

func TestIsEnvReference_ENVFetch(t *testing.T) {
	if !isEnvReference(`ENV.fetch("DATABASE_URL")`, "DATABASE_URL") {
		t.Error("expected match for ENV.fetch(\"DATABASE_URL\")")
	}
}

func TestIsEnvReference_ShellVariable(t *testing.T) {
	if !isEnvReference("echo ${DATABASE_URL}", "DATABASE_URL") {
		t.Error("expected match for ${DATABASE_URL}")
	}
}

func TestIsEnvReference_QuotedString(t *testing.T) {
	if !isEnvReference(`"DATABASE_URL"`, "DATABASE_URL") {
		t.Error("expected match for \"DATABASE_URL\"")
	}
}

func TestIsEnvReference_CaseInsensitive(t *testing.T) {
	if !isEnvReference(`os.environ["database_url"]`, "DATABASE_URL") {
		t.Error("expected case-insensitive match for os.environ[...]")
	}
}

func TestIsEnvReference_CaseInsensitiveProcessEnv(t *testing.T) {
	if !isEnvReference(`process.env.database_url`, "DATABASE_URL") {
		t.Error("expected case-insensitive match for process.env.lowercase")
	}
}

func TestIsEnvReference_CaseInsensitiveOsGetenv(t *testing.T) {
	if !isEnvReference(`os.Getenv("DATABASE_URL")`, "database_url") {
		t.Error("expected case-insensitive match for os.Getenv uppercase vs lowercase key")
	}
}

func TestIsEnvReference_NoMatchForSimilar(t *testing.T) {
	if isEnvReference("const x = DATABASE_URL_OLD", "DATABASE_URL") {
		t.Error("should not match DATABASE_URL_OLD for key DATABASE_URL")
	}
}

func TestIsEnvReference_NoMatchForComment(t *testing.T) {
	if isEnvReference("// DATABASE_URL is the main db url", "DATABASE_URL") {
		t.Error("should not match plain comment mentions")
	}
}

func TestIsEnvReference_EmptyKeyDoesNotPanic(t *testing.T) {
	// Empty key patterns resolve to "" + "" which strings.Contains handles
	// without panicking. In practice the caller never passes empty keys
	// because validateKeyName rejects them.
	isEnvReference("process.env.SOME_KEY", "")
	isEnvReference("", "")
}

func TestIsEnvReference_OsEnviron(t *testing.T) {
	if !isEnvReference(`os.Environ["DATABASE_URL"]`, "DATABASE_URL") {
		t.Error("expected match for os.Environ[\"DATABASE_URL\"]")
	}
}

// ---------------------------------------------------------------------------
// isScannableExt tests
// ---------------------------------------------------------------------------

func TestIsScannableExt_Valid(t *testing.T) {
	exts := []string{".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".rb", ".sh", ".bash",
		".yaml", ".yml", ".json", ".toml", ".cfg", ".conf", ".ini",
		".env", ".env.example", ".env.sample"}
	for _, ext := range exts {
		t.Run(ext, func(t *testing.T) {
			if !isScannableExt(ext) {
				t.Errorf("isScannableExt(%q) = false, want true", ext)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// scopePromptDefaults tests — verify saved scope/project memory
// ---------------------------------------------------------------------------

func TestScopePromptDefaults_NoSavedScope(t *testing.T) {
	d := scopePromptDefaults("", "")
	if d.key != "g" {
		t.Errorf("key = %q, want %q", d.key, "g")
	}
	if d.hint != "global" {
		t.Errorf("hint = %q, want %q", d.hint, "global")
	}
}

func TestScopePromptDefaults_SavedGlobal(t *testing.T) {
	d := scopePromptDefaults("global", "")
	if d.key != "g" {
		t.Errorf("key = %q, want %q", d.key, "g")
	}
	if d.hint != "global" {
		t.Errorf("hint = %q, want %q", d.hint, "global")
	}
}

func TestScopePromptDefaults_SavedProjectNoName(t *testing.T) {
	d := scopePromptDefaults("project", "")
	if d.key != "p" {
		t.Errorf("key = %q, want %q", d.key, "p")
	}
	if d.hint != "project" {
		t.Errorf("hint = %q, want %q", d.hint, "project")
	}
}

func TestScopePromptDefaults_SavedProjectWithName(t *testing.T) {
	d := scopePromptDefaults("project", "myapp")
	if d.key != "p" {
		t.Errorf("key = %q, want %q", d.key, "p")
	}
	if d.hint != "project - myapp" {
		t.Errorf("hint = %q, want %q", d.hint, "project - myapp")
	}
}

func TestScopePromptDefaults_SwitchingBack(t *testing.T) {
	// After project scope, explicitly choose global — call savedScope="global"
	d := scopePromptDefaults("global", "myapp")
	if d.key != "g" {
		t.Errorf("key = %q, want %q", d.key, "g")
	}
	if d.hint != "global" {
		t.Errorf("hint = %q, want %q", d.hint, "global")
	}
}

// ---------------------------------------------------------------------------
// printMigrationInstructions tests — verify the rewritten output
// ---------------------------------------------------------------------------

func TestPrintMigrationInstructions_Output(t *testing.T) {
	keys := []migratedKey{
		{Key: "API_KEY", Scope: "global", Project: ""},
		{Key: "DB_URL", Scope: "project", Project: "myapp"},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	printMigrationInstructions(keys)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Must start with the marker
	if !strings.Contains(output, "---INSTRUCTIONS_START---") {
		t.Error("output missing INSTRUCTIONS_START marker")
	}
	if !strings.Contains(output, "---INSTRUCTIONS_END---") {
		t.Error("output missing INSTRUCTIONS_END marker")
	}

	// Must contain proper TypeScript client import (not execSync)
	if !strings.Contains(output, `import { getSecret } from "@keysync/node"`) {
		t.Error("output missing TypeScript client library import")
	}
	if strings.Contains(output, "execSync") {
		t.Error("output should not contain execSync shell-out examples")
	}

	// Must contain proper Go client import
	if !strings.Contains(output, `import "github.com/dipockdas/keysync/clients/go"`) {
		t.Error("output missing Go client library import")
	}

	// Must contain proper Python client import
	if !strings.Contains(output, `from keysync import get_secret`) {
		t.Error("output missing Python client library import")
	}

	// Must NOT contain Ruby examples
	if strings.Contains(output, "Ruby") || strings.Contains(output, `ENV["`) {
		t.Error("output should not contain Ruby examples")
	}

	// Must NOT contain circular "re-run migrate" advice
	if strings.Contains(output, "Re-run") {
		t.Error("output should not contain 're-run migrate' advice")
	}

	// Must contain the migrated keys
	if !strings.Contains(output, "API_KEY") || !strings.Contains(output, "DB_URL") {
		t.Error("output missing migrated key names")
	}

	// Must contain scope info
	if !strings.Contains(output, "global") || !strings.Contains(output, "project/myapp") {
		t.Error("output missing scope information")
	}
}

func TestPrintMigrationInstructions_Empty(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	printMigrationInstructions(nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should still produce valid marked output
	if !strings.Contains(output, "---INSTRUCTIONS_START---") {
		t.Error("output missing INSTRUCTIONS_START marker")
	}
}

func TestIsScannableExt_Invalid(t *testing.T) {
	exts := []string{".md", ".txt", ".css", ".png", ".jpg", ".svg", ".exe",
		".zip", ".tar", ".gz", "", ".", ".h", ".c", ".cpp", ".rs"}
	for _, ext := range exts {
		t.Run(ext, func(t *testing.T) {
			if isScannableExt(ext) {
				t.Errorf("isScannableExt(%q) = true, want false", ext)
			}
		})
	}
}
