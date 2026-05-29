#!/usr/bin/env bash
# Regenerate Formula/keysync.rb version and sha256 lines from a GitHub release.
# Usage: ./scripts/update-homebrew-formula.sh [v1.0.3]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FORMULA="${ROOT}/Formula/keysync.rb"
REPO="${KEYSYNC_REPO:-dipockdas/keysync}"

TAG="${1:-}"
if [[ -z "${TAG}" ]]; then
  TAG="$(gh release view --repo "${REPO}" --json tagName -q .tagName)"
fi
TAG="${TAG#v}"
VERSION="${TAG}"

echo "Updating formula for version ${VERSION} from ${REPO} ..."

declare -A SHA
while IFS= read -r line; do
  name="${line%% *}"
  hash="${line##* }"
  SHA["${name}"]="${hash}"
done < <(gh release view "v${VERSION}" --repo "${REPO}" --json assets \
  -q '.assets[] | select(.name | startswith("keysync_")) | "\(.name) \(.digest)"' \
  | sed 's/sha256://')

get_sha() {
  local asset="$1"
  local h="${SHA[$asset]:-}"
  if [[ -z "${h}" ]]; then
    echo "error: missing digest for ${asset} (is the release published?)" >&2
    exit 1
  fi
  echo "${h}"
}

cat > "${FORMULA}" <<RUBY
# typed: false
# frozen_string_literal: true

# Install:
#   brew tap dipockdas/keysync https://github.com/dipockdas/keysync
#   brew install keysync
#
# Bump version/checksums after a release:
#   ./scripts/update-homebrew-formula.sh v${VERSION}
class Keysync < Formula
  desc "Unified secret management — OS keychain, GitHub Secrets, deployment platforms"
  homepage "https://github.com/dipockdas/keysync"
  version "${VERSION}"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_darwin_arm64.zip"
      sha256 "$(get_sha keysync_darwin_arm64.zip)"
    end
    on_intel do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_darwin_amd64.zip"
      sha256 "$(get_sha keysync_darwin_amd64.zip)"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_linux_arm64.tar.gz"
      sha256 "$(get_sha keysync_linux_arm64.tar.gz)"
    end
    on_intel do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_linux_amd64.tar.gz"
      sha256 "$(get_sha keysync_linux_amd64.tar.gz)"
    end
  end

  def install
    bin.install "keysync"
  end

  def caveats
    <<~EOS
      keysync uses your OS keychain and syncs via the GitHub CLI.

      Install and authenticate gh if you have not already:
        brew install gh
        gh auth login

      Quick start:
        keysync init --project my-app
        keysync set API_KEY=your_value
        keysync push -p my-app
    EOS
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/keysync version")
  end
end
RUBY

echo "Wrote ${FORMULA}"
echo "Commit and push, then: brew update && brew upgrade keysync"
