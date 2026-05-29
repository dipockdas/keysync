package commands

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// validateKeyName tests
// ---------------------------------------------------------------------------

func TestValidateKeyName_Valid(t *testing.T) {
	keys := []string{"MY_KEY", "MY_KEY_123", "A", "ABC_DEF_123", "_PREFIX"}
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			if err := validateKeyName(key); err != nil {
				t.Errorf("validateKeyName(%q) = %v, want nil", key, err)
			}
		})
	}
}

func TestValidateKeyName_Empty(t *testing.T) {
	err := validateKeyName("")
	if err == nil {
		t.Fatal("expected error for empty key, got nil")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("error = %q, want 'cannot be empty'", err.Error())
	}
}

func TestValidateKeyName_TooLong(t *testing.T) {
	key := strings.Repeat("A", 257)
	err := validateKeyName(key)
	if err == nil {
		t.Fatal("expected error for too-long key, got nil")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("error = %q, want 'too long'", err.Error())
	}
}

func TestValidateKeyName_StartsWithDigit(t *testing.T) {
	err := validateKeyName("1INVALID")
	if err == nil {
		t.Fatal("expected error for digit-starting key, got nil")
	}
	if !strings.Contains(err.Error(), "cannot start with a digit") {
		t.Errorf("error = %q, want 'cannot start with a digit'", err.Error())
	}
}

func TestValidateKeyName_SpecialChars(t *testing.T) {
	keys := []string{"KEY-WITH-DASH", "KEY.WITH.DOT", "KEY/WITH/SLASH", "KEY WITH SPACE"}
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			err := validateKeyName(key)
			if err == nil {
				t.Errorf("expected error for %q, got nil", key)
			}
			if !strings.Contains(err.Error(), "invalid character") {
				t.Errorf("error = %q, want 'invalid character'", err.Error())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// generateSecret tests (existing TestGenerateSecret covers crypto, add coverage)
// ---------------------------------------------------------------------------

func TestGenerateSecret_Length(t *testing.T) {
	s, err := generateSecret()
	if err != nil {
		t.Fatalf("generateSecret failed: %v", err)
	}
	// 32 bytes → 43 chars in base64 RawStdEncoding (no padding)
	if len(s) != 43 {
		t.Errorf("len = %d, want 43", len(s))
	}
}

func TestGenerateSecret_Uniqueness(t *testing.T) {
	s1, _ := generateSecret()
	s2, _ := generateSecret()
	if s1 == s2 {
		t.Error("two generated secrets should not be equal")
	}
}

// ---------------------------------------------------------------------------
// generateTestValue tests
// ---------------------------------------------------------------------------

func TestGenerateTestValue_Length(t *testing.T) {
	v, err := generateTestValue()
	if err != nil {
		t.Fatalf("generateTestValue failed: %v", err)
	}
	// 16 bytes → 32 chars in hex
	if len(v) != 32 {
		t.Errorf("len = %d, want 32", len(v))
	}
}

func TestGenerateTestValue_HexEncoding(t *testing.T) {
	v, err := generateTestValue()
	if err != nil {
		t.Fatalf("generateTestValue failed: %v", err)
	}
	for _, r := range v {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			t.Errorf("non-hex character %q in value %q", r, v)
			break
		}
	}
}

func TestGenerateTestValue_Uniqueness(t *testing.T) {
	v1, _ := generateTestValue()
	v2, _ := generateTestValue()
	if v1 == v2 {
		t.Error("two generated test values should not be equal")
	}
}

// ---------------------------------------------------------------------------
// F() color formatting tests
// ---------------------------------------------------------------------------

func TestF_StripsTagsWhenNoColor(t *testing.T) {
	// Save and restore noColor
	orig := noColor
	noColor = true
	defer func() { noColor = orig }()

	input := "{b}bold{/b} {c}command{/c} {g}green{/g} {y}gold{/y} {u}url{/u}"
	got := F(input)
	want := "bold command green gold url"
	if got != want {
		t.Errorf("F() = %q, want %q", got, want)
	}
}

func TestF_NestedTags(t *testing.T) {
	orig := noColor
	noColor = true
	defer func() { noColor = orig }()

	input := "{b}{c}both{/c}{/b}"
	got := F(input)
	if got != "both" {
		t.Errorf("F() = %q, want %q", got, "both")
	}
}

func TestF_NoTags(t *testing.T) {
	orig := noColor
	noColor = true
	defer func() { noColor = orig }()

	input := "plain text with no tags"
	got := F(input)
	if got != input {
		t.Errorf("F() = %q, want %q", got, input)
	}
}

func TestF_WithColors(t *testing.T) {
	origNoColor := noColor
	noColor = false
	defer func() { noColor = origNoColor }()

	// Ensure ANSI codes are set (they're populated by init(), which may
	// have run with NO_COLOR or non-terminal. If they're empty, this test
	// is only meaningful when the terminal supports color.)
	if cBold == "" {
		t.Skip("ANSI codes not populated (no-color terminal)")
	}

	input := "{b}bold{/b}"
	got := F(input)
	want := cBold + "bold" + cReset
	if got != want {
		t.Errorf("F() = %q, want %q", got, want)
	}
}

func TestF_EmptyString(t *testing.T) {
	orig := noColor
	noColor = true
	defer func() { noColor = orig }()

	got := F("")
	if got != "" {
		t.Errorf("F() = %q, want empty", got)
	}
}
