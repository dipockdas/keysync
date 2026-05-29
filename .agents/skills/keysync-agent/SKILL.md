---
name: keysync-agent
description: >-
  Use keysync safely when helping users store, migrate, export, or push secrets.
  Covers local keychain ops, init/migrate, push dry-run, scope rules, and what
  agents must never log or commit. Use when the user mentions keysync, .keysync.json,
  keychain secrets, or migrating from .env.
---

# keysync agent workflow

Help users manage secrets with [keysync](https://github.com/dipockdas/keysync) without leaking values or pushing to the wrong repository.

**Canonical user guide:** [docs/getting-started.md](../../../docs/getting-started.md)  
**Policy doc:** [docs/coding-assistants.md](../../../docs/coding-assistants.md)

---

## User-initiated only (read first)

**Do not run `keysync set` or `keysync migrate` for the user.** If they ask you to set a secret or import `.env`, explain why that must happen in **their terminal**:

- `keysync set KEY=value` requires the secret in the message or shell — pasting into chat exposes the value (history, logs, retention).
- `keysync migrate` reads local `.env` files; the user should run it locally and may share only the **`---MIGRATION_RESULT_START---`** block (key names + scopes, no values) for code migration help.

**What you should do instead**

1. Give exact commands for the user to copy and run locally.
2. Use placeholders in examples (`your_value_here`, never real credentials).
3. Help with everything that does **not** require secret values in chat (below).

**Assistant-friendly:** edit `.keysync.json` (repos, platform IDs, generic platform configs), `keysync push --dry-run`, `keysync init` (scaffold), `keysync doctor`, `keysync export` wiring in docs/scripts (user runs export), client-library migration, `keysync trust` instructions on macOS.

---

## Safety rules (non-negotiable)

1. **Never ask for, repeat, or run commands containing secret values** — decline `keysync set` / `migrate` in chat; redirect to terminal.
2. **Never output secret values** in chat, logs, commits, or PRs — only key names, scopes, and repo targets.
3. **Never commit** `.env`, `.keysync.json` with real tokens, or files containing secret values.
4. **`keysync set` is local only** — it does not sync to GitHub or platforms (`keysync push` does).
5. **Suggest `keysync push --dry-run`** before the user runs a real `keysync push`; confirm repo in `.keysync.json`.
6. **Never `keysync init` or `keysync push` in the `dipockdas/keysync` repository** — template config only; push is blocked.
7. **Reject placeholder config** — if `.keysync.json` still has `YOUR_ORG/YOUR_REPO`, user must fix it before push.
8. **Real `keysync push`** — user-initiated after they review dry-run (same reasoning as `set`: confirms intent and auth on their machine).

---

## Three-stage flow (teach users in this order)

### Stage 1 — Local (any directory)

No `.keysync.json` required. **User runs `set` / `get` in their terminal** — provide commands only.

```bash
keysync set API_KEY=your_value_here
keysync get API_KEY                    # clipboard; -u to print
keysync set -p my-app DATABASE_URL=postgres://localhost/db --env ""
keysync get DATABASE_URL -p my-app
keysync list
keysync list -p my-app
```

**Scope note:** With `-p`, `set` defaults to **`dev`** environment. Use `--env ""` for project-wide, or `--env production` for prod.

### Stage 2 — Project repo

```bash
cd /path/to/user-app
keysync init --project my-app      # assistant may scaffold; user confirms
keysync migrate                    # USER ONLY — optional; reads .env locally
```

**Assistant:** edit `.keysync.json` (repo slug, platform IDs, generic platforms). **User:** `keysync set VERCEL_TOKEN=...` (and other tokens) in terminal — never in chat, never in JSON.

### Stage 3 — Cloud

Requires [`gh` CLI](https://cli.github.com) authenticated.

```bash
keysync push -p my-app --dry-run    # assistant or user
keysync push -p my-app              # USER after reviewing dry-run
```

Optional: `--platforms vercel,railway` · `--env production` for non-dev pushes.

---

## Command cheat sheet

| Command | Agent notes |
|---------|-------------|
| `set KEY=val` | **User only** — never run or ask for values in chat |
| `migrate` | **User only** — user may paste migration JSON (names/scopes), not `.env` |
| `get KEY` | **User only** if value needed; assistant explains flags only |
| `export [KEY]` | Document for user scripts; user runs `eval $(keysync export …)` locally |
| `list` | OK without `--unmask` (names only) |
| `push --dry-run` | OK — no secret values in output |
| `push` | **User only** after dry-run |
| `init` | OK to scaffold; user confirms project name |
| Edit `.keysync.json` | OK — no secret values in file |
| `doctor` / `trust` | OK |

---

## Working in the keysync source repo

When the workspace **is** `github.com/dipockdas/keysync`:

- Do **not** run `keysync init` or store real secrets for this repo.
- Do **not** test `push` against `dipockdas/keysync`.
- Use `make test` / memory store for code changes.
- Build: `make build` or `make build-signed` + `keysync trust` on macOS.

---

## Migrating application code from `.env`

**User runs** `keysync migrate` locally. If they want code help, they paste only `---MIGRATION_RESULT_START---` … `---MIGRATION_RESULT_END---` (key names + scopes, no values). Do not ask for `.env` contents.

1. Search: `rg "process\.env\.|os\.Getenv|load_dotenv" --glob '*.{ts,tsx,js,jsx,go,py,rb}'`
2. Replace with client library — see [AGENTS.md](../../../AGENTS.md) for per-language patterns.
3. Remove dotenv imports; add `.env*` to `.gitignore`.
4. Tell user `.env` is safe to delete manually (keysync never modifies it).

---

## macOS Keychain pop-ups

If the user sees repeated Keychain dialogs:

```bash
keysync trust
```

After each `make build` or copying a new binary, run `trust` again. Signed builds: `make build-signed`.

---

## Troubleshooting (safe commands)

```bash
keysync doctor
keysync list -p PROJECT --env dev
gh auth status
```

See [docs/troubleshooting.md](../../../docs/troubleshooting.md).

---

## Related skills and docs

- [docs/coding-assistants.md](../../../docs/coding-assistants.md) — full policy for humans and agents
- [keysync-setup](../keysync-setup/SKILL.md) — install, build, prerequisites
- [docs/configuration.md](../../../docs/configuration.md)
- [docs/pushing-secrets.md](../../../docs/pushing-secrets.md)
- [docs/platform-examples.md](../../../docs/platform-examples.md)
