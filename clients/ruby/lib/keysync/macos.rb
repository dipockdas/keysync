# frozen_string_literal: true

require "open3"
require_relative "errors"

module KeySync
  # macOS keychain access via the built-in `security` CLI.
  module MacOS
    module_function

    # Retrieve a secret from the macOS Keychain.
    #
    # On newer macOS versions, the password is written to stderr by the
    # `security` CLI. We check both stdout and stderr and strip whitespace.
    #
    # @param service [String] keychain service name
    # @param account [String] key/account name
    # @return [String] the secret value
    # @raise [SecretNotFoundError] if the item does not exist
    # @raise [KeySyncError] if the keychain tool fails unexpectedly
    def get(service, account)
      stdout, stderr, status = Open3.capture3(
        "security", "find-generic-password",
        "-s", service,
        "-a", account,
        "-w"
      )

      output = stdout.strip
      # On newer macOS, the password goes to stderr instead of stdout
      output = stderr.strip if output.empty?

      if status.success? && !output.empty?
        return output
      end

      # Exit code 44 means "item not found" in the security CLI
      if status.exitstatus == 44
        raise SecretNotFoundError, "#{service}/#{account}"
      end

      raise KeySyncError.new("keychain_error",
        "keychain read failed (exit #{status.exitstatus}): #{stderr.strip}"
      )
    end

    # List all keysync secrets from the macOS Keychain.
    #
    # Parses the output of `security dump-keychain` to find all
    # generic password entries whose service name starts with "keysync/".
    #
    # @return [Array<Hash>] array of {service:, account:} hashes
    def list
      stdout, _stderr, status = Open3.capture3("security", "dump-keychain")
      return [] unless status.success?
      return [] if stdout.empty?

      records = stdout.split("\nkeychain:")
      entries = []

      records.each do |rec|
        next unless rec.include?('class: "genp"')

        svc = find_attr(rec, "svce")
        next if svc.nil? || !svc.start_with?("#{Service::KEYCHAIN_PREFIX}/")

        acct = find_attr(rec, "acct")
        entries << { "service" => svc, "account" => acct } if acct
      end

      entries
    end

    # Extract a named attribute value from a keychain dump record.
    #
    # @param record [String] a single keychain record block
    # @param attr_name [String] attribute name (e.g. "svce", "acct")
    # @return [String, nil] the attribute value or nil
    def find_attr(record, attr_name)
      idx = record.index(%("#{attr_name}"))
      return nil unless idx

      after = record[(idx + attr_name.length + 2)..]
      eq_idx = after.index("=")
      return nil unless eq_idx

      val = after[(eq_idx + 1)..].strip
      return nil if val == "<NULL>"

      if val.start_with?('"')
        end_idx = val.index('"', 1)
        return val[1...end_idx] if end_idx
        return val[1..]
      end

      val.delete('"')
    end
  end
end
