/*
 * TeamSpeak SDK server creation params sample
 *
 * Copyright (c) TeamSpeak-Systems
 */

#ifdef _WINDOWS
#define _CRT_SECURE_NO_WARNINGS
#include <Windows.h>
#else
#include <unistd.h>
#include <stdlib.h>
#include <string.h>
#endif
#include <stdio.h>

#include <teamspeak/public_definitions.h>
#include <teamspeak/public_errors.h>
#include <teamspeak/serverlib_publicdefinitions.h>
#include <teamspeak/serverlib.h>
#include "id_io.h"

#define DEFAULT_VIRTUAL_SERVER_ID 1

/* Maximum number of clients allowed per virtual server */
#define MAX_CLIENTS 8

#ifdef _WINDOWS
#define SLEEP(x) Sleep(x)
#else
#define SLEEP(x) usleep(x*1000)
#endif

#define CHECK_ERROR(x) if((error = x) != ERROR_ok) { goto on_error; }

/*
 * Callback when client has connected.
 *
 * Parameter:
 *   serverID  - Virtual server ID
 *   clientID  - ID of connected client
 *   channelID - ID of channel the client joined
 */
void onClientConnected(uint64 serverID, anyID clientID, uint64 channelID, unsigned int* removeClientError) {
    char* clientName;
    unsigned int error;

    /* Query client nickname */
    if((error = ts3server_getClientVariableAsString(serverID, clientID, CLIENT_NICKNAME, &clientName)) != ERROR_ok) {
        char* errormsg;
        if(ts3server_getGlobalErrorMessage(error, &errormsg) == ERROR_ok) {
            printf("Error querying client nickname: %s\n", errormsg);
            ts3server_freeMemory(errormsg);
        }
        return;
    }

    printf("Client '%s' joined channel %llu on virtual server %llu\n", clientName, (unsigned long long) channelID, (unsigned long long)serverID);
}

/*
 * Callback when client has disconnected.
 *
 * Parameter:
 *   serverID  - Virtual server ID
 *   clientID  - ID of disconnected client
 *   channelID - ID of channel the client left
 */
void onClientDisconnected(uint64 serverID, anyID clientID, uint64 channelID) {
    printf("Client %u left channel %llu on virtual server %llu\n", clientID, (unsigned long long)channelID, (unsigned long long)serverID);
}

/*
 * Callback when client has moved.
 *
 * Parameter:
 *   serverID     - Virtual server ID
 *   clientID     - ID of moved client
 *   oldChannelID - ID of old channel the client left
 *   newChannelID - ID of new channel the client joined
 */
void onClientMoved(uint64 serverID, anyID clientID, uint64 oldChannelID, uint64 newChannelID) {
    printf("Client %u moved from channel %llu to channel %llu on virtual server %llu\n", clientID, (unsigned long long)oldChannelID, (unsigned long long)newChannelID, (unsigned long long)serverID);
}

/*
 * Callback when channel has been created.
 *
 * Parameter:
 *   serverID        - Virtual server ID
 *   invokerClientID - ID of the client who created the channel
 *   channelID       - ID of the created channel
 */
void onChannelCreated(uint64 serverID, anyID invokerClientID, uint64 channelID) {
    printf("Channel %llu created by %u on virtual server %llu\n", (unsigned long long)channelID, invokerClientID, (unsigned long long)serverID);
}

/*
 * Callback when channel has been edited.
 *
 * Parameter:
 *   serverID        - Virtual server ID
 *   invokerClientID - ID of the client who edited the channel
 *   channelID       - ID of the edited channel
 */
void onChannelEdited(uint64 serverID, anyID invokerClientID, uint64 channelID) {
    printf("Channel %llu edited by %u on virtual server %llu\n", (unsigned long long)channelID, invokerClientID, (unsigned long long)serverID);
}

/*
 * Callback when channel has been deleted.
 *
 * Parameter:
 *   serverID        - Virtual server ID
 *   invokerClientID - ID of the client who deleted the channel
 *   channelID       - ID of the deleted channel
 */
void onChannelDeleted(uint64 serverID, anyID invokerClientID, uint64 channelID) {
    printf("Channel %llu deleted by %u on virtual server %llu\n", (unsigned long long)channelID, invokerClientID, (unsigned long long)serverID);
}

void showHelp() {
    printf("\n[q] - Quit\n[h] - Show this help\n[v] - List virtual servers\n[c] - Show channels of virtual server %d\n", DEFAULT_VIRTUAL_SERVER_ID);
    printf("[l] - Show clients of virtual server %d\n[n] - Create new channel on virtual server %d with generated name\n[N] - Create new channel on virtual server %d with custom name\n", DEFAULT_VIRTUAL_SERVER_ID, DEFAULT_VIRTUAL_SERVER_ID, DEFAULT_VIRTUAL_SERVER_ID);
    printf("[d] - Delete channel on virtual server %d\n\n", DEFAULT_VIRTUAL_SERVER_ID);
    printf("[C] - Create new virtual server\n[E] - Edit virtual server\n[S] - Stop virtual server\n\n");
}

void emptyInputBuffer() {
    int c;
    while((c = getchar()) != '\n' && c != EOF);
}

void showVirtualServers() {
    uint64* ids;
    int i;
    unsigned int error;

    printf("\nList of virtual servers:\n");
    if((error = ts3server_getVirtualServerList(&ids)) != ERROR_ok) {  /* Get array of virtual server IDs */
        printf("Error getting virtual server list: %d\n", error);
        return;
    }
    if(!ids[0]) {
        printf("No virtual servers\n\n");
        ts3server_freeMemory(ids);
        return;
    }
    for(i=0; ids[i]; i++) {
        char* name;
        int slotCount;
        char* virtualServerUniqueIdentifier;
        if((error = ts3server_getVirtualServerVariableAsString(ids[i], VIRTUALSERVER_NAME, &name)) != ERROR_ok) {  /* Query server name */
            printf("Error getting virtual server nickname: %d\n", error);
            break;
        }
        if((error = ts3server_getVirtualServerVariableAsInt(ids[i], VIRTUALSERVER_MAXCLIENTS, &slotCount)) != ERROR_ok) {
            printf("Error getting virtual server slot count: %d\n", error);
            break;
        }
        if((error = ts3server_getVirtualServerVariableAsString(ids[i], VIRTUALSERVER_UNIQUE_IDENTIFIER, &virtualServerUniqueIdentifier)) != ERROR_ok) {
            printf("Error getting virtual server unique identifier: %d\n", error);
            break;
        }
        printf("ID=%llu  NAME=\"%s\"  CAPACITY=%d  Unique Identifier=\"%s\"\n", (unsigned long long)ids[i], name, slotCount, virtualServerUniqueIdentifier);
        ts3server_freeMemory(name);  /* Do not free memory if above function returned an error */
        ts3server_freeMemory(virtualServerUniqueIdentifier);
    }
    printf("\n");

    ts3server_freeMemory(ids);  /* Release array */
}

void showChannels(uint64 serverID) {
    uint64* ids;
    int i;
    unsigned int error;

    printf("\nList of channels on virtual server %llu:\n", (unsigned long long)serverID);
    if((error = ts3server_getChannelList(serverID, &ids)) != ERROR_ok) {  /* Get array of channel IDs */
        printf("Error getting channel list: %d\n", error);
        return;
    }
    if(!ids[0]) {
        printf("No channels\n\n");
        ts3server_freeMemory(ids);
        return;
    }
    for(i=0; ids[i]; i++) {
        char* name;
        if((error = ts3server_getChannelVariableAsString(serverID, ids[i], CHANNEL_NAME, &name)) != ERROR_ok) {  /* Query channel name */
            printf("Error querying channel name: %d\n", error);
            break;
        }
        printf("%llu - %s\n", (unsigned long long)ids[i], name);
        ts3server_freeMemory(name);  /* Do not free memory if above function returned an error */
    }
    printf("\n");

    ts3server_freeMemory(ids);  /* Release array */
}

void showClients(uint64 serverID) {
    anyID* ids;
    int i;
    unsigned int error;

    printf("\nList of clients on virtual server %llu:\n", (unsigned long long)serverID);
    if((error = ts3server_getClientList(serverID, &ids)) != ERROR_ok) {  /* Get array of client IDs */
        printf("Error getting client list: %d\n", error);
        return;
    }
    if(!ids[0]) {
        printf("No clients\n\n");
        ts3server_freeMemory(ids);
        return;
    }
    for(i=0; ids[i]; i++) {
        char* name;
        if((error = ts3server_getClientVariableAsString(serverID, ids[i], CLIENT_NICKNAME, &name)) != ERROR_ok) {  /* Query client nickname */
            printf("Error querying client nickname: %d\n", error);
            break;
        }
        printf("%u - %s\n", ids[i], name);
        ts3server_freeMemory(name);  /* Do not free memory if above function returned an error */
    }
    printf("\n");

    ts3server_freeMemory(ids);  /* Release array */
}

void createDefaultChannelName(char *name) {
    static int i = 11;  /* We already have 10 channels in this example */
    sprintf(name, "Channel_%d", i++);
}

void enterName(char *name) {
    char *s;
    printf("\nEnter name: ");
    fgets(name, BUFSIZ, stdin);
    s = strrchr(name, '\n');
    if(s) {
        *s = '\0';
    }
}

void createChannel(uint64 serverID, const char *name) {
    /* This code demonstrates how to use the new createChannel API. */

    unsigned int error;
    uint64 newChannelID;
    struct TS3ChannelCreationParams* ccp;
    struct TS3Variables* vars;

    /* Create a new struct channel parameters struct. Memory is allocated in the clientlib,
     * so the struct needs to be freed using ts3server_freeMemory when done */
    error = ts3server_makeChannelCreationParams(&ccp);
    if(error != ERROR_ok) {
        printf("Failed to make channel creation params: %d\n", error);
        goto leave;
    }

    /* Set essential channel paramters:
     * parentID 0 -> create as top-level channel
     * channelID 0 -> server will automatically assign a new channel ID. An existing channel ID would be an error */
    error = ts3server_setChannelCreationParams(ccp, 0, 0);
    if(error != ERROR_ok) {
        printf("Failed to set channel creation params: %d\n", error);
        goto leave;
    }

    /* Query a struct TS3Variables, used to set additional parameters below */
    error = ts3server_getChannelCreationParamsVariables(ccp, &vars);
    if(error != ERROR_ok) {
        printf("Failed to get variables from channel creation params: %d\n", error);
        goto leave;
    }

    /* Use above queried struct TS3Variables to set additional channel parameters */

    /* Set codec quality */
    error = ts3server_setVariableAsInt(vars, CHANNEL_CODEC_QUALITY, 10);
    if (error != ERROR_ok) {
        printf("Error setting channel quality: %d", error);
        goto leave;
    }

    /* Set channel name */
    error = ts3server_setVariableAsString(vars, CHANNEL_NAME, name);
    if(error != ERROR_ok) {
        printf("Failed to set channel name: %d\n", error);
        goto leave;
    }

    /* Make channel permanent (important, otherwise the empty channel would be immediately deleted after creation) */
    error = ts3server_setVariableAsInt(vars, CHANNEL_FLAG_PERMANENT, 1);
    if(error != ERROR_ok) {
        printf("Failed to set channel name: %d\n", error);
        goto leave;
    }

    /* Set channel topic */
    error = ts3server_setVariableAsString(vars, CHANNEL_TOPIC, "My topic");
    if(error != ERROR_ok) {
        printf("Failed to set channel topic: %d\n", error);
        goto leave;
    }

    /* Finally create the channel. Write the ID of the created channel into newChannelID  */
    error = ts3server_createChannel(serverID, ccp, CHANNEL_CREATE_FLAG_NONE, &newChannelID);
    if(error != ERROR_ok) {
        printf("Failed to create channel: %d\n", error);
        goto leave;
    }

    printf("\nCreated channel: %llu\n\n", (unsigned long long)newChannelID);

leave:
    /* Cleanup struct TS3ChannelCreationParams */
    ts3server_freeMemory(ccp);
}

void deleteChannel(uint64 serverID) {
    uint64 channelID;
    int n;
    unsigned int error;

    /* Query channel ID from user */
    printf("\nEnter ID of channel to delete: ");
    n = scanf("%llu", (unsigned long long*)&channelID);
    emptyInputBuffer();
    if(n == 0) {
        printf("Invalid input. Please enter a number.\n\n");
        return;
    }

    /* Delete channel */
    if((error = ts3server_channelDelete(serverID, channelID, 0)) != ERROR_ok) {
        char* errormsg;
        if(ts3server_getGlobalErrorMessage(error, &errormsg) == ERROR_ok) {
            printf("Error deleting channel: %s\n\n", errormsg);
            ts3server_freeMemory(errormsg);
        }
    }
}

void renameChannel(uint64 serverID) {
    uint64 channelID;
    int n;
    unsigned int error;
    char name[BUFSIZ];

    /* Query channel ID from user */
    printf("\nEnter ID of channel to rename: ");
    n = scanf("%llu", (unsigned long long*)&channelID);
    emptyInputBuffer();
    if(n == 0) {
        printf("Invalid input. Please enter a number.\n\n");
        return;
    }

    /* Query new channel name from user */
    enterName(name);

    /* Change channel name and flush changes */
    CHECK_ERROR(ts3server_setChannelVariableAsString(serverID, channelID, CHANNEL_NAME, name));
    CHECK_ERROR(ts3server_flushChannelVariable(serverID, channelID));

    printf("Renamed channel %llu\n\n", (unsigned long long)channelID);
    return;

on_error:
    printf("Error renaming channel: %d\n\n", error);
}

void moveClient(uint64 serverID) {
    anyID clientIDArray[2];  /* We only want to move one client plus terminating null end-marker */
    uint64 newChannelID;  /* ID of channel to move the client into */
    unsigned int error;
    int n;

    /* Query client ID from user */
    printf("\nEnter ID of client to move: ");
    n = scanf("%hu", &clientIDArray[0]);
    emptyInputBuffer();
    if(n == 0) {
        printf("Invalid input. Please enter a number.\n\n");
        return;
    }

    clientIDArray[1] = 0;  /* Add end-marker */

    /* Query channel ID from user */
    printf("\nEnter ID of channel to move client into: ");
    n = scanf("%llu", (unsigned long long*)&newChannelID);
    emptyInputBuffer();
    if(n == 0) {
        printf("Invalid input. Please enter a number.\n\n");
        return;
    }

    /* Move client and check for error */
    if((error = ts3server_clientMove(serverID, newChannelID, clientIDArray)) != ERROR_ok) {
        char* errormsg;
        if(ts3server_getGlobalErrorMessage(error, &errormsg) == ERROR_ok) {
            printf("Error moving client: %s\n\n", errormsg);
            ts3server_freeMemory(errormsg);
        }
        return;
    }

    printf("Client %d moved to channel %llu\n", clientIDArray[0], (unsigned long long)newChannelID);
}

/* Shows how to use the new server params method to create a virtual server */
uint64 createVirtualServer2(const char* name, int port, unsigned int maxClients) {
    char buffer[BUFSIZ] = { 0 };
    char filename[BUFSIZ];
    char port_str[20];
    char* keyPair;
    unsigned int error;
    int i;
    struct TS3VirtualServerCreationParams* vscp;
    struct TS3ChannelCreationParams*       ccp;
    struct TS3Variables*                   vars;
    uint64 serverID = 0;

    /* Assemble filename: keypair_<port>.txt */
    strcpy(filename, "keypair_");
    sprintf(port_str, "%d", port);
    strcat(filename, port_str);
    strcat(filename, ".txt");

    /* Try reading keyPair from file */
    if(readKeyPairFromFile(filename, buffer) == 0) {
        keyPair = buffer;  /* Id read from file */
    } else {
        keyPair = "";  /* No Id saved, start virtual server with empty keyPair string */
    }

    /* Create server creation params, write result into empty struct TS3VirtualServerCreationParams */
    error = ts3server_makeVirtualServerCreationParams(&vscp);
    if(error != ERROR_ok) {
        printf("Error during makeVirtualServerCreationParams: %d\n", error);
        return 0;
    }

    /* Set essential connection data to server creation params:
     * port 9987, ip NULL (localhost), server keypair, max clients, channel count, virtual server ID */
    error = ts3server_setVirtualServerCreationParams(vscp, port, NULL, keyPair, maxClients, 10, 1);
    if(error != ERROR_ok) {
        printf("Error during setVirtualServerCreationParams: %d\n", error);
        goto leave;
    }

    /* Query the struct TS3Variables from the server creation params, to set some additional parameters */
    error = ts3server_getVirtualServerCreationParamsVariables(vscp, &vars);
    if(error != ERROR_ok) {
        printf("Error during getVirtualServerCreationParamsVariables: %d\n", error);
        goto leave;
    }

    /* Below we write some additional server parameters into the struct TS3Variables.
     * These parameters are not part of the essential parameters, which we defined earlier
     * in ts3server_setVirtualServerCreationParams */

    /* Set virtual server name */
    error = ts3server_setVariableAsString(vars, VIRTUALSERVER_NAME, "TeamSpeak3 SDK Server");
    if(error != ERROR_ok) {
        printf("Error setting server name: %d\n", error);
        goto leave;
    }

    /* Set virtual server password */
    error = ts3server_setVariableAsString(vars, VIRTUALSERVER_PASSWORD, "secret");
    if(error != ERROR_ok) {
        printf("Error setting server password: %d\n", error);
        goto leave;
    }

    /* Create 10 channels, write them into the struct channel params, which we queried earlier */
    for(i = 0; i < 10; ++i) {
        /* Get the channel creation param for the channel index. This channel param structs are subobjects
         * created inside the server creation params.
         * The number of available channel params depends on the number of channels set above in
         * ts3server_setVirtualServerCreationParams (10 in this sample)
         * Write result into the reused struct TS3ChannelCreationParams. */
        error = ts3server_getVirtualServerCreationParamsChannelCreationParams(vscp, i, &ccp);
        if(error != ERROR_ok) {
            printf("Error during getVirtualServerCreationParamsChannelCreationParams: %d\n", error);
            goto leave;
        }

        /* Now fill the struct channel creation params with some channel data */

        /* Set essential data: channel parent ID and channel ID.
         * The idea here is to be able to restore a previously saved channel structure keeping the
         * same channel ID over server restarts. If we would create channels the old way, it would not
         * be possible to guarantee the channelID, as it would be assigned automatically by the server.
         * Basically this allows to recreate snapshots of the channel tree. */
        error = ts3server_setChannelCreationParams(ccp, 0, i + 1);
        if(error != ERROR_ok) {
            printf("Error during setChannelCreationParams: %d\n", error);
            goto leave;
        }

        /* As above with the server, query a TS3Variables (reused) for this channel, which we
         * can fill with some additional parameters that are not part of the essentials */
        error = ts3server_getChannelCreationParamsVariables(ccp, &vars);
        if(error != ERROR_ok) {
            printf("Error during getChannelCreationParamsVariables: %d\n", error);
            goto leave;
        }

        /* Now fill the queried struct TS3Variables with additional parameters */

        /* Make first channel default */
        if(i == 0) {
            error = ts3server_setVariableAsInt(vars, CHANNEL_FLAG_DEFAULT, 1);
            if(error != ERROR_ok) {
                printf("Error setting channel %d default: %d", (i + 1), error);
                goto leave;
            }
        }

        /* Set codec quality */
        error = ts3server_setVariableAsInt(vars, CHANNEL_CODEC_QUALITY, 10);
        if (error != ERROR_ok) {
            printf("Error setting channel quality: %d", error);
            goto leave;
        }

        /* Set channel name as "channel #" */
        sprintf(buffer, "channel %d", (i + 1));
        error = ts3server_setVariableAsString(vars, CHANNEL_NAME, buffer);
        if(error != ERROR_ok) {
            printf("Error setting channel %d name: %d", (i + 1), error);
            goto leave;
        }

        /* Make channel permanent */
        error = ts3server_setVariableAsInt(vars, CHANNEL_FLAG_PERMANENT, 1);
        if(error != ERROR_ok) {
            printf("Error setting channel %d permanent: %d", (i + 1), error);
            goto leave;
        }
    }

    /* Finally create the virtual server, using the server parameters we setup earlier.
     * In addition automatically create all channels we set in the channel parameters, which is
     * included as part of the server parameters.
     * The function writes the virtual server ID into serverID variable */
    error = ts3server_createVirtualServer2(vscp, VIRTUALSERVER_CREATE_FLAG_NONE, &serverID);
    if(error != ERROR_ok) {
        printf("Error during createVirtualServer2: %d\n", error);
        goto leave;
    }

leave:
    /* Cleanup struct virtual server param. The included struct channel param will be automatically
     * freed when the virtual server param is freed, so you do not need to call freeMemory on
     * the channel params, too. */
    ts3server_freeMemory(vscp);

    /* Finally return the virtual server ID of our just created virtual server */
    return serverID;
}

uint64 startVirtualServer() {
    int n;
    int port;
    unsigned int maxClients;
    char name[BUFSIZ];

    /* Ask user for server port */
    printf("\nEnter server port (default 9987): ");
    n = scanf("%d", &port);
    emptyInputBuffer();
    if(n == 0) {
        printf("Invalid input. Please enter a number.\n\n");
        return 0;
    }

    printf("\nEnter server capacity (default %d): ", MAX_CLIENTS);
    n = scanf("%d", &maxClients);
    emptyInputBuffer();
    if(n == 0) {
        printf("Invalid input. Please enter a number.\n\n");
        return 0;
    }
    
    /* Ask user for server name */
    enterName(name);

    return createVirtualServer2(name, port, maxClients);
}

void editVirtualServer() {
    int n;
    uint64 serverID;
    int currentSlotCount, newSlotCount;
    unsigned int error;

    printf("\nEnter ID of virtual server to edit: ");
    n = scanf("%llu", (unsigned long long*)&serverID);
    emptyInputBuffer();
    if(n == 0) {
        printf("Invalid input. Please enter a number.\n\n");
        return;
    }
    if((error = ts3server_getVirtualServerVariableAsInt(serverID, VIRTUALSERVER_MAXCLIENTS, &currentSlotCount)) != ERROR_ok) {
        printf("Error getting the current capcity of virtual server: %d\n\n", error);
        return;
    }
    printf("\nEnter new capacity of virtual server (currently %d): ", currentSlotCount);
    n = scanf("%d", &newSlotCount);
    emptyInputBuffer();
    if(n == 0) {
        printf("Invalid input. Please enter a number.\n\n");
        return;
    }
    if((error = ts3server_setVirtualServerVariableAsInt(serverID, VIRTUALSERVER_MAXCLIENTS, newSlotCount)) != ERROR_ok) {
        printf("Error setting the new capacity: %d\n\n", error);
        return;
    }
    if((error = ts3server_flushVirtualServerVariable(serverID)) != ERROR_ok) {
        printf("Error flushing server variable updates %d\n\n", error);
        return;
    }
}

void stopVirtualServer() {
    int n;
    uint64 serverID;
    unsigned int error;

    printf("\nEnter ID of virtual server to stop: ");
    n = scanf("%llu", (unsigned long long*)&serverID);
    emptyInputBuffer();
    if(n == 0) {
        printf("Invalid input. Please enter a number.\n\n");
        return;
    }

    if((error = ts3server_stopVirtualServer(serverID)) != ERROR_ok) {
        printf("Error stopping virtual server: %d\n\n", error);
    }
}

int main(int argc, char **argv) {
    char *version;
    short abort = 0;
    uint64 serverID;
    unsigned int error;
    int unknownInput = 0;
    uint64* ids;
    int i;

    /* Create struct for callback function pointers */
    struct ServerLibFunctions funcs;

    /* Initialize all callbacks with NULL */
    memset(&funcs, 0, sizeof(struct ServerLibFunctions));

    /* Now assign the used callback function pointers */
    funcs.onClientConnected          = onClientConnected;
    funcs.onClientDisconnected       = onClientDisconnected;
    funcs.onClientMoved              = onClientMoved;
    funcs.onChannelCreated           = onChannelCreated;
    funcs.onChannelEdited            = onChannelEdited;
    funcs.onChannelDeleted           = onChannelDeleted;

    /* Initialize server lib with callbacks */
    if((error = ts3server_initServerLib(&funcs, LogType_FILE | LogType_CONSOLE | LogType_USERLOGGING, NULL, argc, (const char* const *)argv)) != ERROR_ok) {
        char* errormsg;
        if(ts3server_getGlobalErrorMessage(error, &errormsg) == ERROR_ok) {
            printf("Error initialzing serverlib: %s\n", errormsg);
            ts3server_freeMemory(errormsg);
        }
        return 1;
    }

    printf("Server running\n");

    /* Query and print server lib version */
    if((error = ts3server_getServerLibVersion(&version)) != ERROR_ok) {
        printf("Error querying server lib version: %d\n", error);
        return 1;
    }
    printf("Server lib version: %s\n", version);
    ts3server_freeMemory(version);  /* Release dynamically allocated memory */

    /* Create a virtual server with the new server params method */
    serverID = createVirtualServer2("TS3 SDK Test Server", 9987, MAX_CLIENTS);

    /* Simple commandline interface */
    printf("\nTeamSpeak 3 server commandline interface\n");
    showHelp();

    while(!abort) {
        int c;
        if(unknownInput == 0) {
            printf("\nEnter Command (h for help)> ");
        }
        unknownInput = 0;
        c = getchar();
        switch(c) {
            case 'q':
                printf("\nShutting down server...\n");
                abort = 1;
                break;
            case 'h':
                showHelp();
                break;
            case 'v':
                showVirtualServers();
                break;
            case 'c':
                showChannels(serverID);
                break;
            case 'l':
                showClients(serverID);
                break;
            case 'n':
            {
                char name[BUFSIZ];
                createDefaultChannelName(name);
                createChannel(serverID, name);
                break;
            }
            case 'N':
            {
                char name[BUFSIZ];
                emptyInputBuffer();
                enterName(name);
                createChannel(serverID, name);
                break;
            }
            case 'd':
                deleteChannel(serverID);
                break;
            case 'C':
                startVirtualServer();
                break;
            case 'E':
                editVirtualServer();
                break;
            case 'S':
                stopVirtualServer();
                break;
            default:
                unknownInput = 1;
        }

        SLEEP(50);
    }

    /* Stop virtual servers to make sure connected clients are notified instead of dropped */
    if((error = ts3server_getVirtualServerList(&ids)) != ERROR_ok) {  /* Get array of virtual server IDs */
        printf("Error getting virtual server list: %d\n", error);
    } else {
        for(i=0; ids[i]; i++) {
            if((error = ts3server_stopVirtualServer(ids[i])) != ERROR_ok) {
                printf("Error stopping virtual server: %d\n", error);
                break;
            }
      }
      ts3server_freeMemory(ids);
    }

    /* Shutdown server lib */
    if((error = ts3server_destroyServerLib()) != ERROR_ok) {
        printf("Error destroying server lib: %d\n", error);
        return 1;
    }

    return 0;
}
