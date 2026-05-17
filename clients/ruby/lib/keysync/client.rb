# frozen_string_literal: true

require_relative "errors"
require_relative "service"

module KeySync
  # Client for retrieving secrets from the OS keychain.
  #
  # Implements the resolution order:
  # 1. Environment variable (ENV[key]) -- for cloud/CI where platform injects env vars
  # 2. OS keychain -- project scope first, then global scope
  module Client
    module_function

    # Retrieve a secret.
    #
    # Resolution order:
    # 1. Check ENV[key] first -- supports cloud/CI environments
    # 2. If project is provided, check project scope first
    # 3. Fall back to global scope
    #
    # @param key [String] the secret key name
    # @param project [String, nil] optional project name for project-scoped secrets
    # @return [String] the secret value
    # @raise [SecretNotFoundError] if the secret cannot be found in any scope
    # @raise [KeySyncError] if the keychain tool fails
    def get_secret(key, project: nil)
      # Primary path: check environment variable first.
      # In local dev the user runs eval $(keysync export) at shell startup;
      # in cloud/CI the platform injects env vars directly.
      return ENV[key] if ENV.key?(key)

      platform_get = KeySync.platform_get
      platform_get || raise_unsupported_platform

      # Try project scope first
      if project
        svc = Service.service_name("project", project)
        begin
          result = platform_get.call(svc, key)
          return result if result
        rescue SecretNotFoundError
          # fall through to global
        end
      end

      # Fall back to global scope
      svc = Service.service_name("global")
      begin
        result = platform_get.call(svc, key)
        return result if result
      rescue SecretNotFoundError
        raise SecretNotFoundError, key
      end

      raise SecretNotFoundError, key
    end

    # List all stored secret entries for the given project.
    #
    # Returns global secrets plus project-specific secrets. Entries are
    # returned as CredentialEntry structs with scope and project parsed.
    #
    # @param project [String, nil] optional project name to filter by
    # @return [Array<CredentialEntry>] list of credential entries
    def list_secrets(project: nil)
      platform_list = KeySync.platform_list
      return [] unless platform_list

      entries = platform_list.call
      results = []

      entries.each do |entry|
        svc = entry["service"]
        acct = entry["account"]
        scope, entry_project = Service.parse_service_name(svc)

        if project
          # Include globals plus this project's entries
          if scope == "global" || (scope == "project" && entry_project == project)
            results << CredentialEntry.new(svc, acct, scope, entry_project)
          end
        else
          # Include everything
          results << CredentialEntry.new(svc, acct, scope, entry_project)
        end
      end

      results
    end

    # @api private
    def raise_unsupported_platform
      raise KeySyncError.new("unsupported_platform",
        "unsupported platform: #{RUBY_PLATFORM}"
      )
    end
  end
end
