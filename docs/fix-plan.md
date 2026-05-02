# Secret Leakage Fix Plan

## 1. Process argument leakage — macOS Keychain

**File:** `internal/store/keychain_darwin.go:205-208`

**Problem:** `-w value` passes secret as CLI argument to `security add-generic-password`, visible in process table.

**Fix:** Pipe the value via stdin instead of using `-w value`. The `security` CLI accepts input on stdin, so we remove the `-w value` argument and write the value to `cmd.Stdin`:

```go
cmd := exec.Command("security", "add-generic-password",
    "-s", svc,
    "-a", accountName(key),
    "-U",
)
cmd.Stdin = strings.NewReader(value + "\n")
```

This matches the pattern already used in `libsecret_linux.go:53`.

---

## 2. Process argument leakage — GitHub Secrets

**File:** `internal/github/github.go:59-62`

**Problem:** `--body value` passes secret as CLI argument to `gh secret set`, visible in process table.

**Fix:** The `gh` CLI supports reading the value from stdin — pipe it instead:

```go
cmd := exec.Command("gh", "secret", "set", name,
    "--repo", c.repo,
)
cmd.Stdin = strings.NewReader(value)
```

---

## 3. Unencrypted fallback store

**File:** `internal/store/fallback.go`

**Problem:** Secrets written in plaintext JSON to `~/.config/keysync/store.json`. The crypto package at `internal/crypto/crypto.go` exists but is unused.

**Fix:** Wire `crypto.SealedBox` into `FallbackStore`:

- On first use, generate a random key via `crypto.GenerateRandomKey()` and save it alongside the store file (or derive it deterministically)
- In `save()`, encrypt the JSON blob before writing
- In `load()`, decrypt after reading
- The key file goes at `~/.config/keysync/key` (0600 permissions)

**Note:** This will invalidate existing unencrypted store files. Migration path: detect plaintext on load, re-save encrypted.

---

## 4. API error responses leak secrets

**Files:**
- `internal/platforms/vercel.go:94`
- `internal/platforms/railway.go:96`
- `internal/platforms/supabase.go:84`

**Problem:** Non-200 response bodies are included verbatim in error messages. Some platforms echo request payloads (including secrets) in error responses.

**Fix:** Before including the response body in the error, scan for and mask `"value":"...","` patterns. Add a helper function to the `platforms` package:

```go
func sanitizeResponseBody(body []byte) string {
    s := string(body)
    // Mask "value":"<anything>" patterns in JSON responses
    re := regexp.MustCompile(`"value"\s*:\s*"[^"]+"`)
    s = re.ReplaceAllString(s, `"value":"***MASKED***"`)
    // General key=value patterns in form-encoded/plaintext responses
    re2 := regexp.MustCompile(`(?i)(secret|key|token|password|api_key)=[^\s&"]+`)
    s = re2.ReplaceAllString(s, `$1=***MASKED***`)
    // Truncate to first 512 chars to avoid huge error messages
    if len(s) > 512 {
        s = s[:512] + "..."
    }
    return s
}
```

Apply to all three platform files when constructing error messages.

---

## 5. Migrate interactive prompt shows values

**File:** `internal/commands/migrate.go:203-207`

**Problem:** The interactive prompt prints `KEY=VALUE` (truncated to 77 chars), exposing secrets on screen and in terminal scrollback.

**Fix:** Show only the key name in the prompt:

```go
fmt.Printf("  %s=***\n", kv.key)
```

The user can still see which key they're importing, but the value stays hidden. Offer a `--verbose` flag if they want to inspect values.

---

## 6. test-secrets references disabled inject command

**File:** `internal/commands/test_secrets.go:56-57`

**Problem:** Prints: `keysync inject --project X > .env.test` — points to a non-existent command.

**Fix:** Remove or update the reference. Since the whole point is to avoid `.env` files, remove the `.env.test` suggestion entirely and recommend `keysync get KEY` instead:

```go
fmt.Println("\nTo retrieve a test secret:")
fmt.Printf("  keysync get TEST_SECRET_1 --project %s\n", project)
```
