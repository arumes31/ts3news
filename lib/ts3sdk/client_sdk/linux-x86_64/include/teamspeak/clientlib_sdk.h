#ifndef CLIENTLIB_SDK_H
#define CLIENTLIB_SDK_H

// system
#include <stdlib.h>

// own
#include "teamspeak/public_definitions.h"

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief Sets the client to which to transmit voice. Stops standard channel voice transmission.
 *
 * The client will still receive voice from their current channel, however their voice will not be transmitted to their
 * current channel anymore. If this call is successful (check onServerErrorEvent) then voice of the specified client
 * will be transmitted to all specified channels and all the specified clients. Pass 0 to both target parameter arrays
 * to restore default behavior of transmitting voice to current channel. You will receive an onServerErrorEvent with the
 * passed returnCode indicating whether or not the operation was successful.
 *
 * @param server_connection_handler_id the connection handler on which to set the whisper list
 * @param client_id the client to set the whisper list for. Set to 0 or your own client ID to set your own whisper list.
 * @param channel_ids an array of channel ids to transmit voice to.
 * @param channel_ids_size number of elements in aforementioned array.
 * @param client_ids a zero terminated array of client ids to transmit voice to.
 * @param client_ids_size number of elements in aforementioned array.
 * @param impersonate if the target client is a webrtc client, the voice packets will look like as if they have been
 * send by the invoking client id
 * @param return_code a c string to identify this request in callbacks. Pass an empty string if unused.
 * @return An error code from the @ref Ts3ErrorType enum indicating either success or the failure reason
 */
EXPORTDLL unsigned int ts_client_request_client_set_whisper_list(uint64 server_connection_handler_id, anyID client_id,
                                                                 const uint64* channel_ids,
                                                                 int           channel_ids_size,
                                                                 const anyID*  client_ids,
                                                                 int client_ids_size, int impersonate,
                                                                 const char* return_code);

/**
 * @brief Send a binary-serialized ClientCommandRequest protobuf to the client library.
 *
 * The response will be delivered asynchronously via the onProtoResponse callback
 * as a serialized ClientCommandResponse protobuf.
 *
 * @param data Pointer to serialized ClientCommandRequest protobuf bytes
 * @param size Size of the serialized data in bytes
 * @param return_code Caller-provided string to correlate the response in onProtoResponse. May be NULL.
 * @return An error code: ERROR_ok on successful dispatch, ERROR_parameter_invalid on parse failure
 */
EXPORTDLL unsigned int ts3client_postProtoCommand(const void* data, size_t size, const char* return_code);

#ifdef __cplusplus
}
#endif

#endif  // CLIENTLIB_SDK_H
