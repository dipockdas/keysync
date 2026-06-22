# Coding assistants and keysync

AI coding assistants (Cursor, Claude Code, Copilot, etc.) can help you configure keysync, edit `.keysync.json`, migrate application code away from `.env`, and run safe read-only or dry-run commands. They should **not** store secret values on your behalf.

## Why some commands must be user-initiated

Anything that **writes a secret value** needs that value in the conversation or terminal. If you ask an assistant to run `keysync set API_KEY=…`, you typically paste the real key into chat — which is a leak vector (logs, history, training policies, shared threads).

**Rule of thumb:** You type secrets; the assistant helps with everything around them.

| Command / action | Who should run it | Assistant can help with |
|------------------|-------------------|-------------------------|
| `keysync set KEY=value` | **You** (terminal) | Explain syntax and scopes; **`--env` is optional** (omit for project-wide) |
| `keysync migrate` | **You** (terminal) | Interpret migrate output (key names only), update app code |
| `keysync get` / `list --unmask` | **You** | Explain flags; avoid asking assistant to print values |
| `keysync init` | Either | Scaffold project name; you confirm |
| `keysync push --dry-run` | Assistant or you | Preview targets; confirm repo name |
| `keysync push` | **You** (after dry-run) | Remind to dry-run first |
| `keysync share` | **You** (terminal) | Explain syntax and safety; never run, create `.ksx`, or handle passphrases |
| `keysync accept` | **You** (terminal) | Explain syntax; never run, import bundles, or handle passphrases |
| Edit `.keysync.json` | Assistant | Repo slugs, platform IDs, generic platform configs — **no tokens** |
| `keysync export` / client libs | Assistant | Wiring app code; use placeholders in examples |
| `keysync doctor` / `keysync trust` | Either | Diagnostics and macOS trust steps |

## Safe assistant workflows

### 1. Platform and repo configuration

The assistant can author or update `.keysync.json`:

- GitHub repo slug (`yourorg/yourapp`)
- Vercel `projectId`, Railway service IDs, generic `cli` / `http` platform blocks
- `exclude` / `allowlist` for push

Store `VERCEL_TOKEN`, `GH_TOKEN`, etc. yourself:

```bash
keysync set VERCEL_TOKEN=...    # you run this locally, not in chat
```

### 2. Migrate from `.env`

You run:

```bash
keysync migrate
```

Paste only the **`---MIGRATION_RESULT_START---`** block (key names and scopes) into the assistant if you want help updating source code — not the `.env` contents.

### 3. Push to cloud

Assistant may suggest:

```bash
keysync push -p my-app --dry-run
```

You review output, then run `keysync push` yourself.

### 4. Application code

Assistants should replace `process.env.X` / `os.Getenv` with keysync client libraries using **key names and scopes from migrate output**, never by reading secret values.

### 5. Share secrets with teammates

`keysync share` and `keysync accept` are **user-only** — same reasoning as `set`: they require interactive passphrases and move secret values.

**Agents must not:**

- Run `keysync share` or `keysync accept`
- Create, read, or import `.ksx` bundles
- Request, echo, or transform share passphrases or payload values

**Agents may:**

- Explain command syntax and flags (`--file`, `--wormhole`, `-k KEY`)
- Describe the security model (encrypted `.ksx`, separate passphrase channel, 10-minute file expiry, 5-minute Wormhole timeout)
- Help inspect non-secret metadata (project name, key count)

Example commands for **users** to run locally:

```bash
keysync share -p my-app --file
keysync accept ./my-app.keysync.ksx

keysync share -p my-app --wormhole
keysync accept 7-purple-dolphin
```

Confirmation vocabulary: type **`SHARE`** to create/send, **`ACCEPT`** to import. There is no `IMPORT` confirmation step.

See [SECURITY.md](../SECURITY.md#sharing-secrets-with-teammates) and [architecture.md](architecture.md#sharing-secrets).

## Working in the keysync repository

If the open project is [github.com/dipockdas/keysync](https://github.com/dipockdas/keysync):

- Do not `keysync init` or `keysync push` there (example config only; push is blocked).
- Assistants should use `make test` and code changes — not real user secrets.

## Agent skill in this repo

Maintainers ship **[`.agents/skills/keysync-agent/SKILL.md`](../.agents/skills/keysync-agent/SKILL.md)** for tools that load project skills. See also [AGENTS.md](../AGENTS.md) and [CLAUDE.md](../CLAUDE.md).

## Related docs

- [getting-started.md](getting-started.md) — human quick start
- [configuration.md](configuration.md) — `.keysync.json` schema
- [pushing-secrets.md](pushing-secrets.md) — push behavior and dry-run
- [migration-guide.md](migration-guide.md) — `.env` import details
