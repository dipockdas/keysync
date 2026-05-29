"""Tests for keysync Python client."""

import os as _os
from unittest import mock

from keysync import _service_name, _parse_service_name, get_secret, list_secrets
from keysync._errors import KeySyncError, SecretNotFoundError


class TestServiceName:
    def test_global(self):
        assert _service_name("global") == "keysync/global"

    def test_project(self):
        assert _service_name("project", "my-app") == "keysync/project/my-app"

    def test_project_with_environment(self):
        assert _service_name("project", "my-app", "dev") == "keysync/project/my-app/env/dev"

    def test_global_ignores_project(self):
        assert _service_name("global", "my-app") == "keysync/global"

    def test_global_ignores_environment(self):
        assert _service_name("global", environment="dev") == "keysync/global"

    def test_project_no_name(self):
        assert _service_name("project") == "keysync/project"

    def test_project_no_name_with_env(self):
        assert _service_name("project", environment="dev") == "keysync/project"


class TestParseServiceName:
    def test_global(self):
        scope, project, env = _parse_service_name("keysync/global")
        assert scope == "global"
        assert project is None
        assert env is None

    def test_project(self):
        scope, project, env = _parse_service_name("keysync/project/my-app")
        assert scope == "project"
        assert project == "my-app"
        assert env is None

    def test_project_with_environment(self):
        scope, project, env = _parse_service_name("keysync/project/my-app/env/dev")
        assert scope == "project"
        assert project == "my-app"
        assert env == "dev"

    def test_project_with_environment_staging(self):
        scope, project, env = _parse_service_name("keysync/project/myapp/env/staging")
        assert scope == "project"
        assert project == "myapp"
        assert env == "staging"

    def test_project_deep(self):
        scope, project, env = _parse_service_name("keysync/project/my/deep/app")
        assert scope == "project"
        assert project == "my/deep/app"
        assert env is None

    def test_unprefixed(self):
        scope, project, env = _parse_service_name("other/global")
        assert scope == "global"
        assert project is None
        assert env is None

    def test_empty(self):
        scope, project, env = _parse_service_name("")
        assert scope == "global"
        assert project is None
        assert env is None

    def test_just_keysync(self):
        scope, project, env = _parse_service_name("keysync")
        assert scope == "global"
        assert project is None
        assert env is None

    def test_env_in_project_name_not_parsed_as_env(self):
        """"/env/" embedded deeper in the path is not mistaken for env segment."""
        scope, project, env = _parse_service_name("keysync/project/foo/env/bar/baz")
        assert scope == "project"
        assert project == "foo"
        assert env == "bar/baz"


class TestGetSecretResolution:
    """Test get_secret resolution order with mocked platform access."""

    def test_env_var_takes_priority(self):
        """Environment variable is always checked first and returned if set."""
        with mock.patch.dict(_os.environ, {"MY_KEY": "from-env"}, clear=True):
            result = get_secret("MY_KEY", project="myapp", environment="dev")
            assert result == "from-env"

    def test_environment_scope_resolved(self):
        """When environment is provided, that scope is tried before plain project."""
        def mock_platform_get(svc, acct):
            if svc == "keysync/project/myapp/env/dev":
                return "from-env-scope"
            raise SecretNotFoundError(f"{svc}/{acct}")

        with mock.patch.dict(_os.environ, {}, clear=True):
            with mock.patch("keysync._platform_get", mock_platform_get):
                result = get_secret("MY_KEY", project="myapp", environment="dev")
                assert result == "from-env-scope"

    def test_environment_scope_falls_back_to_project(self):
        """When env scope misses, fall back to plain project scope."""
        def mock_platform_get(svc, acct):
            if svc == "keysync/project/myapp/env/dev":
                raise SecretNotFoundError(f"{svc}/{acct}")
            if svc == "keysync/project/myapp":
                return "from-project-scope"
            raise SecretNotFoundError(f"{svc}/{acct}")

        with mock.patch.dict(_os.environ, {}, clear=True):
            with mock.patch("keysync._platform_get", mock_platform_get):
                result = get_secret("MY_KEY", project="myapp", environment="dev")
                assert result == "from-project-scope"

    def test_project_falls_back_to_global(self):
        """When project scope misses, fall back to global scope."""
        def mock_platform_get(svc, acct):
            if svc == "keysync/global":
                return "from-global"
            raise SecretNotFoundError(f"{svc}/{acct}")

        with mock.patch.dict(_os.environ, {}, clear=True):
            with mock.patch("keysync._platform_get", mock_platform_get):
                result = get_secret("MY_KEY", project="myapp")
                assert result == "from-global"

    def test_global_scope_direct(self):
        """Without project, only global scope is checked."""
        def mock_platform_get(svc, acct):
            if svc == "keysync/global":
                return "from-global"
            raise SecretNotFoundError(f"{svc}/{acct}")

        with mock.patch.dict(_os.environ, {}, clear=True):
            with mock.patch("keysync._platform_get", mock_platform_get):
                result = get_secret("MY_KEY")
                assert result == "from-global"

    def test_all_scopes_miss_raises(self):
        """When no scope has the secret, raise SecretNotFoundError."""
        def mock_platform_get(svc, acct):
            raise SecretNotFoundError(f"{svc}/{acct}")

        with mock.patch.dict(_os.environ, {}, clear=True):
            with mock.patch("keysync._platform_get", mock_platform_get):
                try:
                    get_secret("MISSING_KEY", project="myapp", environment="dev")
                except SecretNotFoundError as e:
                    assert e.key == "MISSING_KEY"
                else:
                    assert False, "expected SecretNotFoundError"


class TestListSecretsFiltering:
    """Test list_secrets filtering with environment support."""

    def test_returns_environment_key_when_present(self):
        def mock_platform_list():
            return [
                {"service": "keysync/project/myapp/env/dev", "account": "DB_URL"},
                {"service": "keysync/global", "account": "API_KEY"},
            ]

        with mock.patch("keysync._platform_list", mock_platform_list):
            results = list_secrets()
            assert len(results) == 2
            assert results[0]["scope"] == "project"
            assert results[0]["project"] == "myapp"
            assert results[0]["environment"] == "dev"
            assert results[0]["key"] == "DB_URL"
            assert results[1]["scope"] == "global"
            assert results[1]["project"] is None
            assert "environment" not in results[1]

    def test_filter_by_environment(self):
        def mock_platform_list():
            return [
                {"service": "keysync/project/myapp/env/dev", "account": "DB_URL_DEV"},
                {"service": "keysync/project/myapp/env/prod", "account": "DB_URL_PROD"},
                {"service": "keysync/global", "account": "API_KEY"},
            ]

        with mock.patch("keysync._platform_list", mock_platform_list):
            results = list_secrets(environment="dev")
            assert len(results) == 1
            assert results[0]["key"] == "DB_URL_DEV"
            assert results[0]["environment"] == "dev"

    def test_filter_by_scope_project_environment_combined(self):
        def mock_platform_list():
            return [
                {"service": "keysync/project/myapp/env/dev", "account": "DB_URL"},
                {"service": "keysync/project/myapp/env/prod", "account": "DB_URL"},
                {"service": "keysync/project/other/env/dev", "account": "OTHER_KEY"},
                {"service": "keysync/global", "account": "API_KEY"},
            ]

        with mock.patch("keysync._platform_list", mock_platform_list):
            results = list_secrets(scope="project", project="myapp", environment="dev")
            assert len(results) == 1
            assert results[0]["project"] == "myapp"
            assert results[0]["environment"] == "dev"


class TestErrors:
    def test_secret_not_found(self):
        err = SecretNotFoundError("MY_KEY")
        assert err.code == "notFound"
        assert "MY_KEY" in str(err)

    def test_keychain_error(self):
        err = KeySyncError("keychainError", "something broke")
        assert err.code == "keychainError"
        assert "something broke" in str(err)

    def test_unsupported_platform(self):
        err = KeySyncError("unsupportedPlatform", "bad platform")
        assert err.code == "unsupportedPlatform"
