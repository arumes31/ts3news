message("generating TeamSpeak SDK sample ${TS_SDK_SAMPLE}")

set (TS_SAMPLE_SRC
    "${CMAKE_CURRENT_LIST_DIR}/main.cpp"
    "${CMAKE_CURRENT_LIST_DIR}/ts_client.hpp"
    "${CMAKE_CURRENT_LIST_DIR}/ts_client.cpp"
    "${CMAKE_CURRENT_LIST_DIR}/connection_handler.hpp"
    "${CMAKE_CURRENT_LIST_DIR}/connection_handler.cpp"
    "${CMAKE_CURRENT_LIST_DIR}/helpers.hpp"
    "${CMAKE_CURRENT_LIST_DIR}/helpers.cpp"
    "${CMAKE_CURRENT_LIST_DIR}/custom_device.hpp"
    "${CMAKE_CURRENT_LIST_DIR}/custom_device.cpp"
)
