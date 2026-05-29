# Platform setup

Keysync uses each operating system's native credential store automatically. This guide covers edge cases and troubleshooting.

## macOS

- Uses the built-in `security` CLI — no extra install
- Secrets are generic passwords in the login keychain
- Index file: `~/.config/keysync/index.json` (for fast listing)

### Avoiding keychain prompts

macOS may prompt for your keychain password when accessing secrets.

**Option 1 — Sign the binary** (persists "Always Allow" across rebuilds):

```bash
make build-signed      # or: make build && make sign
make install-signed    # ~/.local/bin/keysync
keysync trust          # once per install path; no secret values printed
```

Requires a Developer ID Application certificate. See [Apple's documentation](https://developer.apple.com/developer-id).

Prefer `keysync export KEY` for scripts (one read). Use `keysync export` without `KEY` only when you need every matching secret in the shell.

**Option 2 — Fallback store** (no keychain prompts; for scripts and headless use):

```bash
export KEYSYNC_STORE=fallback
keysync get DATABASE_URL
```

**Service launch scripts:**

```bash
eval $(keysync --store fallback export --project my-app)
exec my-app
```

View secrets in **Keychain Access.app** — search for `keysync`.

## Linux

Install `secret-tool` (libsecret):

```bash
# Debian / Ubuntu
sudo apt-get install libsecret-tools

# Fedora
sudo dnf install libsecret

# Arch
sudo pacman -S libsecret
```

Requires D-Bus and an unlocked keyring (GNOME Keyring, KDE Wallet, KeePassXC).

### Headless servers

```bash
sudo apt-get install libsecret-tools gnome-keyring dbus-x11
export $(dbus-launch)
echo -n "" | gnome-keyring-daemon --unlock --daemonize --components=secrets
```

If libsecret is unavailable, use `--store fallback` (see [troubleshooting.md](troubleshooting.md)).

## Windows

- Uses Windows Credential Manager (Win32 API) — no extra install
- Windows 10+ desktop and server
- View: **Control Panel → Credential Manager → Windows Credentials**
- CLI: `cmdkey /list`, `cmdkey /delete`

### Limitations

- **256-character target limit** — very long project/env/key names may fail after percent-encoding
- **User-scoped** — credentials are not shared across Windows user accounts
- **Wire format** — tagged `keysync|s=…|p=…|e=…|k=…` format; legacy underscore entries remain readable

See [architecture.md](architecture.md#windows-wire-format) for format details.
