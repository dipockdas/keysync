# frozen_string_literal: true

module KeySync
  # Builds and parses keychain service names.
  module Service
    KEYCHAIN_PREFIX = "keysync"

    # Build a keychain service name from scope and optional project.
    #
    # Global scope:  "keysync/global"
    # Project scope: "keysync/project/<project>"
    #
    # @param scope [String] "global" or "project"
    # @param project [String, nil] project name (only meaningful for project scope)
    # @return [String] full service name
    def self.service_name(scope, project = nil)
      if !project || scope != "project"
        "#{KEYCHAIN_PREFIX}/#{scope}"
      else
        "#{KEYCHAIN_PREFIX}/project/#{project}"
      end
    end

    # Parse a keychain service name into (scope, project).
    #
    # "keysync/global"        => ["global", nil]
    # "keysync/project/myapp" => ["project", "myapp"]
    # "other/global"          => ["global", nil]
    #
    # @param svc [String] the raw service name
    # @return [Array(String, String|nil)] scope and optional project
    def self.parse_service_name(svc)
      return ["global", nil] unless svc.start_with?("#{KEYCHAIN_PREFIX}/")

      trimmed = svc.delete_prefix("#{KEYCHAIN_PREFIX}/")
      return ["global", nil] unless trimmed.include?("/")

      parts = trimmed.split("/", 2)
      scope = parts[0].to_s
      project = parts[1]

      if scope == "project"
        [scope, project]
      else
        [scope || "global", nil]
      end
    end
  end
end
