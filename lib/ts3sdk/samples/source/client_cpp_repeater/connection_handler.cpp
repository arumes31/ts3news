#include "connection_handler.hpp"

#include "ts_client.hpp"

namespace com::teamspeak
{
    /* We'll be using the create() function instead */
    Connection_Handler::Connection_Handler(uint64_t connection_id)
        : _connection_id(connection_id)
    {}

    Connection_Handler::~Connection_Handler()
    {
        if (auto error = ts3client_destroyServerConnectionHandler(_connection_id); error != ERROR_ok)
            print_error(error, "Error destroying connection", _connection_id);
    }

    /*static*/ std::unique_ptr<Connection_Handler> Connection_Handler::create()
    {
        auto connection_id = uint64_t{ 0 };
        if (auto error = ts3client_spawnNewServerConnectionHandler(0, &connection_id); error != ERROR_ok)
        {
            print_error(error, "Error spawning server connection handler", 0);
            return {};
        }
        return std::make_unique<Connection_Handler>(connection_id);
    }

    uint32_t Connection_Handler::connect()
    {
        /* Connect to server on localhost:9987 with nickname "client", no default channel, no default channel password and server password "secret" */
        if (auto error = ts3client_startConnection(
            _connection_id,
            _connection_data.identity.c_str(),
            _connection_data.address.c_str(),
            _connection_data.port,
            _connection_data.nick.c_str(),
            nullptr,
            "",
            _connection_data.pw.c_str());
            error != ERROR_ok)
        {
            print_error(error, "Error connecting to server", _connection_id);
            return error;
        }
        return ERROR_ok;
    }

    uint32_t Connection_Handler::disconnect(std::string_view reason)
    {
        if (auto error = ts3client_stopConnection(_connection_id, reason.data()); error != ERROR_ok)
        {
            printf("Error stopping connection: %d\n", error);
            return error;
        }
        return ERROR_ok;
    }

    void Connection_Handler::on_connect_status_change(ConnectStatus status, uint32_t error)
    {
        if (TS_Client::_do_autoreconnect && ConnectStatus::STATUS_DISCONNECTED == status)
        {
            // TODO: maybe should be timer-delayed?
            connect();
        }
    }

    uint32_t Connection_Handler::open_audio(Audio_IO audio_io, std::string_view mode, std::string_view device_id)
    {
        if (Audio_IO::Capture == audio_io)
        {
            if (auto error = ts3client_openPlaybackDevice(_connection_id, mode.data(), device_id.data()); error != ERROR_ok)
            {
                print_error(error, "Error opening playback device.", _connection_id);
                return error;
            }
        }
        else if (Audio_IO::Playback == audio_io)
        {
            if (auto error = ts3client_openCaptureDevice(_connection_id, mode.data(), device_id.data()); error != ERROR_ok)
            {
                print_error(error, "Error opening capture device.", _connection_id);
                return error;
            }

            // Turn off any DSP, except a very low power based Voice Activity Detection

            /* Adjust "vad_mode" value to use power */
            print_error(
                ts3client_setPreProcessorConfigValue(_connection_id, "vad_mode", "1"),
                "Error setting vad_mode value to hybrid.", _connection_id);

            /* Adjust "voiceactivation_level" value */
            print_error(
                ts3client_setPreProcessorConfigValue(_connection_id, "voiceactivation_level", "-50"),
                "Error setting voiceactivation_level.", _connection_id);

            /* turn on vad */
            print_error(
                ts3client_setPreProcessorConfigValue(_connection_id, "vad", "true"),
                "Couldn't turn on VAD.", _connection_id);

            // TODO: Turn off denoiser etc. no matter the default value
        }
        return ERROR_ok;
    }
}
