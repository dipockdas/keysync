# Platform Configuration Examples

This document shows how to configure different platforms in `.keysync.json`.

**⚠️ Deprecation Notice**: Built-in hardcoded platforms (Vercel, Railway, Supabase) are deprecated and will be removed in keysync 2.0. Use the generic engine with first-party configs from `docs/platform-configs/` instead.

## Platform Types

Platforms are configured using the **generic engine** with declarative JSON configs:

- **CLI-based**: Execute platform CLIs with template substitution (e.g., Cloudflare Workers via `wrangler`)
- **HTTP-based**: Send HTTP requests to platform APIs (e.g., GitLab, Netlify, Render)

See `docs/platform-configs/` for canonical examples of Vercel, Railway, and Supabase using the generic engine.

## Generic Platform Structure

Generic platforms require a `"type"` field set to either `"cli"` or `"http"`:

```json
{
  "platforms": {
    "platform-name": {
      "type": "cli",        // or "http"
      "token_env": "TOKEN_ENV_VAR_NAME",
      // ... platform-specific fields
    }
  }
}
```

---

## CLI-Based Platforms

### GitHub Secrets (via `gh` CLI)

```json
{
  "github": {
    "type": "cli",
    "command": "gh secret set {KEY} --repo {REPO}",
    "stdin": "{VALUE}",
    "token_env": "GH_TOKEN",
    "template_vars": {
      "REPO": "yourorg/yourrepo"
    },
    "validation": {
      "command_check": "gh --version"
    }
  }
}
```

**Placeholders**:
- `{KEY}` - Secret key name
- `{VALUE}` - Secret value (sent via stdin for security)
- `{REPO}` - From `config.REPO`
- `{GH_TOKEN}` - From environment variable or OS keychain

**Token**: Stored as global secret `GH_TOKEN` or environment variable

---

### Cloudflare Workers (via `wrangler` CLI)

```json
{
  "cloudflare": {
    "type": "cli",
    "command": "wrangler secret put {KEY}",
    "stdin": "{VALUE}",
    "token_env": "CLOUDFLARE_API_TOKEN",
    "validation": {
      "command_check": "wrangler --version"
    }
  }
}
```

**How it works**:
1. Runs `wrangler secret put API_KEY` for each secret
2. Pipes the secret value to stdin (never in command line args)
3. Requires `CLOUDFLARE_API_TOKEN` in keychain or environment

**Install**: `npm install -g wrangler`

---

### Fly.io (via `flyctl` CLI)

```json
{
  "fly": {
    "type": "cli",
    "command": "flyctl secrets set {KEY}={VALUE} --app {APP_NAME}",
    "token_env": "FLY_API_TOKEN",
    "template_vars": {
      "APP_NAME": "my-app"
    }
  }
}
```

**Install**: `curl -L https://fly.io/install.sh | sh`

---

### Netlify (via `netlify` CLI)

```json
{
  "netlify": {
    "type": "cli",
    "command": "netlify env:set {KEY} {VALUE} --context {CONTEXT}",
    "token_env": "NETLIFY_AUTH_TOKEN",
    "template_vars": {
      "CONTEXT": "production"
    }
  }
}
```

**Contexts**: `production`, `deploy-preview`, `branch-deploy`, `dev`

**Install**: `npm install -g netlify-cli`

---

## HTTP API-Based Platforms

### GitLab CI/CD Variables

```json
{
  "gitlab": {
    "type": "http",
    "endpoint": "https://gitlab.com/api/v4/projects/{PROJECT_ID}/variables",
    "method": "POST",
    "headers": {
      "PRIVATE-TOKEN": "{GITLAB_TOKEN}"
    },
    "body": {
      "key": "{KEY}",
      "value": "{VALUE}",
      "protected": false,
      "masked": true,
      "environment_scope": "*"
    },
    "token_env": "GITLAB_TOKEN",
    "template_vars": {
      "PROJECT_ID": "12345678"
    }
  }
}
```

**Find PROJECT_ID**:
- Go to your GitLab project → Settings → General
- Or from URL: `gitlab.com/group/project` → use numeric ID from API

**Token**: Create at GitLab → Preferences → Access Tokens (scope: `api`)

---

### Render

```json
{
  "render": {
    "type": "http",
    "endpoint": "https://api.render.com/v1/services/{SERVICE_ID}/env-vars",
    "method": "PUT",
    "headers": {
      "Authorization": "Bearer {RENDER_API_KEY}",
      "Content-Type": "application/json"
    },
    "body": [
      {
        "key": "{KEY}",
        "value": "{VALUE}"
      }
    ],
    "token_env": "RENDER_API_KEY",
    "template_vars": {
      "SERVICE_ID": "srv-abc123xyz"
    }
  }
}
```

**Find SERVICE_ID**: Render dashboard → Service → Settings → Service ID

**Token**: Render dashboard → Account Settings → API Keys

---

### Heroku

```json
{
  "heroku": {
    "type": "http",
    "endpoint": "https://api.heroku.com/apps/{APP_NAME}/config-vars",
    "method": "PATCH",
    "headers": {
      "Authorization": "Bearer {HEROKU_API_KEY}",
      "Content-Type": "application/json",
      "Accept": "application/vnd.heroku+json; version=3"
    },
    "body": {
      "{KEY}": "{VALUE}"
    },
    "token_env": "HEROKU_API_KEY",
    "template_vars": {
      "APP_NAME": "my-app"
    }
  }
}
```

**Token**: `heroku auth:token`

---

### DigitalOcean App Platform

```json
{
  "digitalocean": {
    "type": "http",
    "endpoint": "https://api.digitalocean.com/v2/apps/{APP_ID}",
    "method": "PUT",
    "headers": {
      "Authorization": "Bearer {DIGITALOCEAN_TOKEN}",
      "Content-Type": "application/json"
    },
    "body": {
      "spec": {
        "envs": [
          {
            "key": "{KEY}",
            "value": "{VALUE}",
            "scope": "RUN_AND_BUILD_TIME"
          }
        ]
      }
    },
    "token_env": "DIGITALOCEAN_TOKEN",
    "template_vars": {
      "APP_ID": "abc-123-def"
    }
  }
}
```

**Token**: DigitalOcean → API → Generate New Token

---

## Hardcoded Platforms (Legacy)

These platforms have built-in Go implementations and don't require a `"type"` field:

### Vercel

```json
{
  "vercel": {
    "projectId": "prj_abc123xyz",
    "target": ["production", "preview", "development"]
  }
}
```

**Token**: `VERCEL_TOKEN` (global secret or env var)

---

### Railway

```json
{
  "railway": {
    "environment": "production",
    "service": "service-id-here"
  }
}
```

**Token**: `RAILWAY_TOKEN` (global secret or env var)

---

### Supabase

```json
{
  "supabase": {
    "ref": "abcdefghijklmnopqrst"
  }
}
```

**Token**: `SUPABASE_TOKEN` (global secret or env var)

**Find ref**: Supabase dashboard → Project Settings → General → Reference ID

---

## Template Placeholders

All platform configs support these placeholders:

| Placeholder | Replaced with | Example |
|-------------|---------------|---------|
| `{KEY}` | Secret key name | `DATABASE_URL` |
| `{VALUE}` | Secret value | `postgres://...` |
| `{TOKEN_ENV_VAR}` | Value from token_env | Token from keychain/env |
| `{CONFIG_KEY}` | Value from config object | `config.PROJECT_ID` → `12345` |

**Security**: Always use stdin for `{VALUE}` in CLI commands to prevent secrets from appearing in process lists.

---

## Full Example Configuration

```json
{
  "repos": {
    "myorg/myapp": {
      "project": "myapp",
      "globals": ["SENTRY_DSN", "STRIPE_KEY"],
      "platforms": {
        "github": {
          "type": "cli",
          "command": "gh secret set {KEY} --repo {REPO}",
          "stdin": "{VALUE}",
          "token_env": "GH_TOKEN",
          "template_vars": {
            "REPO": "myorg/myapp"
          }
        },
        "vercel": {
          "projectId": "prj_production123",
          "target": ["production"]
        },
        "cloudflare": {
          "type": "cli",
          "command": "wrangler secret put {KEY}",
          "stdin": "{VALUE}",
          "token_env": "CLOUDFLARE_API_TOKEN"
        },
        "gitlab": {
          "type": "http",
          "endpoint": "https://gitlab.com/api/v4/projects/{PROJECT_ID}/variables",
          "method": "POST",
          "headers": {
            "PRIVATE-TOKEN": "{GITLAB_TOKEN}"
          },
          "body": {
            "key": "{KEY}",
            "value": "{VALUE}",
            "masked": true
          },
          "token_env": "GITLAB_TOKEN",
          "template_vars": {
            "PROJECT_ID": "12345678"
          }
        }
      }
    }
  }
}
```

---

## AI Assistant Workflow

When a user asks to add a new platform:

1. **Search for the platform's API/CLI docs**
2. **Determine if it's CLI or HTTP-based**
3. **Generate the JSON config** (no Go code needed!)
4. **Add to `.keysync.json`**
5. **Test**: `keysync push --project myapp --platforms platform-name`

**Example prompt**: "Add Cloudflare Workers support to keysync"

**AI generates**:
```json
{
  "cloudflare": {
    "type": "cli",
    "command": "wrangler secret put {KEY}",
    "stdin": "{VALUE}",
    "token_env": "CLOUDFLARE_API_TOKEN",
    "validation": {
      "command_check": "wrangler --version"
    }
  }
}
```

No code changes, no PR, no release cycle. Just add the config and push secrets.
