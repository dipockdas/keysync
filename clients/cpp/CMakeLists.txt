cmake_minimum_required(VERSION 3.15)
project(keysync VERSION 0.1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# ---------------------------------------------------------------------------
# Library target
# ---------------------------------------------------------------------------
add_library(keysync STATIC
    src/service.cpp
    src/errors.cpp
    src/keysync.cpp
)

target_include_directories(keysync PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:include>
)

# Platform-specific sources
if(APPLE)
    target_sources(keysync PRIVATE src/macos.cpp)
    target_link_libraries(keysync PRIVATE "-framework Security" "-framework Foundation")
elseif(UNIX AND NOT APPLE)
    target_sources(keysync PRIVATE src/linux.cpp)
    # Linux uses popen() to call secret-tool CLI — no library needed.
    # libsecret-1 is a runtime dependency of the secret-tool command.
elseif(WIN32)
    target_sources(keysync PRIVATE src/windows.cpp)
    target_link_libraries(keysync PRIVATE advapi32)
endif()

# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------
option(BUILD_TESTS "Build the test suite" ON)

if(BUILD_TESTS)
    enable_testing()
    add_subdirectory(tests)
endif()
