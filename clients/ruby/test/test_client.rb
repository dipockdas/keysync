# frozen_string_literal: true

require_relative "test_helper"

class TestClient < Minitest::Test
  def teardown
    # Clean up any env vars we set during tests
    ENV.delete("TEST_KEY")
    ENV.delete("TEST_PROJECT_KEY")
    ENV.delete("TEST_EMPTY")
  end

  # Env var tests use a mock get proc so we can verify that
  # the env var short-circuits before any keychain call.

  def test_env_var_short_circuits_get_secret
    ENV["TEST_KEY"] = "from-env"
    called = false

    mock_get = ->(svc, acct) {
      called = true
      "should-not-be-called"
    }
    KeySync.platform_get = mock_get

    result = KeySync.get_secret("TEST_KEY")
    assert_equal "from-env", result
    refute called, "platform get should not have been called when env var exists"
  ensure
    KeySync.platform_get = nil
  end

  def test_env_var_with_project_still_short_circuits
    ENV["TEST_PROJECT_KEY"] = "from-env-project"
    called = false

    mock_get = ->(svc, acct) {
      called = true
      "should-not-be-called"
    }
    KeySync.platform_get = mock_get

    result = KeySync.get_secret("TEST_PROJECT_KEY", project: "myapp")
    assert_equal "from-env-project", result
    refute called, "env var should short-circuit even when project is provided"
  ensure
    KeySync.platform_get = nil
  end

  def test_falls_back_to_project_scope_when_no_env_var
    ENV.delete("TEST_KEY")

    calls = []
    mock_get = ->(svc, acct) {
      calls << [svc, acct]
      if svc == "keysync/project/myapp"
        "project-value"
      else
        raise KeySync::SecretNotFoundError, "#{svc}/#{acct}"
      end
    }
    KeySync.platform_get = mock_get

    result = KeySync.get_secret("TEST_KEY", project: "myapp")
    assert_equal "project-value", result
    assert_equal ["keysync/project/myapp", "TEST_KEY"], calls.first
  ensure
    KeySync.platform_get = nil
  end

  def test_falls_back_to_global_when_project_not_found
    ENV.delete("TEST_KEY")

    calls = []
    mock_get = ->(svc, acct) {
      calls << [svc, acct]
      if svc == "keysync/global"
        "global-value"
      else
        raise KeySync::SecretNotFoundError, "#{svc}/#{acct}"
      end
    }
    KeySync.platform_get = mock_get

    result = KeySync.get_secret("TEST_KEY", project: "myapp")
    assert_equal "global-value", result
    # Should have tried project scope first, then global
    assert_equal 2, calls.length
    assert_equal ["keysync/project/myapp", "TEST_KEY"], calls[0]
    assert_equal ["keysync/global", "TEST_KEY"], calls[1]
  ensure
    KeySync.platform_get = nil
  end

  def test_list_secrets_with_project_includes_globals
    entries = [
      { "service" => "keysync/global", "account" => "API_KEY" },
      { "service" => "keysync/project/myapp", "account" => "DB_URL" },
      { "service" => "keysync/project/other", "account" => "OTHER_KEY" },
    ]

    KeySync.platform_list = -> { entries }

    results = KeySync.list_secrets(project: "myapp")
    assert_equal 2, results.length

    result_services = results.map(&:service).sort
    expected = %w[
      keysync/global
      keysync/project/myapp
    ].sort
    assert_equal expected, result_services

    # Verify CredentialEntry structure
    results.each do |entry|
      assert_kind_of KeySync::CredentialEntry, entry
      assert entry.service
      assert entry.account
      assert entry.scope
    end
  ensure
    KeySync.platform_list = nil
  end

  def test_list_secrets_without_project_returns_all
    entries = [
      { "service" => "keysync/global", "account" => "API_KEY" },
      { "service" => "keysync/project/myapp", "account" => "DB_URL" },
    ]

    KeySync.platform_list = -> { entries }

    results = KeySync.list_secrets
    assert_equal 2, results.length
  ensure
    KeySync.platform_list = nil
  end

  def test_get_secret_raises_not_found_when_all_scopes_exhausted
    ENV.delete("TEST_KEY")

    mock_get = ->(svc, acct) {
      raise KeySync::SecretNotFoundError, "#{svc}/#{acct}"
    }
    KeySync.platform_get = mock_get

    assert_raises(KeySync::SecretNotFoundError) do
      KeySync.get_secret("TEST_KEY", project: "myapp")
    end
  ensure
    KeySync.platform_get = nil
  end
end
