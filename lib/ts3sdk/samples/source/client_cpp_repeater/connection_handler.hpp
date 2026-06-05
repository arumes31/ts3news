#pragma once

#include "helpers.hpp"

#include <teamspeak/clientlib.h>
#include <teamspeak/public_definitions.h>
#include <teamspeak/public_errors.h>

#include <cstdint>
#include <memory>
#include <string_view>

namespace com::teamspeak
{
class Connection_Handler
{
public:
    /* We'll be using the create() function instead */
    Connection_Handler(uint64_t connection_id);
    Connection_Handler() = delete;
    ~Connection_Handler();

    static std::unique_ptr<Connection_Handler> create();

    uint32_t connect();
    uint32_t disconnect(std::string_view reason = "leaving");

    uint32_t open_audio(Audio_IO audio_io, std::string_view mode, std::string_view device_id);

    void on_connect_status_change(ConnectStatus status, uint32_t error);

    struct Connection_Data
    {
        std::string address = "";
        uint16_t port = 9987;
        std::string nick = "";
        std::string identity = "";
        std::string pw = "";
    };
    Connection_Data _connection_data;
    const uint64_t _connection_id;
};

}
