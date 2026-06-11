# typed: false
# frozen_string_literal: true

# Install:
#   brew tap dipockdas/keysync https://github.com/dipockdas/keysync
#   brew install keysync
#
# Bump version/checksums after a release:
#   ./scripts/update-homebrew-formula.sh v1.0.7
class Keysync < Formula
  desc "Unified secret management — OS keychain, GitHub Secrets, deployment platforms"
  homepage "https://github.com/dipockdas/keysync"
  version "1.0.7"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_darwin_arm64.zip"
      sha256 "1c5120b239d121a28427622cfc73fde553e14bea15e5f047e5dcd55c13efd1ff"
    end
    on_intel do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_darwin_amd64.zip"
      sha256 "f4ba4e9b4cea6aa8b4fceb8ec01f088ce49714e6c8f3d60276176fa646017e06"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_linux_arm64.tar.gz"
      sha256 "9742a83835350054014a2429be4159bcfd1ef2941852e5682ebff4d94612c5dd"
    end
    on_intel do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_linux_amd64.tar.gz"
      sha256 "9ad1c21028c1ad40216b1ebc27c6bb145df9c53f5550691c9339ae17c845258e"
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
