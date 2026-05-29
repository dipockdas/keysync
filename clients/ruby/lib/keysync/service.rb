# frozen_string_literal: true

module KeySync
  # Builds and parses keychain service names.
  module Service
    KEYCHAIN_PREFIX = "keysync"

    # Build a keychain service name from scope, optional project, and optional environment.
    #
    # Global scope:      "keysync/global"
    # Project scope:     "keysync/project/<project>"
    # Environment scope: "keysync/project/<project>/env/<environment>"
    #
    # @param scope [String] "global" or "project"
    # @param project [String, nil] project name (only meaningful for project scope)
    # @param environment [String, nil] environment name (e.g. "staging")
    # @return [String] full service name
    def self.service_name(scope, project = nil, environment = nil)
      if !project || scope != "project"
        "#{KEYCHAIN_PREFIX}/#{scope}"
      elsif environment && !environment.empty?
        "#{KEYCHAIN_PREFIX}/project/#{project}/env/#{environment}"
      else
        "#{KEYCHAIN_PREFIX}/project/#{project}"
      end
    end

    # Parse a keychain service name into (scope, project, environment).
    #
    # "keysync/global"                        => ["global", nil, nil]
    # "keysync/project/myapp"                 => ["project", "myapp", nil]
    # "keysync/project/myapp/env/staging"     => ["project", "myapp", "staging"]
    # "other/global"                          => ["global", nil, nil]
    #
    # @param svc [String] the raw service name
    # @return [Array(String, String|nil, String|nil)] scope, optional project, optional environment
    def self.parse_service_name(svc)
      return ["global", nil, nil] unless svc.start_with?("#{KEYCHAIN_PREFIX}/")

      trimmed = svc.delete_prefix("#{KEYCHAIN_PREFIX}/")
      return ["global", nil, nil] unless trimmed.include?("/")

      parts = trimmed.split("/", 2)
      scope = parts[0].to_s
      rest = parts[1]

      return [scope || "global", nil, nil] unless scope == "project"

      # Check for /env/ segment to detect environment
      if rest.include?("/env/")
        project, env_rest = rest.split("/env/", 2)
        [scope, project, env_rest]
      else
        [scope, rest, nil]
      end
    end
  end
end
