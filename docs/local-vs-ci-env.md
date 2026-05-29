# Local vs CI environment secrets

Use **project scope** for local development values and **environment scope** for what GitHub Actions should receive.

## Example: different base URLs

```bash
# Local dev (keysync export --project my-app)
keysync set APP_BASE_URL=http://localhost:3000 --project my-app

# CI / production (used when pushing with --env production)
keysync set APP_BASE_URL=https://api.example.com --project my-app --env production
```

## Push to GitHub

**Always dry-run first:**

```bash
keysync push --project my-app --env production --dry-run
```

Then push only the production-scoped merge:

```bash
keysync push --project my-app --env production --platforms github
```

Without `--env production`, project-scoped `APP_BASE_URL` (localhost) would be pushed and break CI.

## Exclude local-only keys

Keys you never want on GitHub (dev PATs, obsolete URLs):

```json
"exclude": ["LOCAL_DEV_TOKEN", "LOCAL_ONLY_PAT"]
```

Or use a tight `"secrets"` allowlist — see [pushing-secrets.md](pushing-secrets.md).
