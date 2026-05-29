# When to use keysync

Keysync is built for **developer-level secret management** — local development, CI/CD pipelines, and small-to-medium team workflows. It uses the credential stores you already trust on your machine (macOS Keychain, Linux libsecret, Windows Credential Manager).

## Good fit

- Individual developers and small teams replacing `.env` files
- Side projects and startup workflows where secrets should not live in git
- Teams that want one local source of truth and a single `keysync push` to GitHub + deployment platforms
- Projects that already use `gh`, Vercel, Railway, Supabase, or other platforms with CLI/HTTP APIs

## When to add another layer

For large-scale production deployments, many organizations also use dedicated secrets managers (AWS Secrets Manager, HashiCorp Vault, GCP Secret Manager, etc.) with stricter audit, rotation policies, and access control.

Keysync client libraries check **environment variables first**, then the OS keychain. That means apps can read `DATABASE_URL` from the platform in production without code changes — whether those vars were set by `keysync export`, `keysync push`, or a managed secrets service.

## OS keychain trust model

If your OS keychain is compromised, the workstation has broader problems than any single app can solve. Keysync aligns with the same model used for SSH keys, code-signing identities, and browser passwords: OS-native storage for developer machines, with optional NaCl-encrypted file fallback for headless environments (see [troubleshooting.md](troubleshooting.md)).
