# frozen_string_literal: true

module KeySync
  # A single credential entry stored in the OS keychain.
  #
  # @!attribute [r] service
  #   @return [String] the keychain service name (e.g. "keysync/global")
  # @!attribute [r] account
  #   @return [String] the account/key name stored under this service
  # @!attribute [r] scope
  #   @return [String] parsed scope ("global" or "project")
  # @!attribute [r] project
  #   @return [String, nil] parsed project name or nil for global scope
  CredentialEntry = Struct.new(:service, :account, :scope, :project) do
    # Create a CredentialEntry from raw service + account strings.
    # Scope and project are auto-parsed from the service name.
    #
    # @param service [String] the raw keychain service name
    # @param account [String] the account/key name
    # @return [CredentialEntry]
    def self.from_raw(service, account)
      scope, project = Service.parse_service_name(service)
      new(service, account, scope, project)
    end
  end
end
