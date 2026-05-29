# Security scanning

Keysync runs automated security checks on every push and pull request to `main`, plus a weekly schedule.

## What runs in CI

| Check | Tool | Purpose |
|-------|------|---------|
| Vulnerability scan | [govulncheck](https://go.dev/blog/vuln) | Known vulnerabilities in Go dependencies |
| Secret detection | [Gitleaks](https://github.com/gitleaks/gitleaks) | Full-repo scan (`gitleaks detect`); avoids push-range errors on root/fresh history |
| Static analysis | [CodeQL](https://codeql.github.com/) | Security and quality queries for Go |
| Supply chain | [OpenSSF Scorecard](https://scorecard.dev/) | Repository security posture score |
| Unit tests | `go test -race` | Regression and concurrency issues |

Workflows:

- [.github/workflows/security.yml](../.github/workflows/security.yml)
- [.github/workflows/codeql.yml](../.github/workflows/codeql.yml)
- [.github/workflows/scorecard.yml](../.github/workflows/scorecard.yml)

## Badges

The README shows CI and security workflow status. After the repository is public:

- **CI** — build and test health
- **Security** — govulncheck + gitleaks
- **Go Report Card** — static analysis summary (external)

## Running locally

```bash
# Dependencies
go install golang.org/x/vuln/cmd/govulncheck@latest

# Vulnerability scan
govulncheck ./...

# Secret scan (requires gitleaks binary)
gitleaks detect --source . --verbose
```

Install Gitleaks: https://github.com/gitleaks/gitleaks#installing

## Reporting vulnerabilities

Do **not** open public issues for security bugs. Use [GitHub private vulnerability reporting](https://github.com/dipockdas/keysync/security/advisories/new). See [SECURITY.md](../SECURITY.md).

## OpenSSF Scorecard badge

After the first Scorecard workflow run on `main`, the README badge reflects your score. View details:

https://scorecard.dev/viewer/?uri=github.com/dipockdas/keysync

## Future enhancements

- Signed releases and SLSA provenance (release workflow already signs/notarizes macOS when secrets are configured)
- Add `analyze` (CodeQL) to required branch protection checks after the first successful run
