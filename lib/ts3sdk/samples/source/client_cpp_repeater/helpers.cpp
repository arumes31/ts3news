#include "helpers.hpp"

#include <teamspeak/clientlib.h>
#include <teamspeak/public_errors.h>

#ifdef _WIN32
#define _CRT_SECURE_NO_WARNINGS
#include <Windows.h>
#else
#include <unistd.h>
#include <stdlib.h>
#include <string.h>
#endif

namespace com::teamspeak {
    void print_error(uint32_t error, const std::string& msg, uint64_t connection_id)
    {
        if (error == ERROR_ok)
            return;

        char* errormsg = nullptr;
        if (ts3client_getErrorMessage(error, &errormsg) == ERROR_ok)
        {
            auto error_msg = msg + " " + std::string(errormsg);
            ts3client_freeMemory(errormsg);
            ts3client_logMessage(error_msg.c_str(), LogLevel_ERROR, "", connection_id);
            return;
        }
        ts3client_logMessage(msg.c_str(), LogLevel_ERROR, "", connection_id);
    }

    auto create_identity() -> std::string
    {
        /* Create a new client identity */
        /* In your real application you should do this only once, store the assigned identity locally and then reuse it. */
        auto result = std::string();
        char* identity = nullptr;
        if (auto error = ts3client_createIdentity(&identity); error != ERROR_ok)
        {
            print_error(error, "Error creating identity", 0);
        }
        else
        {
            result = std::string(identity);
            ts3client_freeMemory(identity);  /* Release dynamically allocated memory */
            identity = nullptr;
        }
        return result;
    }
}
