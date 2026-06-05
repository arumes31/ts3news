# Collect a small set of SDK headers to show in the IDE solution.
# Depends on find_package(team_client|team_server) having run.

if("${sample_type}" STREQUAL "client")
    get_target_property(_ts_sdk_inc_dir teamspeak::client INTERFACE_INCLUDE_DIRECTORIES)
elseif("${sample_type}" STREQUAL "server")
    get_target_property(_ts_sdk_inc_dir teamspeak::server INTERFACE_INCLUDE_DIRECTORIES)
else()
    set(_ts_sdk_inc_dir "")
endif()

set(TS_SDK_IDE_FILES "")
if(_ts_sdk_inc_dir)
    file(GLOB _ts_log_headers "${_ts_sdk_inc_dir}/teamlog/*.h")
    list(APPEND TS_SDK_IDE_FILES
        ${_ts_log_headers}
        "${_ts_sdk_inc_dir}/teamspeak/public_definitions.h"
        "${_ts_sdk_inc_dir}/teamspeak/public_errors.h"
    )
    if("${sample_type}" STREQUAL "client")
        list(APPEND TS_SDK_IDE_FILES "${_ts_sdk_inc_dir}/teamspeak/clientlib.h")
    elseif("${sample_type}" STREQUAL "server")
        list(APPEND TS_SDK_IDE_FILES
            "${_ts_sdk_inc_dir}/teamspeak/server_commands_file_transfer.h"
            "${_ts_sdk_inc_dir}/teamspeak/serverlib.h"
            "${_ts_sdk_inc_dir}/teamspeak/serverlib_publicdefinitions.h"
        )
    endif()
endif()
