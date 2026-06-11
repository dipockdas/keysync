# Homebrew installation

Install **keysync** from the official tap (formulas live in this repository under `Formula/`).

## Requirements

- [Homebrew](https://brew.sh)
- A published [GitHub release](https://github.com/dipockdas/keysync/releases) (the tap downloads pre-built binaries)
- For `keysync push` / `pull`: [`gh`](https://cli.github.com) (`brew install gh`)

> **Note:** The tap downloads release assets from GitHub. Anyone can install from the public repository.

## Install

```bash
brew tap dipockdas/keysync https://github.com/dipockdas/keysync
brew install keysync
keysync version
```

On macOS after install or upgrade: `keysync trust` once.

### `brew tap` fails with Git LFS / `post-checkout` error

If you see *"This repository is configured for Git LFS but 'git-lfs' was not found"* when tapping:

- **keysync does not use Git LFS** — no `.gitattributes` or LFS files in this repo.
- Your global Git has LFS hooks from `git lfs install` (runs on every `git clone`).
- Homebrew runs `git clone` with a **minimal `PATH`** that does not include `/opt/homebrew/bin`, so `git-lfs` can be installed but still not found during the tap. Prefixing `PATH=...` on the `brew` command does **not** fix this — Homebrew sanitizes the environment.

**Fix — temporarily remove global LFS hooks, tap, then restore** (works for any user):

```bash
brew install git-lfs   # if not already installed
git lfs uninstall
brew untap dipockdas/keysync 2>/dev/null || true
brew tap dipockdas/keysync https://github.com/dipockdas/keysync
brew install keysync
git lfs install        # restore LFS hooks for your other repos
```

**Alternative — tap from a local clone** (maintainers / offline):

```bash
brew tap dipockdas/keysync /path/to/keysync
brew install dipockdas/keysync/keysync
```

### Homebrew binary shadowed by another `keysync`

If `brew install` warns that `keysync` is shadowed (e.g. by `~/.local/bin/keysync` from `make build`), your shell still runs the other copy first. Check with `which keysync`. Use the Homebrew binary explicitly or remove/rename the older one:

```bash
/opt/homebrew/bin/keysync version    # Apple Silicon
keysync trust
```

Verify:

```bash
keysync version
keysync doctor
```

Upgrade after a new release:

```bash
brew update
brew upgrade keysync
```

## Fresh machine demo script (asciinema)

Use this flow on a clean Mac for an **install** recording (separate from a features demo):

```bash
# 1. Homebrew (skip if already installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# 2. GitHub CLI (needed for push/pull)
brew install gh
gh auth login

# 3. keysync
brew tap dipockdas/keysync https://github.com/dipockdas/keysync
brew install keysync

# 4. Smoke test
keysync version
keysync doctor
keysync init --project demo-app
```

Record with:

```bash
asciinema rec keysync-install.cast
```

## Install from a local clone (development)

Homebrew installs formulae from a **tap**, not a bare file path. From a checkout (formula must be committed on the branch you tap):

```bash
cd /path/to/keysync
brew tap dipockdas/keysync "$(pwd)"
brew install dipockdas/keysync/keysync
```

When finished testing:

```bash
brew untap dipockdas/keysync
```

## Maintainers: update formula after a release

After publishing a GitHub release (`v*` tag, non-draft assets):

```bash
./scripts/update-homebrew-formula.sh v1.0.4
git add Formula/keysync.rb
git commit -m "chore: bump Homebrew formula to v1.0.4"
git push
```

The script reads asset `sha256` digests from the GitHub API via `gh`.

## Uninstall

```bash
brew uninstall keysync
brew untap dipockdas/keysync
```
