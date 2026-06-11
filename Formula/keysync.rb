# typed: false
# frozen_string_literal: true

# Install:
#   brew tap dipockdas/keysync https://github.com/dipockdas/keysync
#   brew install keysync
#
# Bump version/checksums after a release:
#   ./scripts/update-homebrew-formula.sh v1.0.9
class Keysync < Formula
  desc "Unified secret management — OS keychain, GitHub Secrets, deployment platforms"
  homepage "https://github.com/dipockdas/keysync"
  version "1.0.9"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_darwin_arm64.zip"
      sha256 "45ad40aa7c7aef660fb9fbe218ac5cb84378b8731b0774aac317850450be6c2c"
    end
    on_intel do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_darwin_amd64.zip"
      sha256 "08884214052cc24f4276a52cee8ffc94038af4068051367d94f5eb812fb6328a"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_linux_arm64.tar.gz"
      sha256 "702d9104b83eb55a30e43802cd0e34c826d62d70040f9722c629ffea9a00ca2f"
    end
    on_intel do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_linux_amd64.tar.gz"
      sha256 "6901f87f97167533a6ba51483f7ab883d1deeafd102dd23ed19cdf8e55e0af89"
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
