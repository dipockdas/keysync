# Migrating from .env to keysync

This guide walks you through migrating a project from `.env` files to keysync. You can follow these steps yourself or hand them to an AI assistant.

## Prerequisites

```bash
# macOS: install keysync
brew install dipockdas/tap/keysync

# Verify installation
keysync doctor
```

## Step 1: Run the migration wizard

The `keysync migrate` command reads your `.env` file and interactively stores each secret in your OS keychain.

```bash
cd /path/to/your/project
keysync migrate --file .env
```

For each secret, you'll be asked:
1. **Scope** — choose `[g]lobal` for secrets shared across projects (e.g. OPENAI_API_KEY) or `[p]roject` for project-specific ones (e.g. DATABASE_URL)
2. **Project name** — if project-scoped, enter the project name (or use `--project my-app`)
3. **Confirm** — verify you want to store it

Project-scoped imports are stored **project-wide** (no environment). You do not need `--env` for typical `.env` migrations. Use `keysync set -p NAME KEY=value --env production` later only if you split local vs CI values — see [configuration.md](configuration.md#secret-scopes-and---env).

Use `--dry-run` first to preview without storing:
```bash
keysync migrate --file .env --dry-run
```

## Step 2: Follow the cleanup guide

After migration, keysync prints a cleanup guide showing exactly which lines in your source code reference each migrated secret. For each file, replace the `process.env` / `os.Getenv` calls with the appropriate keysync client call:

### Node.js / TypeScript

```typescript
// Before
const apiKey = process.env.OPENAI_API_KEY;

// After
import { getSecret } from '@keysync/node';
const apiKey = await getSecret('OPENAI_API_KEY');
```

### Go

```go
// Before
apiKey := os.Getenv("OPENAI_API_KEY")

// After
import "github.com/dipockdas/keysync/clients/go"
apiKey, err := keysync.GetGlobal("OPENAI_API_KEY")
```

### Python

```python
# Before
import os
api_key = os.environ.get("OPENAI_API_KEY")

# After
from keysync import get_secret
api_key = get_secret("OPENAI_API_KEY")
```

## Step 3: Remove .env from source control

```bash
# Add to .gitignore
echo ".env" >> .gitignore
echo ".env.*" >> .gitignore

# Remove tracked .env
git rm --cached .env
git commit -m "Migrate secrets from .env to keysync"
```

## Step 4: Sync to GitHub and platforms

Create a `.keysync.json` in your project root:

```json
{
  "repos": {
    "owner/repo": {
      "project": "my-project",
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
            "target": ["production", "preview"],
            "type": "encrypted"
          },
          "template_vars": {
            "PROJECT_ID": "prj_xxxxx"
          }
        }
      }
    }
  }
}
```

See [platform-examples.md](platform-examples.md) for Railway, Supabase, and other platforms.

Then push your secrets to GitHub Secrets and deployment platforms:

```bash
keysync push --project my-project
```

This reads secrets from your OS keychain and pushes them to:
- GitHub Secrets (via `gh` CLI)
- Configured platforms (Vercel, Railway, Supabase)

## Step 5: Use secrets in CI

Your GitHub Actions workflows can now access secrets:

```yaml
# .github/workflows/ci.yml
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Deploy
        env:
          DATABASE_URL: ${{ secrets.DATABASE_URL }}
          API_KEY: ${{ secrets.API_KEY }}
        run: ./deploy.sh
```

## Step 6: Share with your team

Each team member:
1. Runs `keysync migrate --file .env` on their machine
2. Or uses `keysync pull` to download secrets from GitHub Secrets
3. Runs `keysync push` when they add/change secrets locally

## Rolling back

If something goes wrong, your original `.env` is untouched during migration. Keysync never modifies or deletes your `.env` file — it only reads it. You can switch back at any time.
