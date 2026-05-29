# Installation

## Homebrew (macOS and Linux)

```bash
brew tap dipockdas/keysync https://github.com/dipockdas/keysync
brew install keysync
```

See [homebrew.md](homebrew.md) for upgrade, uninstall, demo script, and maintainer bump instructions.

## GitHub Releases

Pre-built binaries are published when a version tag is pushed (`v*`).

1. Open [Releases](https://github.com/dipockdas/keysync/releases)
2. Download the archive for your OS/architecture:
   - `keysync_darwin_arm64.zip` — Apple Silicon Mac
   - `keysync_darwin_amd64.zip` — Intel Mac
   - `keysync_linux_amd64.tar.gz` / `keysync_linux_arm64.tar.gz`
   - `keysync_windows_amd64.zip` / `keysync_windows_arm64.zip`
3. Extract and move `keysync` (or `keysync.exe`) onto your `PATH`
4. Verify:

   ```bash
   keysync version
   keysync doctor
   ```

Releases are created as **drafts** by the release workflow — publish manually after reviewing assets and `checksums.txt`.

### macOS notarized builds

Darwin archives are code-signed and notarized when release secrets (`DEVELOPER_ID_CERT`, `APPLE_ID`, etc.) are configured. First launch may still require allowing the app in **System Settings → Privacy & Security**.

## Install with Go

Requires Go 1.25+:

```bash
go install github.com/dipockdas/keysync/cmd/keysync@latest
```

The binary is installed to `$GOPATH/bin` or `$HOME/go/bin`. Ensure that directory is on your `PATH`.

Install a specific version:

```bash
go install github.com/dipockdas/keysync/cmd/keysync@v1.0.3
```

## Build from source

```bash
git clone https://github.com/dipockdas/keysync.git
cd keysync
make build              # macOS: make build-signed when you have a Developer ID certificate
export PATH="$PWD/bin:$PATH"
keysync trust           # macOS: after every build or copy — avoids repeated keychain prompts
keysync version
keysync doctor
```

Optional: install to `/usr/local/bin` (run `keysync trust` again on macOS after copying):

```bash
cp ./bin/keysync /usr/local/bin/keysync
keysync trust
```

macOS: signed builds and install helper:

```bash
make build-signed
make install-signed   # ~/.local/bin/keysync
keysync trust
```

## Prerequisites

| Tool | Required for |
|------|----------------|
| `gh` CLI | `keysync push`, `keysync pull`, GitHub sync |
| Platform CLIs | Optional — `keysync migrate --cloud` only |

Authenticate GitHub:

```bash
gh auth login
```

## Next steps

- [Quick start](../README.md#quick-start)
- [Platform setup](platform-setup.md)
- [Configuration](configuration.md)
