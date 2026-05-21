# First-Party Platform Configurations

This directory contains canonical configuration examples for popular deployment
platforms. These platforms are supported through keysync's declarative generic
engine, not hardcoded implementations.

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

- **vercel.json** - Vercel Environment Variables API
- **railway.json** - Railway GraphQL API
- **supabase.json** - Supabase Secrets Management API

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
