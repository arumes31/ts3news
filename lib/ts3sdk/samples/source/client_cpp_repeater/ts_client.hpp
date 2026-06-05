#pragma once

#include "connection_handler.hpp"
#include "custom_device.hpp"

#include <teamspeak/public_definitions.h>

#include <array>
#include <cstdint>
#include <memory>
#include <string>
#include <string_view>

namespace com::teamspeak
{
    class TS_Client
    {
    public:
        TS_Client(std::string_view path, bool& success);
        ~TS_Client();

        static bool create(std::string_view path);

        bool log_clientlib_version();

        void on_client_move_common(uint64_t connection_id, uint16_t client_id, uint64_t old_channel_id, uint64_t new_channel_id, Visibility visibility);
        void on_connect_status_change(uint64_t connection_id, ConnectStatus status, uint32_t error);

        ClientUIFunctions _funcs;
        std::string _identity = "";
        std::array<std::unique_ptr<Connection_Handler>, 2> _connections;
        bool _shutting_down = false;
        static constexpr bool _do_autoreconnect{ true };
    private:
        std::unique_ptr<Custom_Device> _custom_device;
    public:
        static std::unique_ptr<TS_Client> ts_client;
    };
}