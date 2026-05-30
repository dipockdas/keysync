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
