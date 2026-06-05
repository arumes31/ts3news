#include "ts_client.hpp"

#include "helpers.hpp"

#include <teamspeak/clientlib.h>
#include <teamspeak/public_errors.h>

#include <algorithm>
#include <cstdint>
#include <cstring>
#include <iostream>
#include <memory>

namespace com::teamspeak
{
    /*static*/ std::unique_ptr<TS_Client> TS_Client::ts_client;

    TS_Client::TS_Client(std::string_view path, bool& success)
    {
        success = true;
        /* Create struct for callback function pointers */
        struct ClientUIFunctions funcs;

        /* Initialize all callbacks with NULL */
        memset(&funcs, 0, sizeof(struct ClientUIFunctions));

        /* Callback function pointers */
        /* It is sufficient to only assign those callback functions you are using. When adding more callbacks, add those function pointers here. */

        /*
        * Callback for connection status change.
        * Connection status switches through the states STATUS_DISCONNECTED, STATUS_CONNECTING, STATUS_CONNECTED and STATUS_CONNECTION_ESTABLISHED.
        *
        * Parameters:
        *   serverConnectionHandlerID - Server connection handler ID
        *   newStatus                 - New connection status, see the enum ConnectStatus in public_definitions.h
        *   errorNumber               - Error code. Should be zero when connecting or actively disconnection.
        *                               Contains error state when losing connection.
        */
        funcs.onConnectStatusChangeEvent = [](uint64_t connection_id, int32_t status, uint32_t error)
        {
            if (TS_Client::ts_client)
                TS_Client::ts_client->on_connect_status_change(connection_id, static_cast<ConnectStatus>(status), error);
        };
        funcs.onClientMoveEvent = [](uint64 connection_id, anyID clientID, uint64 oldChannelID, uint64 newChannelID, int visibility, const char* /*msg*/)
        {
            if (TS_Client::ts_client)
                TS_Client::ts_client->on_client_move_common(connection_id, clientID, oldChannelID, newChannelID, static_cast<Visibility>(visibility));
        };
        funcs.onClientMoveSubscriptionEvent = [](uint64 connection_id, anyID clientID, uint64 oldChannelID, uint64 newChannelID, int visibility)
        {
            if (TS_Client::ts_client)
                TS_Client::ts_client->on_client_move_common(connection_id, clientID, oldChannelID, newChannelID, static_cast<Visibility>(visibility));
        };
        funcs.onClientMoveTimeoutEvent = [](uint64 connection_id, anyID clientID, uint64 oldChannelID, uint64 newChannelID, int visibility, const char* /*msg*/)
        {
            if (TS_Client::ts_client)
                TS_Client::ts_client->on_client_move_common(connection_id, clientID, oldChannelID, newChannelID, static_cast<Visibility>(visibility));
        };
        funcs.onClientMoveMovedEvent = [](uint64 connection_id, anyID clientID, uint64 oldChannelID, uint64 newChannelID, int visibility, anyID /*moverID*/, const char* /*moverName*/, const char* /*moverUniqueIdentifier*/, const char* /*msg*/)
        {
            if (TS_Client::ts_client)
                TS_Client::ts_client->on_client_move_common(connection_id, clientID, oldChannelID, newChannelID, static_cast<Visibility>(visibility));
        };
        funcs.onClientKickFromChannelEvent = [](uint64 connection_id, anyID clientID, uint64 oldChannelID, uint64 newChannelID, int visibility, anyID /*kickerID*/, const char* /*kickerName*/, const char* /*kickerUniqueIdentifier*/, const char* /*msg*/)
        {
            if (TS_Client::ts_client)
                TS_Client::ts_client->on_client_move_common(connection_id, clientID, oldChannelID, newChannelID, static_cast<Visibility>(visibility));
        };
        funcs.onClientKickFromServerEvent = [](uint64 connection_id, anyID clientID, uint64 oldChannelID, uint64 newChannelID, int visibility, anyID /*kickerID */, const char* /*kickerName*/, const char* /*kickerUniqueIdentifier*/, const char* /*msg*/)
        {
            if (TS_Client::ts_client)
                TS_Client::ts_client->on_client_move_common(connection_id, clientID, oldChannelID, newChannelID, static_cast<Visibility>(visibility));
        };
        /*
        * This event is called when a client starts or stops talking.
        *
        * Parameters:
        *   serverConnectionHandlerID - Server connection handler ID
        *   status                    - 1 if client starts talking, 0 if client stops talking
        *   isReceivedWhisper         - 1 if this event was caused by whispering, 0 if caused by normal talking
        *   clientID                  - ID of the client who announced the talk status change
        */
        funcs.onTalkStatusChangeEvent = [](uint64 serverConnectionHandlerID, int status, int isReceivedWhisper, anyID clientID)
        {
            char* name = nullptr;
            /* Query client nickname from ID */
            if (ts3client_getClientVariableAsString(serverConnectionHandlerID, clientID, CLIENT_NICKNAME, &name) != ERROR_ok)
                return;

            auto status_str = std::string();
            switch (status)
            {
            case TalkStatus::STATUS_TALKING:
                status_str = "starts";
                break;
            case TalkStatus::STATUS_NOT_TALKING:
                status_str = "stops";
                break;
            case TalkStatus::STATUS_TALKING_WHILE_DISABLED:
                status_str = "starts (while disabled)";
                break;
            default:
                break;
            }
            std::cout << "Client " << name << " " << status_str << " talking." << std::endl;
            /* Release dynamically allocated memory only if function succeeded */
            ts3client_freeMemory(name);
        };
        funcs.onServerErrorEvent = [](uint64 connection_id, const char* error_msg, uint32_t error, const char* /*return_code*/, const char* extra_msg)
        {
            auto msg = std::string("onServerError: ");
            if (error_msg)
                msg += error_msg;
            
            if (extra_msg)
            {
                auto extra = std::string(extra_msg);
                if (!extra.empty())
                    msg += " Extra Msg: " + extra;
            }
            if (error == ERROR_ok)
                ts3client_logMessage(msg.c_str(), LogLevel::LogLevel_DEBUG, "", connection_id);
            else
                print_error(error, msg, connection_id);
        };
        funcs.onIgnoredWhisperEvent = [](uint64 connection_id, anyID client_id)
        {
            print_error(ts3client_allowWhispersFrom(connection_id, client_id), "Error allowing whisper", connection_id);
        };

        if (auto error = ts3client_initClientLib(&funcs, nullptr, LogType_FILE | LogType_CONSOLE | LogType_USERLOGGING, nullptr, path.data()); error != ERROR_ok)
        {
            print_error(error, "Error initialzing clientlib", 0);
            success = false;
        }
        _funcs = funcs;

        if (success)
        {
            auto error = uint32_t{ ERROR_ok };
            _custom_device = std::make_unique<Custom_Device>(error);
            if (ERROR_ok != error)
                _custom_device = {};

            success = ERROR_ok == error;
        }
    }

    TS_Client::~TS_Client()
    {
        if (auto error = ts3client_destroyClientLib(); error != ERROR_ok)
        {
            print_error(error, "Failed to destroy clientlib", 0);
        }
    }

    /*static*/ bool TS_Client::create(std::string_view path)
    {
        if (TS_Client::ts_client)
            return false;

        auto success = false;
        auto result = std::make_unique<TS_Client>(path, success);
        if (!success)
            return false;

        success = result->log_clientlib_version();
        if (!success)
            return false;

        TS_Client::ts_client.swap(result);
        return true;
    }

    bool TS_Client::log_clientlib_version()
    {
        char* version = nullptr;
        if (auto error = ts3client_getClientLibVersion(&version); error != ERROR_ok)
        {
            print_error(error, "Failed to get clientlib version", 0);
            return false;
        }
        auto msg = "Client lib version: " + std::string(version);
        ts3client_freeMemory(version);  /* Release dynamically allocated memory */
        version = nullptr;
        ts3client_logMessage(msg.c_str(), LogLevel_INFO, "", 0);
        return true;
    }

    void TS_Client::on_client_move_common(uint64_t connection_id, uint16_t client_id, uint64_t oldChannelID, uint64_t newChannelID, Visibility visibility)
    {
    }

    void TS_Client::on_connect_status_change(uint64_t connection_id, ConnectStatus status, uint32_t error)
    {
        {
            auto msg = std::string("Connect status changed: ") + std::to_string(connection_id) + " " + std::to_string(status);
            ts3client_logMessage(msg.c_str(), LogLevel_INFO, "", connection_id);
        }
        /* Failed to connect ? */
        if (status == STATUS_DISCONNECTED && error == ERROR_failed_connection_initialisation)
        {
            ts3client_logMessage("Looks like there is no server running.\n", LogLevel_INFO, "", connection_id);
        }
        print_error(error, "onConnectStatusChange", connection_id);

        if (_shutting_down)
            return;

        /* pass the event on to the connection instance */
        if (auto it = std::find_if(std::begin(_connections), std::end(_connections), [connection_id](auto&& connection)
            {
                return connection && connection_id == connection->_connection_id;
            }); it != std::end(_connections))
        {
            auto&& connection = *it;
            connection->on_connect_status_change(status, error);
        }
    }
}
