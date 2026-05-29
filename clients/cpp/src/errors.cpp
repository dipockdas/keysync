#include "keysync/errors.hpp"

namespace keysync {

KeySyncError::KeySyncError(ErrorCode code, const std::string& message)
    : std::runtime_error(message), code_(code) {}

KeySyncError::KeySyncError(ErrorCode code, const char* message)
    : std::runtime_error(message), code_(code) {}

} // namespace keysync
