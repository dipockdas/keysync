# Troubleshooting

## Secret not found

```bash
keysync list                    # see scope and key names
keysync list --project my-app
keysync get KEY -p my-app -u    # print value to stdout
```

Remember: `keysync get` falls back global ← project ← env when `-p` is set. `keysync rm` and `keysync mv` use **exact** scope from flags.

## Fallback storage (headless / no keychain)

When the OS keychain is unavailable, opt in explicitly:

```bash
keysync --store fallback set -p my-app API_KEY=value
export KEYSYNC_STORE=fallback
```

Data is stored in `~/.config/keysync/store.json` encrypted with NaCl (key in `~/.config/keysync/key`). This is weaker than OS keychain protection because key and ciphertext share the filesystem.

**Production alternatives:**

- SSH agent forwarding or remote keychain access
- Platform env vars in CI (`export VERCEL_TOKEN=…` then `keysync push`)
- Dedicated secrets managers (Vault, AWS Secrets Manager, etc.)

## CI/CD without keychain

Set platform tokens via environment variables:

```bash
export VERCEL_TOKEN=...
export RAILWAY_TOKEN=...
keysync push -p my-app
```

GitHub Actions typically receives secrets from repository settings; keysync is used on developer machines and optional sync workflows, not as a hosted runtime.

## macOS repeated keychain prompts

Each keychain read can prompt if the binary is unsigned or was rebuilt/copied to a new path.

**Recommended (local dev):**

```bash
make build-signed          # Developer ID sign (persists "Always Allow" across rebuilds)
make install-signed        # installs to ~/.local/bin/keysync
keysync trust              # updates ACLs for all indexed secrets (no values printed)
```

**Export one secret** (one keychain read):

```bash
eval $(keysync export API_KEY)
eval $(keysync export DATABASE_URL -p my-app)
```

**Export everything** (one read per secret — many prompts if trust is missing):

```bash
eval $(keysync export -p my-app)
```

**Headless / CI:** use `KEYSYNC_STORE=fallback` or platform env vars. See [platform-setup.md](platform-setup.md).

## Linux D-Bus / libsecret errors

Install `libsecret-tools`, ensure `dbus-launch` and an unlocked keyring, or use fallback mode.

## Windows credential limits

Keep project, environment, and key names reasonably short (256-character target limit). See [platform-setup.md](platform-setup.md#limitations).

## Push failures

```bash
keysync doctor
gh auth status
```

Verify `.keysync.json` project IDs and tokens. See [pushing-secrets.md](pushing-secrets.md).
