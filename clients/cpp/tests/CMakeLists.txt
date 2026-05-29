cmake_minimum_required(VERSION 3.15)

add_executable(keysync_tests
    test_main.cpp
    test_service.cpp
)

target_include_directories(keysync_tests PRIVATE
    ${CMAKE_SOURCE_DIR}/include
    ${CMAKE_SOURCE_DIR}/src
)

target_link_libraries(keysync_tests PRIVATE keysync)

# Register the test with CTest
add_test(NAME keysync_tests COMMAND keysync_tests)
