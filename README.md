# keysync

[![keysync on GitHub](https://img.shields.io/badge/GitHub-dipockdas%2Fkeysync-181717?style=for-the-badge&logo=github)](https://github.com/dipockdas/keysync)
[![Releases](https://img.shields.io/badge/Releases-view-1f6feb?style=for-the-badge&logo=github)](https://github.com/dipockdas/keysync/releases)

[![CI](https://github.com/dipockdas/keysync/actions/workflows/ci.yml/badge.svg)](https://github.com/dipockdas/keysync/actions/workflows/ci.yml)
[![Security](https://github.com/dipockdas/keysync/actions/workflows/security.yml/badge.svg)](https://github.com/dipockdas/keysync/actions/workflows/security.yml)
[![CodeQL](https://github.com/dipockdas/keysync/actions/workflows/codeql.yml/badge.svg)](https://github.com/dipockdas/keysync/actions/workflows/codeql.yml)
[![Scorecard](https://github.com/dipockdas/keysync/actions/workflows/scorecard.yml/badge.svg)](https://github.com/dipockdas/keysync/actions/workflows/scorecard.yml)
[![Cross-Platform](https://github.com/dipockdas/keysync/actions/workflows/cross-platform.yml/badge.svg)](https://github.com/dipockdas/keysync/actions/workflows/cross-platform.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

<!-- Dynamic badges (stars, forks, Go Report Card, OpenSSF API score) need a public repo.
     After launch, see docs/github-repository-settings.md#readme-badges-after-going-public -->

**Stop checking secrets into git.** Keysync stores development secrets in your OS keychain (macOS Keychain, Linux libsecret, Windows Credential Manager) and syncs them to GitHub Secrets and deployment platforms with one command.

---

## Why keysync

`.env` files are convenient until they leak — committed by mistake, copied to the wrong machine, or shared in a screenshot. Your OS already has a credential store you trust for SSH keys and browser passwords. Keysync makes that store the source of truth for every development secret, with a single `keysync push` to GitHub and Vercel, Railway, Supabase, Cloudflare, GitLab, and [any platform with a CLI or HTTP API](docs/platform-examples.md).

At runtime, apps read **environment variables** (from `keysync export` locally or from your platform in production). Client libraries check env first, so you are not locked in — migrate to Vault or AWS Secrets Manager later without rewriting application code.

[When is keysync the right fit?](docs/when-to-use.md) · [Architecture](docs/architecture.md) · [Security scanning](docs/SECURITY-SCANNING.md)

---

## Quick start

No compiler required — download a release, store secrets in your OS keychain, and sync when you are ready.

**Optional:** [`gh` CLI](https://cli.github.com) for `keysync push` to GitHub Secrets (install from [cli.github.com](https://cli.github.com)).

### Install from the releases page

1. Open **[Releases](https://github.com/dipockdas/keysync/releases)** and download the archive for your system:
   - **macOS (Apple Silicon):** `keysync_darwin_arm64.zip`
   - **macOS (Intel):** `keysync_darwin_amd64.zip`
   - **Linux:** `keysync_linux_amd64.tar.gz` or `keysync_linux_arm64.tar.gz`
   - **Windows:** `keysync_windows_amd64.zip` or `keysync_windows_arm64.zip`
2. Extract the archive and put `keysync` (or `keysync.exe` on Windows) on your `PATH`.
3. Verify:

   ```bash
   keysync version
   keysync doctor
   ```

4. **macOS only** (fewer keychain password prompts):

   ```bash
   keysync trust
   ```

5. In your project directory:

   ```bash
   keysync init --project my-app
   keysync set API_KEY=your_value_here
   keysync set -p my-app DATABASE_URL=postgres://localhost:5432/myapp
   keysync push -p my-app --dry-run    # preview what would sync
   keysync push -p my-app              # → GitHub Secrets + platforms (needs gh)
   eval $(keysync export API_KEY)      # one secret into your shell
   ```

Edit `.keysync.json` with your repo name and platform IDs — see [configuration](docs/configuration.md). Platform tokens (`VERCEL_TOKEN`, etc.) are stored with `keysync set`, never in the config file.

More install options (Homebrew, Go): [docs/install.md](docs/install.md).

### Build from source (developers)

**Prerequisites:** [Go 1.25+](https://go.dev/dl/), [`gh` CLI](https://cli.github.com) for push.

```bash
git clone https://github.com/dipockdas/keysync.git
cd keysync && make build-signed   # macOS: signed binary; use make build on Linux/Windows
export PATH="$PWD/bin:$PATH"
keysync trust                     # macOS: once after install
```

Then use the same `keysync init` / `set` / `push` / `export` commands as above.

---

## Demo

Terminal recording coming soon — see [docs/demo.md](docs/demo.md) for `asciinema` instructions. Suggested flow: `init` → `set` → `list` → `push` → `export`.

---

## How it works

<p align="center">
  <img src="docs/assets/how-it-works.png" alt="keysync flow: you set secrets in the OS keychain, push syncs to GitHub and platforms, export loads environment variables for your app" width="920"/>
</p>

Three scope levels: **global** → **project** → **environment** (higher wins on conflict). Move between scopes with `keysync mv`. Details: [architecture](docs/architecture.md).

---

## Features

- OS-native secret storage (macOS, Linux, Windows)
- Global, project, and environment scopes with `keysync mv` between them
- `keysync push` to GitHub Secrets + declarative platform configs (CLI or HTTP)
- Import from `.env` or cloud CLIs (`keysync migrate`)
- Secret rotation, export, shell completion, `keysync doctor`
- Client libraries: Go, Python, TypeScript, Dart, Swift, Java, C#, Rust, C++, Ruby

---

## Commands

| Command | Description |
|---------|-------------|
| `keysync init` | Create `.keysync.json` |
| `keysync set KEY=value` | Store in keychain (local only) |
| `keysync get KEY` | Copy to clipboard (`-u` to print) |
| `keysync list` | List secrets by scope |
| `keysync mv KEY` | Move between scopes (`--to-p`, `--to-g`, …) |
| `keysync export [KEY]` | Print `export KEY=VALUE` lines (`KEY` = one secret; omit for all) |
| `keysync trust` | macOS: grant this binary keychain access (after install/rebuild) |
| `keysync push` | Sync keychain → GitHub + platforms (`--dry-run`, `--only`, `exclude` in config) |
| `keysync pull` | Reconcile names with GitHub |
| `keysync rotate KEY` | New random value + push |
| `keysync migrate` | Import from `.env` or cloud |
| `keysync doctor` | Diagnostics |

Global flags: `--project` / `-p`, `--env` / `-e`, `--config`, `--store fallback`. Full reference: `keysync --help`.

---

## Installation

| Method | Best for |
|--------|----------|
| **[Releases](https://github.com/dipockdas/keysync/releases)** | Everyone — pre-built binaries (recommended) |
| **Homebrew** | `brew tap dipockdas/keysync https://github.com/dipockdas/keysync && brew install keysync` |
| **Go install** | `go install github.com/dipockdas/keysync/cmd/keysync@latest` |
| **From source** | `git clone … && make build` → `./bin/keysync` |

Full details: [docs/install.md](docs/install.md). Platform keychain notes: [docs/platform-setup.md](docs/platform-setup.md).

---

## Client libraries

Read secrets at runtime — env vars first, keychain fallback. No dependency on the `keysync` binary.

| Language | Path | Windows |
|----------|------|---------|
| Go | [`clients/go/`](clients/go/) | Ready |
| Python | [`clients/python/`](clients/python/) | Ready |
| TypeScript | [`clients/node/`](clients/node/) | Ready |
| Dart/Flutter | [`clients/dart/`](clients/dart/) | Ready |
| Swift | [`clients/swift/`](clients/swift/) | macOS/Linux |
| Java, C#, Rust, C++, Ruby | [`clients/`](clients/) | Ready |

```go
import "github.com/dipockdas/keysync/clients/go"
val, err := keysync.GetSecret("my-app", "DATABASE_URL")
```

Per-language guides: [`clients/README.md`](clients/README.md).

---

## Documentation

| Topic | Guide |
|-------|--------|
| Installation | [docs/install.md](docs/install.md) · [Homebrew](docs/homebrew.md) |
| Configuration & platforms | [docs/configuration.md](docs/configuration.md) |
| Testing & CI | [docs/testing.md](docs/testing.md) |
| Pushing secrets | [docs/pushing-secrets.md](docs/pushing-secrets.md) |
| GitHub settings (open source) | [docs/github-repository-settings.md](docs/github-repository-settings.md) |
| Platform setup (macOS/Linux/Windows) | [docs/platform-setup.md](docs/platform-setup.md) |
| Migration from `.env` | [docs/migration-guide.md](docs/migration-guide.md) |
| Troubleshooting | [docs/troubleshooting.md](docs/troubleshooting.md) |
| Platform config examples | [docs/platform-examples.md](docs/platform-examples.md) |
| Shell completion | [docs/shell-completion.md](docs/shell-completion.md) |

### Tutorials

- [Go project](docs/tutorial-go-project.md)
- [Python Flask app](docs/tutorial-python-flask-app.md)
- [Windows setup](docs/tutorial-windows-setup.md)
- [New client library recipe](docs/new-client-library-recipe.md)

---

## Contributing

Contributions are welcome — bugs, docs, client libraries, and platform configs.

1. Read [CONTRIBUTING.md](CONTRIBUTING.md)
2. `make build && make test`
3. Open a pull request

Report security issues via [private vulnerability reporting](https://github.com/dipockdas/keysync/security/advisories/new) — not public issues. See [SECURITY.md](SECURITY.md).

---

## Development

```bash
make build          # → ./bin/keysync
make test           # unit tests
make test-platform  # platform client tests
make build-signed   # macOS codesign (fewer keychain prompts)
```

---

## License

Apache 2.0 — see [LICENSE](LICENSE).

---

## For AI coding assistants

- [AGENTS.md](AGENTS.md) — architecture and client-library conventions
- [CLAUDE.md](CLAUDE.md) — build commands and repo layout
- Per-language `AGENTS.md` under [`clients/`](clients/)
