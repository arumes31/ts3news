cmake_minimum_required(VERSION 3.21)

project("TeamSpeak SDK Samples" LANGUAGES C CXX)

set(CMAKE_SKIP_BUILD_RPATH FALSE)
set(CMAKE_MACOSX_RPATH TRUE)
set(CMAKE_BUILD_WITH_INSTALL_RPATH TRUE)
if (APPLE)
    set(CMAKE_INSTALL_RPATH "@executable_path")
else()
    set(CMAKE_INSTALL_RPATH "\$ORIGIN")
endif()
set(CMAKE_INSTALL_RPATH_USE_LINK_PATH FALSE)
set(BUILD_RPATH_USE_ORIGIN TRUE)

find_package(team_client CONFIG)
find_package(team_server CONFIG)

if(NOT team_client_FOUND AND NOT team_server_FOUND)
    message(FATAL_ERROR
        "Neither team_client nor team_server SDK found via CMAKE_PREFIX_PATH. "
        "Pass -DCMAKE_PREFIX_PATH=... pointing at a stage directory.")
endif()

set(TS_SAMPLES
    client
    client_customdevice
    client_minimal
    client_minimal_filetransfer
    client_cpp_repeater
    server
    server_creation_params
    server_filetransfer
    server_minimal
    server_permissions
)
set(TS_SAMPLE_TYPES client client client client client server server server server server)

set_property(GLOBAL PROPERTY USE_FOLDERS ON)

foreach(sample_folder sample_type IN ZIP_LISTS TS_SAMPLES TS_SAMPLE_TYPES)
    if("${sample_type}" STREQUAL "client" AND NOT team_client_FOUND)
        continue()
    endif()
    if("${sample_type}" STREQUAL "server" AND NOT team_server_FOUND)
        continue()
    endif()
    if("${sample_type}" STREQUAL "server"
       AND CMAKE_SYSTEM_NAME STREQUAL "Linux"
       AND CMAKE_SYSTEM_PROCESSOR STREQUAL "x86")
        continue()
    endif()

    set(ts_sample_bin "ts_${sample_folder}")
    include("${CMAKE_CURRENT_LIST_DIR}/${sample_folder}/sources.cmake")

    add_executable(${ts_sample_bin} ${TS_SAMPLE_SRC})
    set_target_properties(${ts_sample_bin} PROPERTIES CXX_STANDARD 17)
    source_group("" FILES ${TS_SAMPLE_SRC})

    include("${CMAKE_CURRENT_LIST_DIR}/cmake/ide.cmake")
    if(TS_SDK_IDE_FILES)
        target_sources(${ts_sample_bin} PRIVATE ${TS_SDK_IDE_FILES})
        source_group("teamspeak" FILES ${TS_SDK_IDE_FILES})
    endif()

    if("${sample_type}" STREQUAL "client")
        target_link_libraries(${ts_sample_bin} PRIVATE teamspeak::client)
        if(APPLE)
            set_target_properties(${ts_sample_bin} PROPERTIES
                INSTALL_RPATH "@executable_path"
            )
        endif()
        find_package(Threads REQUIRED)
        target_link_libraries(${ts_sample_bin} PRIVATE Threads::Threads)
        set(ts_sdk_target teamspeak::client)
    elseif("${sample_type}" STREQUAL "server")
        target_link_libraries(${ts_sample_bin} PRIVATE teamspeak::server)
        if(APPLE)
            set_target_properties(${ts_sample_bin} PROPERTIES
                INSTALL_RPATH "@executable_path"
            )
        endif()
        set(ts_sdk_target teamspeak::server)
    endif()

    if(CMAKE_SYSTEM_NAME STREQUAL "Linux")
        target_link_libraries(${ts_sample_bin} PRIVATE dl)
    endif()

    # Copy the SDK shared library next to the sample so it runs from the build tree.
    add_custom_command(TARGET ${ts_sample_bin} POST_BUILD
        COMMAND ${CMAKE_COMMAND} -E copy_if_different
            "$<TARGET_FILE:${ts_sdk_target}>"
            "$<TARGET_FILE_DIR:${ts_sample_bin}>"
    )

    install(TARGETS ${ts_sample_bin} RUNTIME DESTINATION bin)
endforeach()

set(_ts_imported_libs "")
if(team_client_FOUND)
    list(APPEND _ts_imported_libs teamspeak::client)
endif()
if(team_server_FOUND)
    list(APPEND _ts_imported_libs teamspeak::server)
endif()
if(_ts_imported_libs)
    install(IMPORTED_RUNTIME_ARTIFACTS ${_ts_imported_libs}
        RUNTIME DESTINATION bin
        LIBRARY DESTINATION bin
    )
endif()
