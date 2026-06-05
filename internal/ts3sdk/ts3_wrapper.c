#include "teamspeak/clientlib.h"
#include "teamspeak/public_errors.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// Forward declaration of Go export. We forward the new connection status and the
// SDK error number so the Go side can react and report failures precisely.
extern void GoCallbackHandler(char* eventName, unsigned long long serverConnectionHandlerID, unsigned long long newStatus, unsigned long long errorNumber);

// Explicit declaration of the poke function (not part of the public SDK headers, found via nm).
extern unsigned int ts3client_requestClientPoke(uint64 serverConnectionHandlerID, uint64 clientID, const char* message, const char* returnCode);

static void onConnectStatusChangeEvent(uint64 serverConnectionHandlerID, int newStatus, unsigned int errorNumber) {
    printf("SDK: Connect status changed to: %d (error %u)\n", newStatus, errorNumber);
    fflush(stdout);
    GoCallbackHandler("onConnectStatusChangeEvent", serverConnectionHandlerID, (unsigned long long)newStatus, (unsigned long long)errorNumber);
}

static void onServerErrorEvent(uint64 serverConnectionHandlerID, const char* errorMessage, unsigned int error, const char* returnCode, const char* extraMessage) {
    printf("SDK: Server error: %s (code %u) %s\n", errorMessage ? errorMessage : "", error, extraMessage ? extraMessage : "");
    fflush(stdout);
}

static struct ClientUIFunctions callbacks;

static inline int init_ts3() {
    memset(&callbacks, 0, sizeof(struct ClientUIFunctions));
    callbacks.onConnectStatusChangeEvent = onConnectStatusChangeEvent;
    callbacks.onServerErrorEvent = onServerErrorEvent;

    // resourcesFolder "." points the lib at the working directory to locate the
    // soundbackends (statically bundled in this SDK build, but the path must be valid).
    int res = ts3client_initClientLib(&callbacks, NULL, LogType_FILE | LogType_CONSOLE, NULL, "./");
    printf("SDK: Initialization result: %d\n", res);
    fflush(stdout);
    return res;
}

static inline int spawn_connection_handler() {
    uint64 id = 0;
    if (ts3client_spawnNewServerConnectionHandler(0, &id) == 0) {
        printf("SDK: Spawned handler ID: %llu\n", (unsigned long long)id);
        fflush(stdout);
        return (int)id;
    }
    printf("SDK: Failed to spawn handler\n");
    fflush(stdout);
    return -1;
}

// Creates a fresh identity. Caller owns the returned pointer and must release it
// with free_sdk_memory(). Returns NULL on failure.
static inline char* create_identity() {
    char* identity = NULL;
    unsigned int err = ts3client_createIdentity(&identity);
    if (err != ERROR_ok) {
        printf("SDK: createIdentity failed: %u\n", err);
        fflush(stdout);
        return NULL;
    }
    return identity;
}

static inline void free_sdk_memory(void* ptr) {
    if (ptr) ts3client_freeMemory(ptr);
}

static inline int start_connection(int serverConnectionHandlerID, const char* identity, const char* host, int port, const char* nickname, const char* serverPassword) {
    int res = ts3client_startConnection((uint64)serverConnectionHandlerID, identity, host, (unsigned int)port, nickname, NULL, "", serverPassword ? serverPassword : "");
    printf("SDK: Connection request result: %d\n", res);
    fflush(stdout);
    return res;
}

static inline int poke_client(int serverConnectionHandlerID, int clientID, const char* message) {
    return (int)ts3client_requestClientPoke((uint64)serverConnectionHandlerID, (uint64)clientID, message, NULL);
}
