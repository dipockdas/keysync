---
name: keysync-setup
description: >-
  Install and verify keysync (releases, Homebrew, or build from source), OS
  keychain prerequisites, and macOS trust. For day-to-day secret workflows,
  use the keysync-agent skill instead.
---

# keysync setup

Install and verify [keysync](https://github.com/dipockdas/keysync). For **agents helping with keysync**, use **[keysync-agent](../keysync-agent/SKILL.md)** — `set` and `migrate` are **user-initiated only** (no secrets in chat). See [docs/coding-assistants.md](../../../docs/coding-assistants.md).

User guide after install: [docs/getting-started.md](../../../docs/getting-started.md)

---

## Install (pick one)

### Releases (recommended for end users)

1. [GitHub Releases](https://github.com/dipockdas/keysync/releases) — download zip/tar for OS/arch
2. Extract; add `keysync` to `PATH`
3. `keysync version` && `keysync doctor`
4. **macOS:** `keysync trust`

### Homebrew

```bash
brew tap dipockdas/keysync https://github.com/dipockdas/keysync
brew install keysync
keysync trust    # macOS only
```

### Go install

```bash
go install github.com/dipockdas/keysync/cmd/keysync@latest
# Ensure $HOME/go/bin is on PATH, then macOS:
keysync trust
```

### Build from source

```bash
git clone https://github.com/dipockdas/keysync.git
cd keysync
make build-signed    # macOS; use make build on Linux/Windows
export PATH="$PWD/bin:$PATH"
keysync trust       # macOS
```

---

## Prerequisites

| Tool | Required for |
|------|----------------|
| `gh` | `keysync push`, `keysync pull` |
| Platform CLIs | Optional — `keysync migrate --cloud` only |

```bash
gh auth login
gh auth status
```

---

## OS keychain

| OS | Notes |
|----|--------|
| **macOS** | Built-in Keychain; run `keysync trust` to reduce pop-ups |
| **Linux** | `libsecret-tools` (`secret-tool`); D-Bus session required |
| **Windows** | Credential Manager via Win32 API — no extra install |

Headless Linux: `keysync --store fallback` — see [docs/platform-setup.md](../../../docs/platform-setup.md).

---

## Verify

```bash
keysync version
keysync doctor
make test          # from source clone only
```

---

## Next step

Open **[keysync-agent](../keysync-agent/SKILL.md)** and follow the three-stage flow: local → init/migrate → push.

---

## References

- [docs/install.md](../../../docs/install.md)
- [docs/homebrew.md](../../../docs/homebrew.md)
- [CLAUDE.md](../../../CLAUDE.md)
- [AGENTS.md](../../../AGENTS.md)
