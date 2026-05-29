#ifndef KEYSYNC_ERRORS_HPP
#define KEYSYNC_ERRORS_HPP

#include <stdexcept>
#include <string>

namespace keysync {

/// Error codes for keysync operations.
enum class ErrorCode {
    NotFound,            // The requested secret was not found in any scope.
    KeychainError,       // An OS-level keychain error occurred.
    UnsupportedPlatform, // The platform is not supported for keychain access.
};

/// Exception thrown by keysync operations.
class KeySyncError : public std::runtime_error {
public:
    KeySyncError(ErrorCode code, const std::string& message);
    KeySyncError(ErrorCode code, const char* message);

    /// The error code identifying the failure type.
    ErrorCode code() const noexcept { return code_; }

private:
    ErrorCode code_;
};

} // namespace keysync

#endif // KEYSYNC_ERRORS_HPP
