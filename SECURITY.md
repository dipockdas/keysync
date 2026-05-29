# Security Policy

## Supported Versions

We release security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please use **GitHub private vulnerability reporting**:

**https://github.com/dipockdas/keysync/security/advisories/new**

Select "Report a vulnerability" and include reproduction steps. You should receive a response within 48 hours. If you do not, comment on the advisory thread to follow up.

For urgent issues or if private reporting is unavailable, you may email **agent@dipockdas.com** (PGP key not required; use GitHub private reporting when possible).

Please include the following information in your report:

- Type of vulnerability (e.g., buffer overflow, SQL injection, cross-site scripting)
- Full paths of source file(s) related to the vulnerability
- Location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

This information will help us triage your report more quickly.

## Security Update Process

When we receive a security report:

1. We will confirm receipt of your vulnerability report within 48 hours
2. We will investigate and validate the vulnerability
3. We will work on a fix and prepare a security advisory
4. We will release a patched version and publish the security advisory
5. We will credit you in the security advisory (unless you prefer to remain anonymous)

## Preferred Languages

We prefer all communications to be in English.

## Safe Harbor

We support safe harbor for security researchers who:

- Make a good faith effort to avoid privacy violations, destruction of data, and interruption or degradation of our services
- Only interact with accounts you own or with explicit permission of the account holder
- Do not exploit a security issue beyond what is necessary to demonstrate it
- Report vulnerabilities promptly
- Keep vulnerability details confidential until we've had a reasonable time to address them

We will not pursue legal action against researchers who follow these guidelines.

## Security Best Practices for Users

When using keysync:

1. **Keep your OS keychain secure** - Use full disk encryption and strong passwords
2. **Rotate secrets regularly** - Use `keysync rotate` for automated rotation
3. **Limit platform token scopes** - Give tokens minimum required permissions
4. **Review GitHub Secrets** - Regularly audit what's stored in your repositories
5. **Use environment scoping** - Separate dev/staging/production secrets
6. **Never commit secrets** - The CLI is designed to prevent this, but always verify

## Known Security Considerations

### OS Keychain Access

Keysync stores secrets in your OS keychain (macOS Keychain, Linux libsecret, Windows Credential Manager). Any process running as your user can potentially access these secrets. This is the same security model used by:

- SSH keys
- GPG keys
- Browser saved passwords
- Git credentials

Compromised OS keychain = compromised system. Use full disk encryption and strong authentication.

### Platform API Tokens

Platform tokens (VERCEL_TOKEN, RAILWAY_TOKEN, SUPABASE_TOKEN) are stored as global secrets. These tokens have permissions to modify your deployments. Always:

- Use project-scoped tokens when available
- Rotate tokens if you suspect compromise
- Review token permissions in platform settings

### GitHub Secrets

Secrets synced to GitHub are encrypted at rest by GitHub. However:

- Repository admins can access GitHub Secrets
- Workflows running in the repo can read secrets
- Forks do NOT have access to secrets (GitHub security feature)

Only sync secrets to repositories you trust.

## Security Features

keysync includes several security features:

- **No plaintext storage** - All secrets stored in OS keychains
- **Read-only client libraries** - Cannot write/delete secrets in production
- **Validation checks** - Prevents syncing to wrong repositories
- **Encrypted fallback store** - Uses NaCl box encryption (Curve25519+XSalsa20-Poly1305)
- **No network calls in runtime** - Client libraries read from local keychain only
- **Platform token fallback** - Reads from env vars when keychain unavailable (CI/CD)

## Acknowledgments

We appreciate the security research community's efforts to help keep keysync and its users safe. Security researchers who responsibly disclose vulnerabilities will be credited in our security advisories.
