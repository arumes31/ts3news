#pragma once

#include <cstdint>
#include <string>

namespace com::teamspeak {

enum Audio_IO : uint8_t
{
    Playback = 0,
    Capture
};

void print_error(uint32_t error, const std::string& msg, uint64_t connection_id = 0);
auto create_identity()->std::string;

}
