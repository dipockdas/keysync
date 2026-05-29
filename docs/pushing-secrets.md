# Pushing secrets

`keysync push` reads secrets from your OS keychain and pushes them to GitHub Secrets and configured deployment platforms. **Push always runs locally** on your machine — there is no keysync cloud service.

## Which keys are pushed?

| Source | Rule |
|--------|------|
| **Global** | Only keys listed in `"globals"` for that repo |
| **Project** | **All** project-scoped keys, unless restricted |
| **Environment** | All keys for `--env NAME` when set (wins over project for the same key name) |

**Restrict project keys** (recommended for repos with local-only credentials):

```json
{
  "repos": {
    "myorg/my-app": {
      "project": "my-app",
      "globals": ["SHARED_API_KEY"],
      "secrets": ["SHARED_API_KEY", "DATABASE_URL", "OAUTH_CLIENT_SECRET"],
      "exclude": ["LOCAL_DEV_TOKEN", "LOCAL_PAT"]
    }
  }
}
```

| Field | Effect |
|-------|--------|
| `globals` | Which **global** keys to include |
| `secrets` | Optional **allowlist** — when set, only these key names are pushed |
| `exclude` | Keys that are **never** pushed (applied after allowlist) |

CLI overrides: `--only KEY1,KEY2` for a single run. Preview: `--dry-run` (prints key + scope source).

### GitHub Actions: secrets vs variables

Sensitive values belong in **secrets** (`gh secret set`). Non-sensitive config can use **variables** (`gh variable set`) so workflows can reference `vars.*` instead of `secrets.*`.

```json
{
  "platforms": {
    "github": {
      "repo": "myorg/my-app",
      "secrets": ["OAUTH_CLIENT_SECRET", "SESSION_SECRET"],
      "variables": ["OAUTH_CLIENT_ID", "AUTH_ISSUER_URL", "APP_BASE_URL"]
    }
  }
}
```

Keys listed under `variables` are pushed with `gh variable set`; all other pushed keys use `gh secret set`. See [platform-configs/github.json](platform-configs/github.json).

## Basic usage

```bash
keysync push --project my-app --dry-run
keysync push --project my-app
```

This command:

1. Builds a push plan (globals + project + optional env, minus exclude/allowlist)
2. Pushes to GitHub Secrets via `gh`
3. Pushes to each platform in `.keysync.json`

## When to push

- After `keysync set` or `keysync migrate`
- After `keysync rotate`
- After `keysync mv` between scopes
- When onboarding a new environment or platform

## Examples

```bash
# All configured platforms
keysync push -p my-app

# Production environment scope only
keysync push -p my-app --env production

# Subset of platforms
keysync push -p my-app --platforms vercel,railway

# Resolve repo from config by project name
keysync push -p my-app
```

## Platform tokens

Push reads platform API tokens from global keychain secrets (`VERCEL_TOKEN`, `RAILWAY_TOKEN`, etc.) or from environment variables when the keychain is unavailable (typical in CI).

```bash
keysync set VERCEL_TOKEN=...
export VERCEL_TOKEN=...   # CI fallback
```

## Timeouts

HTTP platform requests use a 30-second timeout. CLI-based platforms inherit the command context. Failed pushes return an error without automatic retry.

## Custom platforms

Add declarative configs to `.keysync.json` — no Go code required. See [platform-examples.md](platform-examples.md) and [platform-configs/](platform-configs/README.md).

## Security note

`keysync push` includes validation to prevent pushing to the keysync repository itself when placeholder config values are left unchanged. Always update `YOUR_ORG/YOUR_REPO` in `.keysync.json` before use.
