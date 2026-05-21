# First-Party Platform Configurations

This directory contains canonical configuration examples for popular deployment
platforms. These platforms are supported through keysync's declarative generic
engine, not hardcoded implementations.

**Two approaches**: Each platform can be configured using either:
1. **HTTP API** - Direct API calls (requires API token, more control)
2. **CLI** - Shell out to platform CLI (simpler config, requires CLI installed)

The `.json` files in this directory use the **HTTP API** approach. See examples below for **CLI alternatives**.

## Usage

Copy the relevant config into your `.keysync.json` and replace placeholder values
with your actual project IDs and settings.

**Example** (Vercel):

```json
{
  "repos": {
    "myorg/myrepo": {
      "project": "my-app",
      "platforms": {
        "vercel": {
          "type": "http",
          "endpoint": "https://api.vercel.com/v9/projects/{PROJECT_ID}/env",
          "method": "POST",
          "token_env": "VERCEL_TOKEN",
          "headers": {
            "Authorization": "Bearer {TOKEN}",
            "Content-Type": "application/json"
          },
          "body": {
            "key": "{KEY}",
            "value": "{VALUE}",
            "target": ["production"],
            "type": "encrypted"
          },
          "template_vars": {
            "PROJECT_ID": "prj_abc123xyz"
          }
        }
      }
    }
  }
}
```

## Available Configs

- **vercel.json** - Vercel Environment Variables API (HTTP)
- **railway.json** - Railway GraphQL API (HTTP)
- **supabase.json** - Supabase Secrets Management API (HTTP)

## CLI Alternatives (Simpler)

If you have the platform CLI installed and authenticated, these are much simpler:

### Vercel CLI
```json
"vercel": {
  "type": "cli",
  "command": "vercel env add {KEY} production preview --yes",
  "stdin": "{VALUE}",
  "token_env": "VERCEL_TOKEN"
}
```

Requires: `vercel` CLI installed and authenticated (`vercel login`)

### Railway CLI
```json
"railway": {
  "type": "cli",
  "command": "railway variables set {KEY}={VALUE}",
  "token_env": "RAILWAY_TOKEN"
}
```

Requires: `railway` CLI installed and authenticated

### Supabase CLI
```json
"supabase": {
  "type": "cli",
  "command": "supabase secrets set {KEY}={VALUE} --project-ref {REF}",
  "token_env": "SUPABASE_TOKEN",
  "template_vars": {
    "REF": "YOUR_PROJECT_REF"
  }
}
```

Requires: `supabase` CLI installed and authenticated

## Template Variables

All configs support these built-in variables:
- `{KEY}` - Secret key name
- `{VALUE}` - Secret value
- `{TOKEN}` - Platform API token (from keychain or environment)

Platform-specific variables (in `template_vars`):
- Vercel: `PROJECT_ID`, `TARGETS`
- Railway: `SERVICE_ID`, `ENVIRONMENT`
- Supabase: `REF`

## Backward Compatibility

Keysync 1.0+ still supports legacy config formats (without `"type": "http"`).
These will be routed to the old built-in implementations, which are deprecated
and will be removed in a future version.
