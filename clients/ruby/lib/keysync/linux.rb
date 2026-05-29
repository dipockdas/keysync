# frozen_string_literal: true

require "open3"
require_relative "errors"

module KeySync
  # Linux keychain access via the `secret-tool` CLI (libsecret).
  module Linux
    module_function

    # Retrieve a secret from the Linux libsecret keyring.
    #
    # Uses `secret-tool lookup service <service> account <account>`.
    #
    # @param service [String] keychain service name
    # @param account [String] key/account name
    # @return [String] the secret value
    # @raise [SecretNotFoundError] if the item does not exist
    # @raise [KeySyncError] if secret-tool fails unexpectedly
    def get(service, account)
      stdout, stderr, status = Open3.capture3(
        "secret-tool", "lookup",
        "keysync-service", service,
        "keysync-key", account
      )

      if status.success?
        val = stdout.strip
        raise SecretNotFoundError, "#{service}/#{account}" if val.empty?
        return val
      end

      # secret-tool returns exit code 1 for "not found"
      raise SecretNotFoundError, "#{service}/#{account}"
    rescue Errno::ENOENT
      raise KeySyncError.new("keychain_error",
        "secret-tool not found. Install libsecret-tools: sudo apt install libsecret-tools"
      )
    end

    # List all keysync secrets from the Linux keyring.
    #
    # Uses `secret-tool search --all keysync-service` to find all entries
    # whose service name starts with "keysync/", then parses the output.
    #
    # @return [Array<Hash>] array of {service:, account:} hashes
    def list
      stdout, _stderr, status = Open3.capture3(
        "secret-tool", "search", "--all",
        "keysync-service", Service::KEYCHAIN_PREFIX
      )
      return [] unless status.success?
      return [] if stdout.empty?

      entries = []
      current_svc = ""
      current_key = ""

      stdout.each_line do |line|
        trimmed = line.strip
        if trimmed.empty?
          if !current_svc.empty? && !current_key.empty? &&
             current_svc.start_with?("#{Service::KEYCHAIN_PREFIX}/")
            entries << { "service" => current_svc, "account" => current_key }
          end
          current_svc = ""
          current_key = ""
          next
        end

        if trimmed.start_with?("keysync-service = ")
          current_svc = parse_attr(trimmed)
        elsif trimmed.start_with?("keysync-key = ")
          current_key = parse_attr(trimmed)
        end
      end

      # Handle last entry if no trailing blank line
      if !current_svc.empty? && !current_key.empty? &&
         current_svc.start_with?("#{Service::KEYCHAIN_PREFIX}/")
        entries << { "service" => current_svc, "account" => current_key }
      end

      entries
    rescue Errno::ENOENT
      []
    end

    # Parse an "attribute = value" line.
    #
    # @param line [String] a line like "keysync-service = keysync/global"
    # @return [String] the value portion
    def parse_attr(line)
      eq_idx = line.index("=")
      eq_idx ? line[(eq_idx + 1)..].strip : ""
    end
  end
end
