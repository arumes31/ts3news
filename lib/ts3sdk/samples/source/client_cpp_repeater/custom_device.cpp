#include "custom_device.hpp"

#include "helpers.hpp"

#include <teamspeak/clientlib.h>
#include <teamspeak/public_errors.h>

namespace com::teamspeak
{
    Custom_Device::Custom_Device(uint32_t& error)
    {
        error = ts3client_registerCustomDevice(custom_device, custom_device, 48000, 1, 48000, 1);
        if (error != ERROR_ok)
        {
            print_error(error, "Error creating custom device.", 0);
        }
    }

    Custom_Device::~Custom_Device()
    {
        /* Unregister the custom device. This automatically closes the device.*/
        if (auto error = ts3client_unregisterCustomDevice(custom_device); error != ERROR_ok)
        {
            printf("Error unregistering custom device: %d\n", error);
        }
    }
}
