# frozen_string_literal: true

require_relative "test_helper"

class TestErrors < Minitest::Test
  def test_secret_not_found
    err = KeySync::SecretNotFoundError.new("MY_KEY")
    assert_equal "not_found", err.code
    assert_equal "MY_KEY", err.key
    assert_includes err.message, "MY_KEY"
    assert_kind_of KeySync::KeySyncError, err
    assert_kind_of StandardError, err
  end

  def test_keychain_error
    err = KeySync::KeySyncError.new("keychain_error", "something broke")
    assert_equal "keychain_error", err.code
    assert_includes err.message, "something broke"
  end

  def test_unsupported_platform
    err = KeySync::KeySyncError.new("unsupported_platform", "bad platform")
    assert_equal "unsupported_platform", err.code
    assert_includes err.message, "bad platform"
  end

  def test_keychain_error_with_symbol_code
    err = KeySync::KeySyncError.new(:keychain_error, "broke")
    assert_equal "keychain_error", err.code
  end
end
