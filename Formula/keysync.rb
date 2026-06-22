# typed: false
# frozen_string_literal: true

# Install:
#   brew tap dipockdas/keysync https://github.com/dipockdas/keysync
#   brew install keysync
#
# Bump version/checksums after a release:
#   ./scripts/update-homebrew-formula.sh v1.0.10
class Keysync < Formula
  desc "Unified secret management — OS keychain, GitHub Secrets, deployment platforms"
  homepage "https://github.com/dipockdas/keysync"
  version "1.0.10"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_darwin_arm64.zip"
      sha256 "055b0dcfc1b33cb0073f34705d6de08d3780ef7ec9d7932650d7bb2d352a274b"
    end
    on_intel do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_darwin_amd64.zip"
      sha256 "285b63c5a1c53d92012df0a7094fddb4a2d52e179b7b26d58f4a437b8c247a28"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_linux_arm64.tar.gz"
      sha256 "8e7bb7f1932a5dd20aa9dc560f111b10c632747487252f6efc49996aec80cb2a"
    end
    on_intel do
      url "https://github.com/dipockdas/keysync/releases/download/v#{version}/keysync_linux_amd64.tar.gz"
      sha256 "e8e3ede228d3f5287ad7ebec6c1e14c53fe1c0b58c772a7713326eaa2df25b10"
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
