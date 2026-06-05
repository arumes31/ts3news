#ifndef TS_SESSION_SUBSCRIBER_PROTOCOL_H_
#define TS_SESSION_SUBSCRIBER_PROTOCOL_H_
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif
// Version definitions.
#define TS_SESSION_SUBSCRIBER_COMMAND_VERSION_1 1


// Protocol used for communication between the session subscriber and the session (a single stream).
// Commands are sent bidirectionally or unidirectionally between the parties.
// e.g. SESSION => SUBSCRIBER: ON_FRAME, ON_PAUSED_CHANGED
// e.g. SUBSCRIBER => SESSION: ON_FRAME_ACK, ON_RESIZE


// Command type enum with a lowercase type name.
typedef enum
{
  TS_SESSION_SUBSCRIBER_CMD_ON_FRAME = 0,
  TS_SESSION_SUBSCRIBER_CMD_ON_RESIZE,
  TS_SESSION_SUBSCRIBER_CMD_ON_PAUSED_CHANGED,
  TS_SESSION_SUBSCRIBER_CMD_ON_FRAME_ACK,
  TS_SESSION_SUBSCRIBER_CMD_ON_BUFFER_SINGLE,
  TS_SESSION_SUBSCRIBER_CMD_ON_BUFFER_FRONT_BACK,
  TS_SESSION_SUBSCRIBER_CMD_ON_RELEASE_BUFFER,
  TS_SESSION_SUBSCRIBER_CMD_ADD_SUBSCRIBER,
  TS_SESSION_SUBSCRIBER_CMD_ADD_SUBSCRIBER_RESP,
  TS_SESSION_SUBSCRIBER_CMD_REMOVE_SUBSCRIBER,
  TS_SESSION_SUBSCRIBER_CMD_ON_AUDIO_FRAMES,
  TS_SESSION_SUBSCRIBER_CMD_ON_AUDIO_ENABLED_CHANGED,
} ts_session_subscriber_command_type_t;

// Pixel format enum.
typedef enum
{
  TS_SESSION_SUBSCRIBER_PIXEL_FORMAT_I420 = 0,
  TS_SESSION_SUBSCRIBER_PIXEL_FORMAT_NV12,
  TS_SESSION_SUBSCRIBER_PIXEL_FORMAT_ABGR,
  TS_SESSION_SUBSCRIBER_PIXEL_FORMAT_ARGB,
} ts_session_subscriber_pixel_format_t;

// Buffer location enum.
typedef enum
{
  TS_SESSION_SUBSCRIBER_BUFFER_LOCATION_CPU = 0,
  TS_SESSION_SUBSCRIBER_BUFFER_LOCATION_GPU
} ts_session_subscriber_buffer_location_t;

// Buffer type enum.
typedef enum
{
  TS_SESSION_SUBSCRIBER_BUFFER_TYPE_SINGLE = 0,
  TS_SESSION_SUBSCRIBER_BUFFER_TYPE_FRONT_BACK_MAIN,
  TS_SESSION_SUBSCRIBER_BUFFER_TYPE_FRONT_BACK_SUB
} ts_session_subscriber_buffer_type_t;

#if defined(__clang__) || defined(__GNUC__) || defined(_MSC_VER)
#pragma pack(push, 4)
#endif

// Main buffer header for shared CPU Memory Front/Back Buffers.
// The Main Buffer contains metadata about the current front/back buffer and the versioning for both the main and sub buffer.
// The information from the main buffer can be used to always read the front buffer.
typedef struct
{
  uint32_t version;                                  // E.g., TS_SESSION_SUBSCRIBER_COMMAND_VERSION_1.
  uint32_t front_buffer_index;                       // index of the front buffer. (0: main buffer, 1: sub buffer)
  uint32_t length;                                   // length of valid data of this buffer.
  uint32_t pixel_data_offset;                        // offset of pixel data from the beginning of this buffer. (in bytes, total length of pixel data = length - pixel_data_offset)
  uint32_t frame_id;                                 // frame id. (used to skip duplicated frames)
  uint32_t width;                                    // width of the frame.
  uint32_t height;                                   // height of the frame.
  ts_session_subscriber_pixel_format_t pixel_format; // pixel format.
} ts_session_subscriber_main_buffer_header_t;

// Sub buffer header.
typedef struct
{
  uint32_t length;            // length of valid data of this buffer.
  uint32_t pixel_data_offset; // offset of pixel data from the beginning of this buffer. (in bytes, total length of pixel data = length - pixel_data_offset)
  uint32_t frame_id;          // frame id. (used to skip duplicated frames)
  uint32_t width;             // width of the frame.
  uint32_t height;            // height of the frame.
} ts_session_subscriber_sub_buffer_header_t;

// Single frame buffer header.
typedef struct
{
  uint32_t version;                                  // E.g., TS_SESSION_SUBSCRIBER_COMMAND_VERSION_1.
  uint32_t length;                                   // length of valid data of this buffer.
  uint32_t pixel_data_offset;                        // offset of pixel data from the beginning of this buffer. (in bytes, total length of pixel data = length - pixel_data_offset)
  uint32_t frame_id;                                 // frame id. (used to skip duplicated frames)
  uint32_t width;                                    // width of the frame.
  uint32_t height;                                   // height of the frame.
  ts_session_subscriber_pixel_format_t pixel_format; // pixel format.
} ts_session_subscriber_single_frame_buffer_header_t;

// Command header.
typedef struct
{
  uint32_t                             version; // E.g., TS_SESSION_SUBSCRIBER_COMMAND_VERSION_1.
  uint32_t                             length;  // Total message length (header + payload).
  ts_session_subscriber_command_type_t type;
  uint64_t                             target_session_id;    // always set to the target session id this command is for or is originating from.
  uint64_t                             target_subscriber_id; // 0 if broadcast.
} ts_session_subscriber_command_header_t;

// Payload for TS_SESSION_SUBSCRIBER_CMD_ON_FRAME.
// The frame data is not included in the message, but is attached seperately or was sent beforehand. The buffer is identified by it's id.
// SESSION => SUBSCRIBER.
typedef struct
{
  uint64_t buffer_id; // buffer id, in general the memory address of the buffer in the client lib process.
} ts_session_subscriber_on_frame_payload_t;

// Payload for TS_SESSION_SUBSCRIBER_CMD_ON_FRAME_ACK
// SUBSCRIBER => SESSION.
typedef struct
{
  uint32_t frame_id;
} ts_session_subscriber_on_frame_ack_payload_t;

// Payload for TS_SESSION_SUBSCRIBER_CMD_ON_RESIZE.
// SUBSCRIBER => SESSION.
typedef struct
{
  uint32_t width;
  uint32_t height;
} ts_session_subscriber_on_resize_payload_t;

// Payload for TS_SESSION_SUBSCRIBER_CMD_ON_PAUSED_CHANGED.
// SESSION => SUBSCRIBER.
typedef struct
{
  uint32_t paused; // 0 or 1.
} ts_session_subscriber_on_paused_changed_payload_t;

// Payload for TS_SESSION_SUBSCRIBER_CMD_ON_BUFFER_SINGLE.
// Contains a single frame buffer. The header is of format |ts_session_subscriber_single_frame_buffer_header_t|.
// SESSION => SUBSCRIBER.
typedef struct
{
  uint64_t                                buffer_ptr; // buffer pointer, in general the memory address of the buffer in the client lib process.
  uint32_t                                buffer_length;
  ts_session_subscriber_buffer_location_t buffer_location;
  ts_session_subscriber_buffer_type_t     buffer_type;
} ts_session_subscriber_on_buffer_single_payload_t;

// Payload for TS_SESSION_SUBSCRIBER_CMD_ON_BUFFER_FRONT_BACK.
// Contains a main buffer and a sub buffer. The main buffer header
// contains metadata about the current front/back buffer and
// is of type |ts_session_subscriber_main_buffer_header_t|.
// The sub buffer's header is of type |ts_session_subscriber_sub_buffer_header_t|.
// SESSION => SUBSCRIBER.
typedef struct
{
  uint64_t                                main_buffer_ptr; // main buffer pointer, in general the memory address of the buffer in the client lib process.
  uint32_t                                main_buffer_length;
  uint64_t                                sub_buffer_ptr; // sub buffer pointer, in general the memory address of the buffer in the client lib process.
  uint32_t                                sub_buffer_length;
  ts_session_subscriber_buffer_location_t buffer_location;
} ts_session_subscriber_on_buffer_front_back_payload_t;

// Payload for TS_SESSION_SUBSCRIBER_CMD_ADD_SUBSCRIBER.
// SUBSCRIBER => SESSION.
typedef struct
{
  uint64_t request_id;
} ts_session_subscriber_add_subscriber_payload_t;

// Payload for TS_SESSION_SUBSCRIBER_CMD_ADD_SUBSCRIBER_RESP.
// SESSION => SUBSCRIBER.
typedef struct
{
  uint64_t request_id;
} ts_session_subscriber_add_subscriber_resp_payload_t;

// Payload for TS_SESSION_SUBSCRIBER_CMD_ON_RELEASE_BUFFER.
// SESSION => SUBSCRIBER.
typedef struct
{
  uint64_t buffer_ptr; // buffer pointer, in general the memory address of the buffer in the client lib process.
} ts_session_subscriber_on_release_buffer_payload_t;

// Payload for TS_SESSION_SUBSCRIBER_CMD_ON_AUDIO_FRAMES.
// SESSION => SUBSCRIBER.
typedef struct
{
  uint64_t buffer_ptr;
  int      bits_per_sample;
  int      sample_rate;
  uint32_t number_of_channels;
  uint32_t number_of_frames;
  int64_t  absolute_capture_timestamp_ms;
  float    volume;
} ts_session_subscriber_on_audio_frames_payload_t;

// Payload for TS_SESSION_SUBSCRIBER_CMD_ON_AUDIO_ENABLED_CHANGED.
// SESSION => SUBSCRIBER.
typedef struct
{
  uint32_t audio_enabled;
} ts_session_subscriber_on_audio_enabled_changed_payload_t;

// Overall Command structure.
typedef struct
{
  ts_session_subscriber_command_header_t header;
  union
  {
    ts_session_subscriber_on_frame_payload_t                 frame;
    ts_session_subscriber_on_resize_payload_t                resize;
    ts_session_subscriber_on_paused_changed_payload_t        paused;
    ts_session_subscriber_on_frame_ack_payload_t             frame_ack;
    ts_session_subscriber_on_buffer_single_payload_t         buffer_single;
    ts_session_subscriber_on_buffer_front_back_payload_t     buffer_front_back;
    ts_session_subscriber_on_release_buffer_payload_t        release_buffer;
    ts_session_subscriber_add_subscriber_payload_t           add_subscriber;
    ts_session_subscriber_add_subscriber_resp_payload_t      add_subscriber_resp;
    ts_session_subscriber_on_audio_frames_payload_t          audio_frames;
    ts_session_subscriber_on_audio_enabled_changed_payload_t audio_enabled_changed;
  } payload;
} ts_session_subscriber_command_t;

#if defined(__clang__) || defined(__GNUC__) || defined(_MSC_VER)
#pragma pack(pop)
#endif


#ifdef __cplusplus
}
#endif

#endif // TS_SESSION_SUBSCRIBER_PROTOCOL_H_
