# Architecture

## Overview

```
┌──────────────────────────────────────────────────────────┐
│                  CLI (cobra commands)                     │
│  init │ set │ get │ list │ export │ mv │ push │ pull     │
│  rotate │ migrate │ doctor │ test-secrets                │
└──────────┬───────────────────────────────────────────────┘
           │
     ┌─────┴─────┐
     │   Store   │── OS Keychain (macOS / Linux libsecret / Windows)
     └─────┬─────┘    └── Fallback: NaCl-encrypted file (~/.config/keysync/store.json)
           │
     ┌─────┴──────────┐
     │  GitHub Sync    │── gh secret set (via `keysync push`)
     └─────┬──────────┘
           │
     ┌─────┴──────────────────┐
     │  Platform Sync          │── Declarative CLI/HTTP configs in .keysync.json
     └────────────────────────┘
```

## What goes where

| Layer | Holds | Example |
|-------|-------|---------|
| **`.keysync.json`** | Non-secret project metadata | Platform project IDs, repo mapping |
| **OS keychain** | All secret values | API tokens, database URLs |
| **Environment variables** | Primary runtime path | `eval $(keysync export)` locally; platform injection in cloud/CI |

`.keysync.json` is safe to commit — it contains no secret values.

## Runtime: environment variables first

Client libraries read `process.env` / `os.Getenv` before touching the keychain:

| Environment | How secrets arrive | App reads |
|---|---|---|
| Local dev | `eval $(keysync export --project …)` | Env vars |
| Cloud / CI | GitHub Actions or platform env (from `keysync push`) | Env vars |

The keychain is used by the **CLI** for setup and management, not on every request in a typical production deployment.

## Scope levels

Secrets are organized by scope. Higher precedence wins when the same key exists at multiple levels:

```
Highest:  Environment (project + env name)
              ↓
          Project (project only)
              ↓
Lowest:   Global
```

| Scope | CLI example |
|-------|-------------|
| Global | `keysync set API_KEY=value` |
| Project | `keysync set -p my-app API_KEY=value` |
| Environment | `keysync set -p my-app API_KEY=value --env production` |

**`--env` is optional.** With `-p` and no `--env`, secrets are project-wide (`keysync/project/<name>`). Use `--env` only when you need separate values per environment (local vs CI, staging vs production, etc.). `set`, `list`, and `push` do not pick a default environment name.

`get` and `export` check an environment bucket only when you pass `--env`; otherwise they use project-wide → global fallback.

Move secrets between scopes with `keysync mv` (see `keysync mv --help`).

## Keychain naming

| Scope | Service name | Account |
|-------|--------------|---------|
| Global | `keysync/global` | Secret key |
| Project | `keysync/project/<name>` | Secret key |
| Project + env | `keysync/project/<name>/env/<env>` | Secret key |

### Windows wire format

Windows uses tagged credential targets: `keysync|s=<scope>|p=<project>|e=<environment>|k=<key>` with percent-encoding for special characters. Legacy underscore format is read for backward compatibility; new entries use the v2 tagged format. See `internal/store/wincred_windows.go` for details.

## Sync pipeline

`keysync push` merges secrets (env → project → global precedence), then pushes to:

1. **GitHub Secrets** — via the `gh` CLI
2. **Deployment platforms** — via declarative `"type": "cli"` or `"type": "http"` configs

Push runs **on your machine** from the local keychain. There is no hosted keysync service.

See [pushing-secrets.md](pushing-secrets.md) and [configuration.md](configuration.md).
