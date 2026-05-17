# frozen_string_literal: true

require_relative "test_helper"

class TestService < Minitest::Test
  def test_global_service_name
    assert_equal "keysync/global", KeySync::Service.service_name("global")
  end

  def test_project_service_name
    assert_equal "keysync/project/my-app",
      KeySync::Service.service_name("project", "my-app")
  end

  def test_global_ignores_project
    assert_equal "keysync/global",
      KeySync::Service.service_name("global", "my-app")
  end

  def test_project_without_name
    assert_equal "keysync/project",
      KeySync::Service.service_name("project")
  end

  def test_parse_global
    scope, project = KeySync::Service.parse_service_name("keysync/global")
    assert_equal "global", scope
    assert_nil project
  end

  def test_parse_project
    scope, project = KeySync::Service.parse_service_name("keysync/project/my-app")
    assert_equal "project", scope
    assert_equal "my-app", project
  end

  def test_parse_project_deep
    scope, project = KeySync::Service.parse_service_name("keysync/project/my/deep/app")
    assert_equal "project", scope
    assert_equal "my/deep/app", project
  end

  def test_parse_unprefixed
    scope, project = KeySync::Service.parse_service_name("other/global")
    assert_equal "global", scope
    assert_nil project
  end

  def test_parse_empty
    scope, project = KeySync::Service.parse_service_name("")
    assert_equal "global", scope
    assert_nil project
  end

  def test_parse_just_keysync
    scope, project = KeySync::Service.parse_service_name("keysync")
    assert_equal "global", scope
    assert_nil project
  end
end
