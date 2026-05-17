/// Windows implementation: uses the Win32 Credential Manager API directly.
///
/// Target name format replaces slashes with underscores:
///   "keysync/global"         → "keysync_global"
///   "keysync/project/my-app" → "keysync_project_my-app"
///
/// The credential's UserName field holds the secret key name.

#ifdef _WIN32

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#ifndef NOMINMAX
#define NOMINMAX
#endif

#include <windows.h>
#include <wincred.h>

#include "internal_helpers.hpp"
#include "keysync/errors.hpp"
#include "keysync/credential.hpp"

#include <string>
#include <string_view>
#include <vector>
#include <memory>

namespace keysync {
namespace internal {

namespace {

/// Convert a UTF-8 std::string to a wide string (for Win32 API).
std::wstring toWide(std::string_view s) {
    if (s.empty()) {
        return L"";
    }
    int len = MultiByteToWideChar(CP_UTF8, 0, s.data(), static_cast<int>(s.size()), nullptr, 0);
    std::wstring result(len, L'\0');
    MultiByteToWideChar(CP_UTF8, 0, s.data(), static_cast<int>(s.size()), &result[0], len);
    return result;
}

/// Convert a wide string to UTF-8 std::string.
std::string fromWide(const wchar_t* ws) {
    if (!ws) {
        return "";
    }
    int len = WideCharToMultiByte(CP_UTF8, 0, ws, -1, nullptr, 0, nullptr, nullptr);
    std::string result(len, '\0');
    WideCharToMultiByte(CP_UTF8, 0, ws, -1, &result[0], len, nullptr, nullptr);
    // Remove trailing null terminator added by -1 length
    if (!result.empty() && result.back() == '\0') {
        result.pop_back();
    }
    return result;
}

/// Convert a wide char pointer to a std::wstring.
std::wstring wcharToWstring(const wchar_t* ws) {
    return ws ? std::wstring(ws) : std::wstring();
}

/// Convert a CredentialBlob (byte array) to a UTF-8 string.
/// Windows stores credential blobs as UTF-16LE.
std::string blobToString(const PBYTE blob, DWORD blobSize) {
    if (!blob || blobSize == 0) {
        return "";
    }

    // The blob is stored as UTF-16LE wchar_t array
    size_t wcharCount = blobSize / sizeof(wchar_t);
    std::wstring wstr(reinterpret_cast<const wchar_t*>(blob), wcharCount);

    // Trim trailing null characters
    while (!wstr.empty() && wstr.back() == L'\0') {
        wstr.pop_back();
    }

    if (wstr.empty()) {
        return "";
    }

    int len = WideCharToMultiByte(CP_UTF8, 0, wstr.c_str(), static_cast<int>(wstr.size()),
                                   nullptr, 0, nullptr, nullptr);
    if (len <= 0) {
        return "";
    }
    std::string result(len, '\0');
    WideCharToMultiByte(CP_UTF8, 0, wstr.c_str(), static_cast<int>(wstr.size()),
                         &result[0], len, nullptr, nullptr);
    return result;
}

/// RAII wrapper for CredFree.
struct CredFreeDeleter {
    void operator()(void* p) const { CredFree(p); }
};

} // anonymous namespace

std::string getSecretWindows(const std::string& service, const std::string& account) {
    std::string target = serviceToTarget(service);
    std::wstring wtarget = toWide(target);
    std::wstring waccount = toWide(account);

    PCREDENTIALW pCred = nullptr;
    if (!CredReadW(wtarget.c_str(), CRED_TYPE_GENERIC, 0, &pCred)) {
        DWORD err = GetLastError();
        if (err == ERROR_NOT_FOUND) {
            throw KeySyncError(ErrorCode::NotFound,
                "secret not found: " + service + "/" + account);
        }
        throw KeySyncError(ErrorCode::KeychainError,
            "CredReadW failed with error code: " + std::to_string(err));
    }

    std::unique_ptr<void, CredFreeDeleter> credGuard(pCred);

    // Verify the credential's UserName matches the requested account
    std::wstring storedUser = wcharToWstring(pCred->UserName);
    if (storedUser != waccount) {
        throw KeySyncError(ErrorCode::NotFound,
            "secret not found (account mismatch): " + service + "/" + account);
    }

    return blobToString(pCred->CredentialBlob, pCred->CredentialBlobSize);
}

std::vector<CredentialEntry> listSecretsWindows(const std::string& scope_filter,
                                                  const std::string& project_filter) {
    std::vector<CredentialEntry> entries;

    DWORD count = 0;
    PCREDENTIALW* pCredentials = nullptr;

    std::wstring filter = L"keysync_*";

    if (!CredEnumerateW(filter.c_str(), 0, &count, &pCredentials)) {
        return entries;
    }

    std::unique_ptr<void, CredFreeDeleter> credGuard(pCredentials);

    for (DWORD i = 0; i < count; ++i) {
        PCREDENTIALW cred = pCredentials[i];
        if (!cred) continue;

        std::wstring wtarget = wcharToWstring(cred->TargetName);
        if (wtarget.size() < 8 || wtarget.substr(0, 8) != L"keysync_") {
            continue;
        }

        std::string target = fromWide(wtarget.c_str());
        std::string svc = targetToService(target);

        std::string entryScope, entryProject;
        parseServiceName(svc, entryScope, entryProject);

        bool scopeMatch = scope_filter.empty() || entryScope == scope_filter;
        bool projectMatch = project_filter.empty() || entryProject == project_filter;

        if (scopeMatch && projectMatch) {
            std::string userName = fromWide(cred->UserName);
            if (!userName.empty()) {
                entries.push_back({entryScope, entryProject, userName});
            }
        }
    }

    return entries;
}

bool isNotFoundWindows(const std::string& service, const std::string& account) {
    std::string target = serviceToTarget(service);
    std::wstring wtarget = toWide(target);
    std::wstring waccount = toWide(account);

    PCREDENTIALW pCred = nullptr;
    if (!CredReadW(wtarget.c_str(), CRED_TYPE_GENERIC, 0, &pCred)) {
        return true; // Not found
    }

    std::unique_ptr<void, CredFreeDeleter> credGuard(pCred);

    std::wstring storedUser = wcharToWstring(pCred->UserName);
    if (storedUser != waccount) {
        return true; // Account mismatch = not found
    }

    return false;
}

} // namespace internal
} // namespace keysync

#endif // _WIN32
