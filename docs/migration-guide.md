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

## Step 4: Set up CI/CD (optional)

Create a `.keysync.json` in your project root:

```json
{
  "repos": {
    "owner/repo": {
      "project": "my-project",
      "env": "production"
    }
  }
}
```

Then add this to `.github/workflows/ci.yml`:

```yaml
- name: Sync secrets
  if: github.ref == 'refs/heads/main'
  run: keysync sync --repo ${{ github.repository }} --store fallback
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

This syncs your local secrets to GitHub Secrets on every push to main.

## Step 5: Share with your team

Each team member runs:
```bash
keysync sync --repo owner/repo --store fallback
```

This pulls secrets from GitHub into their local keychain.

## Rolling back

If something goes wrong, your original `.env` is untouched during migration. Keysync never modifies or deletes your `.env` file — it only reads it. You can switch back at any time.
