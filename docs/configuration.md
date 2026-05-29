# Configuration

The `.keysync.json` file maps GitHub repositories to project names, optional global secret allowlists, and deployment platform configs. Keysync searches the current directory and parent directories for this file.

## Minimal example

```json
{
  "repos": {
    "myorg/my-app": {
      "project": "my-app",
      "globals": ["STRIPE_KEY"],
      "platforms": {
        "vercel": { "projectId": "prj_xxxxx", "target": ["production", "preview"] },
        "railway": { "environment": "production", "service": "abc123" },
        "supabase": { "ref": "abcdefghijklmnopqrst" }
      }
    }
  }
}
```

## Repo fields

| Field | Required | Description |
|-------|----------|-------------|
| `project` | Yes | Name used for keychain scoping (`-p` / `--project`) |
| `globals` | No | Global keys to include when pushing this repo |
| `secrets` | No | Allowlist of key names to push (when set, only these keys are uploaded) |
| `exclude` | No | Keys that must never be pushed (local PATs, obsolete values) |
| `platforms` | No | Platform sync configurations |

### Push safety example

```json
{
  "project": "my-app",
  "globals": ["OAUTH_CLIENT_ID", "OAUTH_CLIENT_SECRET", "SESSION_SECRET", "AUTH_ISSUER_URL"],
  "exclude": ["LOCAL_DEV_TOKEN", "LOCAL_ONLY_PAT"],
  "secrets": ["OAUTH_CLIENT_ID", "OAUTH_CLIENT_SECRET", "SESSION_SECRET", "AUTH_ISSUER_URL", "APP_BASE_URL"]
}
```

- `globals` — which global keys to merge in  
- `secrets` — optional allowlist (if omitted, all project keys are pushed)  
- `exclude` — never push these, even if present in the keychain  

See [pushing-secrets.md](pushing-secrets.md) and [local-vs-ci-env.md](local-vs-ci-env.md).

## Platform configuration

### Generic engine (recommended)

Add any platform with `"type": "cli"` or `"type": "http"`. First-party examples live in [platform-configs/](platform-configs/README.md). More samples: [platform-examples.md](platform-examples.md).

**CLI example** (Cloudflare Workers):

```json
{
  "cloudflare": {
    "type": "cli",
    "command": "wrangler secret put {KEY}",
    "stdin": "{VALUE}",
    "token_env": "CLOUDFLARE_API_TOKEN"
  }
}
```

**HTTP example** (GitLab CI variables):

```json
{
  "gitlab": {
    "type": "http",
    "endpoint": "https://gitlab.com/api/v4/projects/{PROJECT_ID}/variables",
    "method": "POST",
    "headers": { "PRIVATE-TOKEN": "{GITLAB_TOKEN}" },
    "body": { "key": "{KEY}", "value": "{VALUE}", "masked": true },
    "token_env": "GITLAB_TOKEN",
    "config": { "PROJECT_ID": "12345" }
  }
}
```

**Placeholders**: `{KEY}`, `{VALUE}`, `{TOKEN}`, plus any `{NAME}` from `template_vars` or `config`.

### GitHub (built-in)

GitHub does not use `"type": "cli"`. Configure under `platforms.github`:

```json
"github": {
  "repo": "myorg/my-app",
  "secrets": ["OAUTH_CLIENT_SECRET", "SESSION_SECRET"],
  "variables": ["OAUTH_CLIENT_ID", "AUTH_ISSUER_URL", "APP_BASE_URL"]
}
```

Keys in `variables` are pushed with `gh variable set` (Actions `vars.*`). All other pushed keys use `gh secret set`. Example: [platform-configs/github.json](platform-configs/github.json).

### Legacy built-in fields

Vercel, Railway, and Supabase still accept shorthand fields in `.keysync.json` for backward compatibility. New projects should copy configs from `docs/platform-configs/` instead.

| Platform | Field | Description |
|----------|-------|-------------|
| Vercel | `projectId` | Vercel project ID (`prj_…`) |
| Vercel | `target` | `production`, `preview`, `development` |
| Railway | `environment` | Deployment environment name |
| Railway | `service` | Service ID |
| Supabase | `ref` | Project reference ID |

## Platform tokens

Store API tokens as **global secrets** in the keychain, not in `.keysync.json`:

```bash
keysync set VERCEL_TOKEN=...
keysync set RAILWAY_TOKEN=...
keysync set SUPABASE_TOKEN=...
```

Environment variable fallbacks for CI/CD:

| Variable | Platform |
|----------|----------|
| `VERCEL_TOKEN` | Vercel |
| `RAILWAY_TOKEN` | Railway |
| `SUPABASE_TOKEN` | Supabase Management API |
| `GH_TOKEN` | GitHub (`gh` CLI) |

## CLI flags

| Flag | Description |
|------|-------------|
| `--config` | Path to `.keysync.json` |
| `-p, --project` | Project name from config |
| `-e, --env` | Environment name for env-scoped secrets |
| `--repo` | GitHub `owner/repo` (alternative to `--project` for push) |
| `--store fallback` | Use encrypted file store instead of OS keychain |
