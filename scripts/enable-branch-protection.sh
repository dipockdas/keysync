#!/usr/bin/env bash
# Enable branch protection on main. Requires public repo or GitHub Pro.
set -euo pipefail

REPO="${1:-dipockdas/keysync}"
BRANCH="${2:-main}"

echo "Configuring branch protection for ${REPO}:${BRANCH} ..."

gh api "repos/${REPO}/branches/${BRANCH}/protection" -X PUT --input - <<'EOF'
{
  "required_status_checks": {
    "strict": true,
    "contexts": [
      "test",
      "govulncheck",
      "gitleaks"
    ]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "required_approving_review_count": 0,
    "dismiss_stale_reviews": true
  },
  "restrictions": null,
  "required_linear_history": false,
  "allow_force_pushes": false,
  "allow_deletions": false
}
EOF

echo "Done. Add CodeQL context 'analyze' after the first CodeQL workflow run:"
echo "  gh api repos/${REPO}/branches/${BRANCH}/protection -X PATCH ..."
