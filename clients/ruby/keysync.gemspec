# frozen_string_literal: true

require_relative "lib/keysync/version"

Gem::Specification.new do |spec|
  spec.name          = "keysync"
  spec.version       = KeySync::VERSION
  spec.authors       = ["keysync contributors"]
  spec.summary       = "Read secrets from the OS keychain"
  spec.description   = "Keysync client library for Ruby. Reads secrets from macOS Keychain, Linux libsecret, and Windows Credential Manager."
  spec.license       = "MIT"
  spec.required_ruby_version = ">= 3.0"

  spec.metadata["source_code_uri"]     = "https://github.com/dipockdas/keysync"
  spec.metadata["bug_tracker_uri"]     = "https://github.com/dipockdas/keysync/issues"

  spec.files = Dir["lib/**/*.rb"]

  spec.add_development_dependency "minitest", "~> 5.0"
  spec.add_development_dependency "rake", "~> 13.0"
end
