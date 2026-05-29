# Getting started

After [installing keysync](install.md), work through three stages. Stage 1 needs no project folder; stages 2 and 3 use your app repository.

**macOS:** Run `keysync trust` after installing or building so keychain prompts stay minimal.

---

## 1. Local — keychain only

Try keysync from any directory. No `.keysync.json` required.

```bash
# Global secret (shared across projects)
keysync set API_KEY=your_value_here
keysync get API_KEY              # clipboard by default; -u prints to stdout

# Project-scoped secret (project-wide, not tied to dev/staging/prod)
keysync set -p my-app DATABASE_URL=postgres://localhost:5432/myapp --env ""
keysync get DATABASE_URL -p my-app

# See what is stored
keysync list
keysync list -p my-app
```

**Note on `--env`:** If you use `-p` without `--env`, `keysync set` stores under the `dev` environment. For a single project-wide value, pass `--env ""` as above. For staging or production, use `--env staging` or `--env production`. See [configuration](configuration.md) and [architecture](architecture.md).

Load secrets into your shell when needed:

```bash
eval $(keysync export API_KEY)
eval $(keysync export DATABASE_URL -p my-app --env "")
```

---

## 2. In your project — init and migrate

In your application repository:

```bash
cd ~/code/my-app
keysync init --project my-app
```

This creates `.keysync.json` with placeholders. Edit it with your GitHub repo name and platform IDs — see [configuration](configuration.md).

If you already have a `.env` file:

```bash
keysync migrate
```

Migrate imports keys into the keychain and can suggest scopes. Your `.env` file is never modified. Details: [migration guide](migration-guide.md).

---

## 3. Cloud — push to GitHub and platforms

Requires the [`gh` CLI](https://cli.github.com) authenticated for your repository.

1. Store platform tokens in the keychain (never in `.keysync.json`):

   ```bash
   keysync set VERCEL_TOKEN=...
   keysync set CLOUDFLARE_API_TOKEN=...
   ```

2. Preview, then push:

   ```bash
   keysync push -p my-app --dry-run
   keysync push -p my-app
   ```

By default, `push` syncs keys for the `dev` environment plus project-wide keys. Use `--env production` for CI/production secrets.

Full reference: [pushing secrets](pushing-secrets.md).

---

## Next steps

| Topic | Guide |
|-------|--------|
| Scopes and `.keysync.json` | [configuration.md](configuration.md) |
| Platform examples | [platform-examples.md](platform-examples.md) |
| Runtime in app code | [clients/README.md](../clients/README.md) |
| Problems | [troubleshooting.md](troubleshooting.md) |
