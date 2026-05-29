# GitHub repository settings

Checklist for open-sourcing **keysync**. Some settings require the repository to be **public** (or GitHub Pro on a private repo).

## 1. Make the repository public

When ready to launch:

```bash
gh repo edit dipockdas/keysync --visibility public
```

Or: **Settings → General → Danger zone → Change visibility**.

## 2. Private vulnerability reporting

Allows researchers to report issues without public disclosure.

**UI:** Settings → Security → **Private vulnerability reporting** → Enable

**API** (after repo exists and you have admin access):

```bash
gh api repos/dipockdas/keysync -X PATCH \
  --input - <<'EOF'
{
  "security_and_analysis": {
    "private_vulnerability_reporting": {
      "status": "enabled"
    }
  }
}
EOF
```

Also enable **Dependabot alerts** and **Dependabot security updates** under Settings → Security.

## 3. Branch protection on `main`

Requires a **public** repository (free) or **GitHub Pro** (private).

**Recommended rules:**

- Require pull request before merging
- Require status checks to pass:
  - `test` (CI)
  - `govulncheck` (Security)
  - `gitleaks` (Security)
  - `analyze` (CodeQL) — optional at first, enable after first CodeQL run
- Require branches to be up to date
- Do not allow force pushes

**Script** (run once the repo is public):

```bash
./scripts/enable-branch-protection.sh
```

**Manual UI:** Settings → Branches → Add rule for `main`.

## 4. Required workflows

These workflows should pass before merging to `main`:

| Workflow | File | Purpose |
|----------|------|---------|
| CI | `.github/workflows/ci.yml` | Go tests |
| Security | `.github/workflows/security.yml` | govulncheck + gitleaks |
| Cross-Platform | `.github/workflows/cross-platform.yml` | OS matrix |
| CodeQL | `.github/workflows/codeql.yml` | Static analysis |
| OpenSSF Scorecard | `.github/workflows/scorecard.yml` | Supply-chain score |

## 5. Release process

1. Tag a release: `git tag v1.0.4 && git push origin v1.0.4`
2. [Release workflow](../.github/workflows/release.yml) builds artifacts (draft release)
3. Review draft on GitHub → edit notes → **Publish release**

See [install.md](install.md) for download instructions to add to release notes.

## 6. Repository metadata

Suggested **Settings → General** values:

- **Description:** Unified secret management — OS keychain, GitHub Secrets, deployment platforms
- **Website:** README link or project docs
- **Topics:** `secrets`, `keychain`, `cli`, `devops`, `github-actions`, `vercel`, `environment-variables`
- **Issues:** Enabled
- **Discussions:** Optional (alternative to Discord while community is small)

## 7. Community channels

Discord server (optional): create when ready, then add to README:

```markdown
[![Discord](https://img.shields.io/badge/Discord-join-5865F2?style=for-the-badge&logo=discord&logoColor=white)](https://discord.gg/YOUR_INVITE)
```

No Twitter/X link required.

## 8. README badges after going public

While the repo is **private**, shields.io and OpenSSF cannot read GitHub stats — badges show `REPO NOT FOUND` or `UNABLE TO SELECT NEXT GITHUB TOKEN FROM POOL`. The README uses **workflow badges** and static links until launch.

After making the repo public, optionally replace the top badge row with:

```markdown
[![GitHub stars](https://img.shields.io/github/stars/dipockdas/keysync?style=for-the-badge&logo=github)](https://github.com/dipockdas/keysync/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/dipockdas/keysync?style=for-the-badge&logo=github)](https://github.com/dipockdas/keysync/network/members)
[![GitHub issues](https://img.shields.io/github/issues/dipockdas/keysync?style=for-the-badge&logo=github)](https://github.com/dipockdas/keysync/issues)
[![GitHub contributors](https://img.shields.io/github/contributors/dipockdas/keysync?style=for-the-badge&logo=github)](https://github.com/dipockdas/keysync/graphs/contributors)
[![Last commit](https://img.shields.io/github/last-commit/dipockdas/keysync/main?style=for-the-badge&logo=github)](https://github.com/dipockdas/keysync/commits/main)
```

And add (once a release is published and GoReportCard has indexed the repo):

```markdown
[![Release](https://img.shields.io/github/v/release/dipockdas/keysync)](https://github.com/dipockdas/keysync/releases)
[![Go Report](https://goreportcard.com/badge/github.com/dipockdas/keysync)](https://goreportcard.com/report/github.com/dipockdas/keysync)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/dipockdas/keysync/badge)](https://scorecard.dev/viewer/?uri=github.com/dipockdas/keysync)
```

Then enable full CodeQL / Scorecard upload:

1. **Settings → Code security → Code scanning** → enable for the repository
2. In `.github/workflows/codeql.yml` set `upload: true`
3. In `.github/workflows/scorecard.yml` set `publish_results: true` and uncomment the SARIF upload step
4. Remove `if: ${{ !github.event.repository.private }}` from the Scorecard job
