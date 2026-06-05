#pragma once

#include <cstdint>

namespace com::teamspeak
{
    class Custom_Device
    {
    public:
        Custom_Device(uint32_t& error);
        ~Custom_Device();

        static constexpr const char* custom_mode = "custom";
        static constexpr const char* custom_device = "loopback";
    };
}
