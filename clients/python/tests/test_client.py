"""Tests for keysync Python client."""

from keysync import _service_name, _parse_service_name
from keysync._errors import KeySyncError, SecretNotFoundError


class TestServiceName:
    def test_global(self):
        assert _service_name("global") == "keysync/global"

    def test_project(self):
        assert _service_name("project", "my-app") == "keysync/project/my-app"

    def test_global_ignores_project(self):
        assert _service_name("global", "my-app") == "keysync/global"

    def test_project_no_name(self):
        assert _service_name("project") == "keysync/project"


class TestParseServiceName:
    def test_global(self):
        scope, project = _parse_service_name("keysync/global")
        assert scope == "global"
        assert project is None

    def test_project(self):
        scope, project = _parse_service_name("keysync/project/my-app")
        assert scope == "project"
        assert project == "my-app"

    def test_project_deep(self):
        scope, project = _parse_service_name("keysync/project/my/deep/app")
        assert scope == "project"
        assert project == "my/deep/app"

    def test_unprefixed(self):
        scope, project = _parse_service_name("other/global")
        assert scope == "global"
        assert project is None

    def test_empty(self):
        scope, project = _parse_service_name("")
        assert scope == "global"
        assert project is None

    def test_just_keysync(self):
        scope, project = _parse_service_name("keysync")
        assert scope == "global"
        assert project is None


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
