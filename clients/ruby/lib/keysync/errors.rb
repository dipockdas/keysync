# frozen_string_literal: true

module KeySync
  # Base error for all keysync client operations.
  # Provides a machine-readable +code+ and a human-readable message.
  class KeySyncError < StandardError
    # @return [String] machine-readable error code:
    #   not_found, keychain_error, unsupported_platform
    attr_reader :code

    # @param code [String, Symbol] machine-readable error code
    # @param message [String] human-readable error message
    def initialize(code, message)
      @code = code.to_s
      super(message)
    end
  end

  # Raised when a secret cannot be found in any scope (project or global).
  class SecretNotFoundError < KeySyncError
    # @return [String] the key that was not found
    attr_reader :key

    # @param key [String] the secret key that was not found
    def initialize(key)
      @key = key
      super("not_found", "secret not found: #{key}")
    end
  end
end
