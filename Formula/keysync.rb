# typed: false
# frozen_string_literal: true

# Install:
#   brew tap dipockdas/keysync https://github.com/dipockdas/keysync
#   brew install keysync
#
# Bump version/checksums after a release:
#   ./scripts/update-homebrew-formula.sh v1.0.11
class Keysync < Formula
  desc "Unified secret management — OS keychain, GitHub Secrets, deployment platforms"
  homepage "https://github.com/dipockdas/keysync"
  version "1.0.11"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_darwin_arm64.zip"
      sha256 "0caeb475459ed3210272bb16c16eb47adcc54f5f7a34961a94afd9628671ac39"
    end
    on_intel do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_darwin_amd64.zip"
      sha256 "c08f9a32e32b1b0ab957375ba8dda5c8aa6b3f2a40d22347ab4eca262d919c7b"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_linux_arm64.tar.gz"
      sha256 "3e047355d8f8195dcf45d36856391086ab283ff4808787d18c7b991006501352"
    end
    on_intel do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_linux_amd64.tar.gz"
      sha256 "e5cb2d24fc96570530d22874fdd15946d87d65f3474ba72b83d6515d1855304a"
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
