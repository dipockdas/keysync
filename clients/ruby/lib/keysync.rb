# frozen_string_literal: true

# KeySync -- Retrieve secrets from the OS-native keychain.
#
# Each platform uses its native keychain tooling:
#   macOS:   security CLI (built-in)
#   Linux:   secret-tool CLI (libsecret)
#   Windows: PowerShell with inline C# (CredReadW from advapi32.dll)
#
# Usage:
#   require "keysync"
#
#   db_url = KeySync.get_secret("DATABASE_URL", project: "my-api")
#   api_key = KeySync.get_secret("GLOBAL_API_KEY")
#   globals = KeySync.list_secrets
#   project = KeySync.list_secrets(project: "my-api")

require_relative "keysync/version"
require_relative "keysync/errors"
require_relative "keysync/service"
require_relative "keysync/credential"
require_relative "keysync/client"

# Conditionally load platform-specific backends
case RUBY_PLATFORM
when /darwin/
  require_relative "keysync/macos"
when /linux/
  require_relative "keysync/linux"
when /mingw|mswin/
  require_relative "keysync/windows"
end

module KeySync
  class << self
    # @api private
    # @return [Proc, nil] the platform-specific get proc, or nil if unsupported
    def platform_get
      @platform_get
    end

    # @api private
    # Set the platform-specific get function.
    # @param func [Proc] a callable taking (service, account) and returning the secret
    def platform_get=(func)
      @platform_get = func
    end

    # @api private
    # @return [Proc, nil] the platform-specific list proc, or nil if unsupported
    def platform_list
      @platform_list
    end

    # @api private
    # Set the platform-specific list function.
    # @param func [Proc] a callable returning an array of {service:, account:} hashes
    def platform_list=(func)
      @platform_list = func
    end

    # Retrieve a secret from the OS keychain.
    #
    # Resolution order:
    # 1. Check ENV[key] first -- supports cloud/CI environments
    # 2. If project is provided, check project scope first
    # 3. Fall back to global scope
    #
    # @param key [String] the secret key name
    # @param project [String, nil] optional project name for project-scoped secrets
    # @return [String] the secret value
    # @raise [SecretNotFoundError] if the secret cannot be found
    # @raise [KeySyncError] if the keychain tool fails or platform is unsupported
    #
    # @example Get a global secret
    #   api_key = KeySync.get_secret("API_KEY")
    #
    # @example Get a project-scoped secret (falls back to global)
    #   db_url = KeySync.get_secret("DATABASE_URL", project: "myapp")
    def get_secret(key, project: nil)
      Client.get_secret(key, project: project)
    end

    # List all stored secret entries.
    #
    # @param project [String, nil] optional project name to filter by.
    #   When provided, returns global secrets plus this project's secrets.
    # @return [Array<CredentialEntry>] list of credential entries
    #
    # @example List all global secrets
    #   globals = KeySync.list_secrets
    #
    # @example List project secrets (includes global fallback)
    #   project = KeySync.list_secrets(project: "myapp")
    def list_secrets(project: nil)
      Client.list_secrets(project: project)
    end
  end

  # Register platform backends at load time.
  # Platform detection via RUBY_PLATFORM:
  #   /darwin/     => macOS
  #   /linux/      => Linux
  #   /mingw|mswin/ => Windows
  case RUBY_PLATFORM
  when /darwin/
    self.platform_get = MacOS.method(:get).to_proc
    self.platform_list = MacOS.method(:list).to_proc
  when /linux/
    self.platform_get = Linux.method(:get).to_proc
    self.platform_list = Linux.method(:list).to_proc
  when /mingw|mswin/
    self.platform_get = Windows.method(:get).to_proc
    self.platform_list = Windows.method(:list).to_proc
  end
end
