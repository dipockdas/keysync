package commands

import (
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
