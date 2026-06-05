/*
 * TeamSpeak SDK client repeater sample
 *
 * Copyright (c) TeamSpeak-Systems
 */

#ifdef _WIN32
#define _CRT_SECURE_NO_WARNINGS
#include <Windows.h>
#else
#include <unistd.h>
#include <stdlib.h>
#include <string.h>
#endif
#include <stdio.h>

#include "custom_device.hpp"
#include "helpers.hpp"
#include "ts_client.hpp"

#include <teamspeak/public_definitions.h>
#include <teamspeak/public_errors.h>
#include <teamspeak/clientlib.h>

#include <chrono>
#include <iostream>
#include <thread>
#include <string>
#include <vector>

#ifdef _WIN32
#define SLEEP(x) Sleep(x)
#define strdup(x) _strdup(x)
#else
#define SLEEP(x) usleep(x*1000)
#endif

struct Opts {
    std::string from_ip = "";
    uint16_t from_port = 0;
    std::string to_ip = "";
    uint16_t to_port = 0;
};

namespace {
    char* programPath(char* programInvocation)
    {
        char* path;
        char* end;
        int length;
        char pathsep;

        if (programInvocation == NULL) return strdup("");

#ifdef _WIN32
        pathsep = '\\';
#else
        pathsep = '/';
#endif

        end = strrchr(programInvocation, pathsep);
        if (!end) return strdup("");

        length = (end - programInvocation) + 2;
        path = (char*)malloc(length);
        strncpy(path, programInvocation, length - 1);
        path[length - 1] = 0;

        return path;
    }

    void print_usage()
    {
        std::cout << "usage: from_id from_port to_id to_port" << std::endl;
    }
}

int main(int argc, char** argv)
{
    // TODO: Decide on a proper header only options parser
    auto opts = Opts();
    if (argc != 5)
    {
        print_usage();
        return -1;
    }
    for (auto i = decltype(argc){1}; i < argc; ++i)
    {
        switch (i)
        {
        case 1:
            opts.from_ip = std::string(argv[i]);
            break;
        case 2:
            try
            {
                opts.from_port = std::stoul(argv[i]);
            }
            catch (std::exception& e)
            {
                print_usage();
                return -1;
            }
            break;
        case 3:
            opts.to_ip = std::string(argv[i]);
            break;
        case 4:
            try
            {
                opts.to_port = std::stoul(argv[i]);
            }
            catch (std::exception& e)
            {
                print_usage();
                return -1;
            }
            break;
        }
    }
    std::cout << "listening to " << opts.from_ip.c_str() << ":" << opts.from_port << ", sending to " << opts.to_ip.c_str() << ":" << opts.to_port << std::endl;

    using namespace com::teamspeak;

    {
        auto* path = programPath(argv[0]);
        auto success = TS_Client::create(path);
        free(path);
        if (!success)
            return 1;
    }
    auto&& ts_client = TS_Client::ts_client;
    {
        auto identity = create_identity();
        if (identity.empty())
            return 1;

        ts_client->_identity = identity;
    }

    // We'll recycle them in case of disconnect, hence spawn these only once
    {
        auto connection = Connection_Handler::create();
        if (!connection)
            return 1;

        ts_client->_connections[Audio_IO::Playback].swap(connection);
    }

    {
        auto connection = Connection_Handler::create();
        if (!connection)
            return 1;

        ts_client->_connections[Audio_IO::Capture].swap(connection);
    }


    auto&& connection_listen = ts_client->_connections[Audio_IO::Playback];
    auto&& connection_broadcast = ts_client->_connections[Audio_IO::Capture];

    connection_listen->open_audio(Audio_IO::Playback, Custom_Device::custom_mode, Custom_Device::custom_device);
    connection_broadcast->open_audio(Audio_IO::Capture, Custom_Device::custom_mode, Custom_Device::custom_device);

    connection_listen->_connection_data = Connection_Handler::Connection_Data{
        opts.from_ip,
        opts.from_port,
        "repeater-listener",
        ts_client->_identity
    };
    connection_listen->connect();
    connection_broadcast->_connection_data = Connection_Handler::Connection_Data{
        opts.to_ip,
        opts.to_port,
        "repeater-broadcaster",
        ts_client->_identity
    };
    connection_broadcast->connect();

    // 48000 kHz, 1ch, 20ms -> 960 samples
    auto playback_buffer = std::array<int16_t, 960>();
    auto custom_audio_thread = std::thread([&playback_buffer]()
        {
            while (!TS_Client::ts_client->_shutting_down)
            {
                for (;;)
                {
                    /* Get playback data from the client lib */
                    if (auto error_playback = ts3client_acquireCustomPlaybackData(Custom_Device::custom_device, playback_buffer.data(), playback_buffer.size()); error_playback != ERROR_ok)
                    {
                        if (ERROR_sound_no_data == error_playback)
                        {
                            /* Not an error. The client lib has no playback data available.
                            Depending on your custom sound API, either pause playback for
                            performance optimization or send a buffer of zeros. */

                            // we're doing a no-op here
                        }
                        else
                        {
                            /* Error occured */
                            print_error(error_playback, "Failed to get playback data", 0);
                        }
                        break; // break draining (inner loop)
                    }
                    else
                    {
                        // we got playback data, loop it back to capture
                        /* Stream your capture data to the client lib */
                        if (auto error_capture = ts3client_processCustomCaptureData(Custom_Device::custom_device, playback_buffer.data(), playback_buffer.size()); ERROR_ok != error_capture)
                        {
                            print_error(error_capture, "Failed to process capture data", 0);
                            break; // break draining (inner loop)
                        }
                    }
                }
                using namespace std::chrono_literals;
                std::this_thread::sleep_for(0.02s); // audio buffer size in our opus is 20ms
            }
        });

    SLEEP(500);

    /* Wait for user input */
    printf("\n--- Press Return to disconnect from server and exit ---\n");
    getchar();

    /* Disconnect from servers */
    ts_client->_shutting_down = true;
    custom_audio_thread.join();
    connection_listen->disconnect();
    connection_broadcast->disconnect();
    return 0;
}
