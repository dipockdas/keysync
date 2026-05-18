# frozen_string_literal: true

require_relative "test_helper"

class TestKeySync < Minitest::Test
  def test_version_is_string
    assert_kind_of String, KeySync::VERSION
    refute_empty KeySync::VERSION
  end

  def test_platform_detection_is_not_nil_on_known_platform
    platform = RUBY_PLATFORM
    if platform =~ /darwin|linux|mingw|mswin/
      refute_nil KeySync.platform_get, "platform_get should be set on #{platform}"
      refute_nil KeySync.platform_list, "platform_list should be set on #{platform}"
    end
  end

  def test_credential_entry_from_raw
    entry = KeySync::CredentialEntry.from_raw("keysync/global", "API_KEY")
    assert_equal "keysync/global", entry.service
    assert_equal "API_KEY", entry.account
    assert_equal "global", entry.scope
    assert_nil entry.project
    assert_nil entry.environment
  end

  def test_credential_entry_from_raw_project
    entry = KeySync::CredentialEntry.from_raw("keysync/project/myapp", "DB_URL")
    assert_equal "keysync/project/myapp", entry.service
    assert_equal "DB_URL", entry.account
    assert_equal "project", entry.scope
    assert_equal "myapp", entry.project
    assert_nil entry.environment
  end

  def test_credential_entry_from_raw_environment
    entry = KeySync::CredentialEntry.from_raw("keysync/project/myapp/env/staging", "DB_URL")
    assert_equal "keysync/project/myapp/env/staging", entry.service
    assert_equal "DB_URL", entry.account
    assert_equal "project", entry.scope
    assert_equal "myapp", entry.project
    assert_equal "staging", entry.environment
  end

  def test_error_hierarchy
    assert KeySync::SecretNotFoundError < KeySync::KeySyncError
    assert KeySync::KeySyncError < StandardError
  end

  def test_macos_methods_exist
    if RUBY_PLATFORM =~ /darwin/
      assert KeySync::MacOS.respond_to?(:get)
      assert KeySync::MacOS.respond_to?(:list)
      assert KeySync::MacOS.respond_to?(:find_attr)
    end
  end

  def test_linux_methods_exist
    if RUBY_PLATFORM =~ /linux/
      assert KeySync::Linux.respond_to?(:get)
      assert KeySync::Linux.respond_to?(:list)
      assert KeySync::Linux.respond_to?(:parse_attr)
    end
  end

  def test_windows_methods_exist
    if RUBY_PLATFORM =~ /mingw|mswin/
      assert KeySync::Windows.respond_to?(:get)
      assert KeySync::Windows.respond_to?(:list)
      assert KeySync::Windows.respond_to?(:service_to_target)
      assert KeySync::Windows.respond_to?(:target_to_service)
      assert KeySync::Windows.respond_to?(:read_cred_ps)
      assert KeySync::Windows.respond_to?(:list_creds_ps)
      assert KeySync::Windows.respond_to?(:run_powershell)
    end
  end

  def test_windows_target_conversion
    # Require the windows module explicitly to test its pure functions
    require_relative "../lib/keysync/windows"
    assert_equal "keysync_global",
      KeySync::Windows.service_to_target("keysync/global")
    assert_equal "keysync_project_myapp",
      KeySync::Windows.service_to_target("keysync/project/myapp")
    assert_equal "keysync_project_my_deep_app",
      KeySync::Windows.service_to_target("keysync/project/my/deep/app")
    assert_equal "keysync_project_myapp_dev",
      KeySync::Windows.service_to_target("keysync/project/myapp/env/dev")
  end

  def test_windows_target_reverse
    require_relative "../lib/keysync/windows"
    assert_equal "keysync/global",
      KeySync::Windows.target_to_service("keysync_global")
    assert_equal "keysync/project/myapp",
      KeySync::Windows.target_to_service("keysync_project_myapp")
    assert_equal "keysync/project/myapp/env/dev",
      KeySync::Windows.target_to_service("keysync_project_myapp_dev")
    assert_nil KeySync::Windows.target_to_service("not_keysync")
  end
end
