/*
 * TeamSpeak SDK server permission sample
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

/*
 * Callback when client has connected.
 *
 * Parameter:
 *   serverID  - Virtual server ID
 *   clientID  - ID of connected client
 *   channelID - ID of channel the client joined
 */
void onClientConnected(uint64 serverID, anyID clientID, uint64 channelID, unsigned int* removeClientError) {
    printf("Client %u joined channel %llu on virtual server %llu\n", clientID, (unsigned long long)channelID, (unsigned long long)serverID);
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

/*
 * Callback for user-defined logging.
 *
 * Parameter:
 *   logMessage        - Log message text
 *   logLevel          - Severity of log message
 *   logChannel        - Custom text to categorize the message channel
 *   logID             - Virtual server ID giving the virtual server source of the log event
 *   logTime           - String with the date and time the log entry occured
 *   completeLogString - Verbose log message including all previous parameters for convinience
 */
void onUserLoggingMessageEvent(const char* logMessage, int logLevel, const char* logChannel, uint64 logID, const char* logTime, const char* completeLogString) {
    /* Your custom error display here... */
    /* printf("LOG: %s\n", completeLogString); */
    if(logLevel == LogLevel_CRITICAL) {
        exit(1);  /* Your custom handling of critical errors */
    }
}

/*
 * Callback triggered when the specified client starts talking.
 *
 * Parameters:
 *   serverID - ID of the virtual server sending the callback
 *   clientID - ID of the client which started talking
 */
void onClientStartTalkingEvent(uint64 serverID, anyID clientID) {
    printf("onClientStartTalkingEvent serverID=%llu, clientID=%u\n", (unsigned long long)serverID, clientID);
}

/*
 * Callback triggered when the specified client stops talking.
 *
 * Parameters:
 *   serverID - ID of the virtual server sending the callback
 *   clientID - ID of the client which stopped talking
 */
void onClientStopTalkingEvent(uint64 serverID, anyID clientID) {
    printf("onClientStopTalkingEvent serverID=%llu, clientID=%u\n", (unsigned long long)serverID, clientID);
}

/*
 * Callback triggered when a license error occurs.
 *
 * Parameters:
 *   serverID  - ID of the virtual server on which the license error occured. This virtual server will be automatically
 *               shutdown. If the parameter is zero, all virtual servers are affected and have been shutdown.
 *   errorCode - Code of the occured error. Use ts3server_getGlobalErrorMessage() to convert to a message string
 */
void onAccountingErrorEvent(uint64 serverID, unsigned int errorCode) {
    char* errorMessage;
    if(ts3server_getGlobalErrorMessage(errorCode, &errorMessage) == ERROR_ok) {
        printf("onAccountingErrorEvent serverID=%llu, errorCode=%u: %s\n", (unsigned long long)serverID, errorCode, errorMessage);
        ts3server_freeMemory(errorMessage);
    }

    /* Your custom handling here. In a real application, you wouldn't stop the whole process because the TS3 server part went down.
     * The whole idea of this callback is to let you gracefully handle the TS3 server failing to start and to continue your application. */
    exit(1);
}

/*
 * Callback triggered before a client connects.
 *
 * Parameters:
 *   serverID  - ID of the virtual server on which the client is connecting.
 *   client    - Struct of client parameters like ident, nickname etc. Please view public_definitions.h.
 *
 * Note: You can deny the permission for special nicknames or identities here so the client connect is not allowed.
 */
unsigned int onPermClientCanConnect(uint64 serverID, const struct ClientMiniExport* client) {
    printf("onPermClientCanConnect\n\tserverID=%llu\n"
        "\tclientChannel=%llu\n\tclientClientID=%u\n\tclientIdent=%s\n\tclientNickname=%s\n",
        (unsigned long long)serverID, (unsigned long long)client->channel, client->ID, client->ident, client->nickname);

    // nickname of sdk client ist "client" so check it for a test
    if (strcmp(client->nickname, "client") == 0) {
        //return ERROR_permissions;
    }
    return ERROR_ok;
}

/*
 * Callback triggered before a channel will be created.
 *
 * Parameters:
 *   serverID        - ID of the virtual server where the client is requesting a channel creation.
 *   client          - Struct of client parameters like ident, nickname etc. Please view public_definitions.h.
 *   parentChannelID - Parent ID of the channel which is going to be created.
 *   variables       - Array of channel properties for the new channel.
 *
 * Note: You can deny the permission so creating the channel is not allowed.
 */
unsigned int onPermChannelCreate(uint64 serverID, const struct ClientMiniExport* client, uint64 parentChannelID, const struct VariablesExport* variables) {
    int i=0;

    printf("onPermChannelCreate\n\tserverID=%llu\n\tparentChannelID=%llu\n"
           "\tclientChannel=%llu\n\tclientClientID=%u\n\tclientIdent=%s\n\tclientNickname=%s\n",
           (unsigned long long)serverID, (unsigned long long)parentChannelID, (unsigned long long)client->channel, client->ID, client->ident, client->nickname);

    // e.g. check nickname for admin and deny creation
    if (strcmp(client->nickname, "admin") == 0) {
        return ERROR_permissions;
    }

    for(; i<CHANNEL_ENDMARKER; ++i) {
        struct VariablesExportItem item = variables->items[i];
        if (item.itemIsValid) {
            printf("\titem=%i itemIsValid=%i current=%s\n", i, item.itemIsValid, item.current);
            if (item.proposedIsSet) {
                printf("\titem=%i proposedIsSet=%i proposed=%s\n", i, item.proposedIsSet, item.proposed);
            }
        }
    }
    return ERROR_ok;
}

/*
 * Callback triggered before a channel description will be sent.
 *
 * Parameters:
 *   serverID        - ID of the virtual server on which the client is requesting the channel description.
 *   client          - Struct of client parameters like ident, nickname etc. Please view public_definitions.h.
 *
 * Note: You can deny the permission so getting the channel descritpion is not allowed.
 */
unsigned int onPermClientCanGetChannelDescription(uint64 serverID, const struct ClientMiniExport* client) {
    printf("onPermClientCanGetChannelDescription\n\tserverID=%llu\n"
        "\tclientChannel=%llu\n\tclientClientID=%u\n\tclientIdent=%s\n\tclientNickname=%s\n",
        (unsigned long long)serverID, (unsigned long long)client->channel, client->ID, client->ident, client->nickname);
    // check nickname for admin and deny
    if (strcmp(client->nickname, "admin") == 0) {
        return ERROR_permissions;
    }
    return ERROR_ok;
}

/*
 * Callback triggered before a client update
 *
 * Parameters:
 *   serverID        - ID of the virtual server on which the client is requesting an udpate.
 *   clientID        - ID of the client which triggered an update.
 *   variables       - Array of client properties.
 *
 * Note: You can deny the permission so updating a client variable is not allowed.
 */
unsigned int onPermClientUpdate(uint64 serverID, anyID clientID, const struct VariablesExport* variables) {
    int i=0;

    printf("onPermClientUpdate\n\tserverID=%llu\n\tclientID=%u\n", (unsigned long long)serverID, clientID);

    for(; i<CLIENT_ENDMARKER; ++i) {
        struct VariablesExportItem item = variables->items[i];
        if (item.itemIsValid) {
            printf("\titem=%i itemIsValid=%i current=%s\n", i, item.itemIsValid, item.current);
            if (item.proposedIsSet) {
                printf("\titem=%i proposedIsSet=%i proposed=%s\n", i, item.proposedIsSet, item.proposed);
                // e.g. check nickname for Admin and deny
                if (i == CLIENT_NICKNAME && strcmp(item.proposed, "Admin") == 0) {
                    return ERROR_permissions;
                }
            }
        }
    }
    return ERROR_ok;
}

/*
 * Callback triggered before one or more clients will be kicked from channel.
 *
 * Parameters:
 *   serverID        - ID of the virtual server on which the client is requesting the kick.
 *   client          - Struct of client parameters like ident, nickname etc. who is causing the kick. Please view public_definitions.h.
 *   toKickCount     - The number of clients to be kicked.
 *   toKickClients   - Array of structs which clients are going to be kicked.
 *   reasonText      - Optional reason text why the kick was initiated.
 *
 * Note: You can deny the permission so kicking is not allowed.
 */
unsigned int onPermClientKickFromChannel(uint64 serverID, const struct ClientMiniExport* client, int toKickCount, const struct ClientMiniExport* toKickClients, const char* reasonText) {
    printf("onPermClientKickFromChannel\n\tserverID=%llu\n"
        "\tclientChannel=%llu\n\tclientClientID=%u\n\tclientIdent=%s\n\tclientNickname=%s\n"
        "\ttoKickCount=%i\n"
        "\ttoKickClientsChannel=%llu\n\ttoKickClientsClientID=%u\n\ttoKickClientsIdent=%s\n\ttoKickClientsNickname=%s\n"
        "\treasonText=%s\n",
        (unsigned long long)serverID, (unsigned long long)client->channel, client->ID, client->ident, client->nickname, toKickCount, (unsigned long long)toKickClients->channel, toKickClients->ID, toKickClients->ident, toKickClients->nickname, reasonText);
    // e.g. check nickname for admin and deny
    if (strcmp(client->nickname, "admin") == 0) {
        return ERROR_permissions;
    }
    return ERROR_ok;
}

/*
 * Callback triggered before one or more clients will be kicked from server.
 *
 * Parameters:
 *   serverID        - ID of the virtual server on which the client is requesting the kick.
 *   client          - Struct of client parameters like ident, nickname etc. who is causing the kick. Please view public_definitions.h.
 *   toKickCount     - The number of clients to be kicked.
 *   toKickClients   - Array of structs which clients are going to be kicked.
 *   reasonText      - Optional reason text why the kick was initiated.
 *
 * Note: You can deny the permission so kicking is not allowed.
 */
unsigned int onPermClientKickFromServer(uint64 serverID, const struct ClientMiniExport* client, int toKickCount, const struct ClientMiniExport* toKickClients, const char* reasonText) {
    printf("onPermClientKickFromServer\n\tserverID=%llu\n"
        "\tclientChannel=%llu\n\tclientClientID=%u\n\tclientIdent=%s\n\tclientNickname=%s\n"
        "\ttoKickCount=%i\n"
        "\ttoKickClientsChannel=%llu\n\ttoKickClientsClientID=%u\n\ttoKickClientsIdent=%s\n\ttoKickClientsNickname=%s\n"
        "\treasonText=%s\n",
        (unsigned long long)serverID, (unsigned long long)client->channel, client->ID, client->ident, client->nickname, toKickCount, (unsigned long long)toKickClients->channel, toKickClients->ID, toKickClients->ident, toKickClients->nickname, reasonText);
    // check nickname for admin and deny
    if (strcmp(client->nickname, "admin") == 0) {
        return ERROR_permissions;
    }
    return ERROR_ok;
}

/*
 * Callback triggered before one or more clients will be moved.
 *
 * Parameters:
 *   serverID           - ID of the virtual server on which the client is requesting the client move.
 *   client             - Struct of client parameters like ident, nickname etc. who is causing the move. Please view public_definitions.h.
 *   toMoveCount        - The number of clients to be moved.
 *   toMoveClients      - Array of structs which clients are going to be moved.
 *   newChannel         - ID of the new channel.
 *   reasonText         - The reason why the move was initiated.
 *
 * Note: You can deny the permission so moving is not allowed.
 */
unsigned int onPermClientMove(uint64 serverID, const struct ClientMiniExport* client, int toMoveCount, const struct ClientMiniExport* toMoveClients, uint64 newChannel, const char* reasonText) {
    printf("onPermClientMove\n\tserverID=%llu\n"
        "\tclientChannel=%llu\n\tclientClientID=%u\n\tclientIdent=%s\n\tclientNickname=%s\n"
        "\ttoMoveCount=%i\n"
        "\ttoMoveClientsChannel=%llu\n\ttoMoveClientsClientID=%u\n\ttoMoveClientsIdent=%s\n\ttoMoveClientsNickname=%s\n"
        "\tnewChannel=%llu\n\treasonText=%s\n",
        (unsigned long long)serverID, (unsigned long long)client->channel, client->ID, client->ident, client->nickname, toMoveCount, (unsigned long long)toMoveClients->channel, toMoveClients->ID, toMoveClients->ident, toMoveClients->nickname,(unsigned long long)newChannel, reasonText);
    // check nickname for admin and deny
    if (strcmp(client->nickname, "admin") == 0) {
        return ERROR_permissions;
    }
    return ERROR_ok;
}

/*
 * Callback triggered before a channel will be moved.
 *
 * Parameters:
 *   serverID           - ID of the virtual server on which the client is requesting the channel move.
 *   client             - Struct of client parameters like ident, nickname etc. who is causing the move. Please view public_definitions.h.
 *   channelID          - ID of the current channel.
 *   newParentChannelID - ID to which parent channel the current channel is going to be moved.
 *
 * Note: You can deny the permission so moving is not allowed.
 */
unsigned int onPermChannelMove(uint64 serverID, const struct ClientMiniExport* client, uint64 channelID, uint64 newParentChannelID) {
    printf("onPermChannelMove\n\tserverID=%llu\n"
        "\tclientChannel=%llu\n\tclientClientID=%u\n\tclientIdent=%s\n\tclientNickname=%s\n"
        "\tchannelID=%llu\n\tnewParentChannelID=%llu\n",
        (unsigned long long)serverID, (unsigned long long)client->channel, client->ID, client->ident, client->nickname,(unsigned long long)channelID, (unsigned long long)newParentChannelID);
    // check nickname for admin and deny
    if (strcmp(client->nickname, "admin") == 0) {
        return ERROR_permissions;
    }
    return ERROR_ok;
}

/*
 * Callback triggered before message will be sent.
 *
 * Parameters:
 *   serverID              - ID of the virtual server on which the client is requesting to sent a textmessage.
 *   client                - Struct of client parameters like ident, nickname etc. who is sending the textmessage. Please view public_definitions.h.
 *   targetMode            - The text message target mode if it is a server, channel or client message.
 *   targetClientOrChannel - ID of the target client or the target channel.
 *   textMessage           - The sent text message.
 *
 * Note: You can deny the permission so sending a text message is not allowed.
 */
unsigned int onPermSendTextMessage(uint64 serverID, const struct ClientMiniExport* client, anyID targetMode, uint64 targetClientOrChannel, const char* textMessage) {
    printf("onPermSendTextMessage\n\tserverID=%llu\n"
        "\tclientChannel=%llu\n\tclientClientID=%u\n\tclientIdent=%s\n\tclientNickname=%s\n"
        "\ttargetMode=%u\n\ttargetClientOrChannel=%llu\n"
        "\ttextMessage=%s\n",
        (unsigned long long)serverID, (unsigned long long)client->channel, client->ID, client->ident, client->nickname, targetMode, (unsigned long long)targetClientOrChannel, textMessage);
    // check nickname for admin and deny
    if (strcmp(client->nickname, "admin") == 0) {
        return ERROR_permissions;
    }
    return ERROR_ok;
}

/*
 * Callback triggered before server connection info will be sent.
 *
 * Parameters:
 *   serverID              - ID of the virtual server on which the client is requesting to sent a textmessage.
 *   client                - Struct of client parameters like ident, nickname etc. who is requesting the connection info. Please view public_definitions.h.
 *
 * Note: You can deny the permission so requesting the connection info is not allowed.
 */
unsigned int onPermServerRequestConnectionInfo(uint64 serverID, const struct ClientMiniExport* client) {
    printf("onPermServerRequestConnectionInfo\n\tserverID=%llu\n"
        "\tclientChannel=%llu\n\tclientClientID=%u\n\tclientIdent=%s\n\tclientNickname=%s\n",
        (unsigned long long)serverID, (unsigned long long)client->channel, client->ID, client->ident, client->nickname);
    // check nickname for admin and deny
    if (strcmp(client->nickname, "admin") == 0) {
        return ERROR_permissions;
    }
    return ERROR_ok;
}

/*
 * Callback triggered before client connection info will be sent.
 *
 * Parameters:
 *   serverID      - ID of the virtual server on which the client is requesting to sent a textmessage.
 *   client        - Struct of client parameters like ident, nickname etc. who is requesting the connection info. Please view public_definitions.h.
 *   mayViewIpPort - Change it to 0 for decline and 1 to allow viewing ip and port. Default is allow. Also, this param is ignored if client==targetClient
 *   targetClient  - Struct of target client parameters like ident, nickname etc. Please view public_definitions.h.
 *
 * Note: You can deny the permission so requesting the connection info is not allowed.
 */
unsigned int onPermSendConnectionInfo(uint64 serverID, const struct ClientMiniExport* client, int* mayViewIpPort, const struct ClientMiniExport* targetClient) {
    *mayViewIpPort = 1;
    printf("onPermSendConnectionInfo\n\tserverID=%llu\n"
        "\tclientChannel=%llu\n\tclientClientID=%u\n\tclientIdent=%s\n\tclientNickname=%s\n"
        "\ttargetClientChannel=%llu\n\ttargetClientClientID=%u\n\ttargetClientIdent=%s\n\ttargetClientNickname=%s\n",
        (unsigned long long)serverID, (unsigned long long)client->channel, client->ID, client->ident, client->nickname, (unsigned long long)targetClient->channel, targetClient->ID, targetClient->ident, targetClient->nickname);
    // check nickname for admin and deny
    if (strcmp(client->nickname, "admin") == 0) {
        return ERROR_permissions;
    }
    return ERROR_ok;
}

/*
 * Callback triggered before channel will be edited.
 *
 * Parameters:
 *   serverID  - ID of the virtual server on which the client is requesting to sent a textmessage.
 *   client    - Struct of client parameters like ident, nickname etc. who is requesting the channel edit. Please view public_definitions.h.
 *   channelID - ID of the channel which is going to be edited.
 *   variables - Array of channel properties.
 *
 * Note: You can deny the permission so editing the channel is not allowed.
 */
unsigned int onPermChannelEdit(uint64 serverID, const struct ClientMiniExport* client, uint64 channelID, const struct VariablesExport* variables) {
    int i=0;
    printf("onPermChannelEdit\n\tserverID=%llu\n\tchannelID=%llu\n"
        "\tclientChannel=%llu\n\tclientClientID=%u\n\tclientIdent=%s\n\tclientNickname=%s\n",
        (unsigned long long)serverID, (unsigned long long)channelID, (unsigned long long)client->channel, client->ID, client->ident, client->nickname);

    for(; i<CHANNEL_ENDMARKER; ++i) {
        struct VariablesExportItem item = variables->items[i];
        if (item.itemIsValid) {
            printf("\titem=%i itemIsValid=%i current=%s\n", i, item.itemIsValid, item.current);
            if (item.proposedIsSet) {
                printf("\titem=%i proposedIsSet=%i proposed=%s\n", i, item.proposedIsSet, item.proposed);
                // check channel name for admin and deny
                if (i == CHANNEL_NAME && strcmp(item.proposed, "admin") == 0) {
                    return ERROR_permissions;
                }
            }
        }
    }
    return ERROR_ok;
}

/*
 * Callback triggered before channel will be deleted.
 *
 * Parameters:
 *   serverID  - ID of the virtual server on which the client is requesting to sent a textmessage.
 *   client    - Struct of client parameters like ident, nickname etc. who is requesting the channel edit. Please view public_definitions.h.
 *   channelID - ID of the channel which is going to be edited.
 *
 * Note: You can deny the permission so deleting the channel is not allowed.
 */
unsigned int onPermChannelDelete(uint64 serverID, const struct ClientMiniExport* client, uint64 channelID) {
    printf("onPermChannelDelete\n\tserverID=%llu\n"
        "\tclientChannel=%llu\n\tclientClientID=%u\n\tclientIdent=%s\n\tclientNickname=%s\n"
        "\tchannelID=%llu\n",
        (unsigned long long)serverID, (unsigned long long)client->channel, client->ID, client->ident, client->nickname,(unsigned long long)channelID);
    // check nickname for admin and deny
    if (strcmp(client->nickname, "admin") == 0) {
        return ERROR_permissions;
    }
    return ERROR_ok;
}

/*
 * Callback triggered before channel will be subscribed.
 *
 * Parameters:
 *   serverID  - ID of the virtual server on which the client is requesting to sent a textmessage.
 *   client    - Struct of client parameters like ident, nickname etc. who is requesting the channel edit. Please view public_definitions.h.
 *   channelID - ID of the channel which is going to be edited.
 *
 * Note: You can deny the permission so subscribing the channel is not allowed.
 */
unsigned int onPermChannelSubscribe(uint64 serverID, const struct ClientMiniExport* client, uint64 channelID) {
    printf("onPermChannelSubscribe\n\tserverID=%llu\n"
        "\tclientChannel=%llu\n\tclientClientID=%u\n\tclientIdent=%s\n\tclientNickname=%s\n"
        "\tchannelID=%llu\n",
        (unsigned long long)serverID, (unsigned long long)client->channel, client->ID, client->ident, client->nickname,(unsigned long long)channelID);
    // check nickname for admin and deny
    if (strcmp(client->nickname, "admin") == 0) {
        return ERROR_permissions;
    }
    return ERROR_ok;
}

/*
 * Read server key from file
 */
int readKeyPairFromFile(const char *fileName, char *keyPair) {
    FILE *file;

    file = fopen(fileName, "r");
    if(file == NULL) {
        printf("Could not open file '%s' for reading keypair\n", fileName);
        return -1;
    }

    fgets(keyPair, BUFSIZ, file);
    if(ferror(file) != 0) {
        fclose (file);
        printf("Error reading keypair from file '%s'.\n", fileName);
        return -1;
    }
    fclose (file);

    printf("Read keypair '%s' from file '%s'.\n", keyPair, fileName);
    return 0;
}

/*
 * Write server key to file
 */
int writeKeyPairToFile(const char *fileName, const char* keyPair) {
    FILE *file;

    file = fopen(fileName, "w");
    if(file == NULL) {
        printf("Could not open file '%s' for writing keypair\n", fileName);
        return -1;
    }

    fputs(keyPair, file);
    if(ferror(file) != 0) {
        fclose (file);
        printf("Error writing keypair to file '%s'.\n", fileName);
        return -1;
    }
    fclose (file);

    printf("Wrote keypair '%s' to file '%s'.\n", keyPair, fileName);
    return 0;
}

int main(int argc, char **argv) {
    char *version;
    uint64 serverID;
    unsigned int error;
    char buffer[BUFSIZ] = { 0 };
    char filename[BUFSIZ];
    char port_str[20];
    char *keyPair;

    /* Create struct for callback function pointers */
    struct ServerLibFunctions funcs;

    /* Initialize all callbacks with NULL */
    memset(&funcs, 0, sizeof(struct ServerLibFunctions));

    /* Now assign the used callback function pointers */
    //funcs.onClientConnected         = onClientConnected;
    //funcs.onClientDisconnected      = onClientDisconnected;
    //funcs.onClientMoved             = onClientMoved;
    //funcs.onChannelCreated          = onChannelCreated;
    //funcs.onChannelEdited           = onChannelEdited;
    //funcs.onChannelDeleted          = onChannelDeleted;
    //funcs.onUserLoggingMessageEvent = onUserLoggingMessageEvent;
    //funcs.onClientStartTalkingEvent = onClientStartTalkingEvent;
    //funcs.onClientStopTalkingEvent  = onClientStopTalkingEvent;
    //funcs.onAccountingErrorEvent    = onAccountingErrorEvent;

    funcs.permClientCanConnect               = onPermClientCanConnect;
    funcs.permClientCanGetChannelDescription = onPermClientCanGetChannelDescription;
    funcs.permClientUpdate                   = onPermClientUpdate;
    funcs.permClientKickFromChannel          = onPermClientKickFromChannel;
    funcs.permClientKickFromServer           = onPermClientKickFromServer;
    funcs.permClientMove                     = onPermClientMove;
    funcs.permChannelMove                    = onPermChannelMove;
    funcs.permSendTextMessage                = onPermSendTextMessage;
    funcs.permSendConnectionInfo             = onPermSendConnectionInfo;
    funcs.permServerRequestConnectionInfo    = onPermServerRequestConnectionInfo;
    funcs.permChannelCreate                  = onPermChannelCreate;
    funcs.permChannelEdit                    = onPermChannelEdit;
    funcs.permChannelDelete                  = onPermChannelDelete;
    funcs.permChannelSubscribe               = onPermChannelSubscribe;

    /* Initialize server lib with callbacks */
    if((error = ts3server_initServerLib(&funcs, LogType_FILE | LogType_CONSOLE | LogType_USERLOGGING, NULL, argc, (const char* const *)argv)) != ERROR_ok) {
        char* errormsg;
        if(ts3server_getGlobalErrorMessage(error, &errormsg) == ERROR_ok) {
            printf("Error initialzing serverlib: %s\n", errormsg);
            ts3server_freeMemory(errormsg);
        }
        return 1;
    }

    /* Query and print server lib version */
    if((error = ts3server_getServerLibVersion(&version)) != ERROR_ok) {
        printf("Error querying server lib version: %d\n", error);
        return 1;
    }
    printf("Server lib version: %s\n", version);
    ts3server_freeMemory(version);  /* Release dynamically allocated memory */

    /* Attempt to load keypair from file */
    /* Assemble filename: keypair_<port>.txt */
    strcpy(filename, "keypair_");
    sprintf(port_str, "%d", 9987);  // Default port
    strcat(filename, port_str);
    strcat(filename, ".txt");

    /* Try reading keyPair from file */
    if(readKeyPairFromFile(filename, buffer) == 0) {
        keyPair = buffer;  /* Id read from file */
    } else {
        keyPair = "";  /* No Id saved, start virtual server with empty keyPair string */
    }

    /* Create virtual server using default port 9987 with max 10 slots */

    /* Create the virtual server with specified port, name, keyPair and max clients */
    printf("Create virtual server using keypair '%s'\n", keyPair);
    //listen on any address on ipv4 and ipv6 (it is also possible to enter multiple ipv4 and ipv6 addresses here)
    if((error = ts3server_createVirtualServer(9987, "0.0.0.0, ::", "TeamSpeak SDK Testserver", keyPair, 8, &serverID)) != ERROR_ok) {
        char* errormsg;
        if(ts3server_getGlobalErrorMessage(error, &errormsg) == ERROR_ok) {
            printf("Error creating virtual server: %s (%d)\n", errormsg, error);
            ts3server_freeMemory(errormsg);
        }
        return 1;
    }

    /* If we didn't load the keyPair before, query it from virtual server and save to file */
    if(!*buffer) {
        if((error = ts3server_getVirtualServerKeyPair(serverID, &keyPair)) != ERROR_ok) {
            char* errormsg;
            if(ts3server_getGlobalErrorMessage(error, &errormsg) == ERROR_ok) {
                printf("Error querying keyPair: %s\n\n", errormsg);
                ts3server_freeMemory(errormsg);
            }
            return 0;
        }

        /* Save keyPair to file "keypair_<port>.txt"*/
        if(writeKeyPairToFile(filename, keyPair) != 0) {
            ts3server_freeMemory(keyPair);
            return 0;
        }
        ts3server_freeMemory(keyPair);
    }

    /* Set welcome message */
    if((error = ts3server_setVirtualServerVariableAsString(serverID, VIRTUALSERVER_WELCOMEMESSAGE, "Hello TeamSpeak")) != ERROR_ok) {
        printf("Error setting server welcomemessage: %d\n", error);
        return 1;
    }

    /* Set server password */
    if((error = ts3server_setVirtualServerVariableAsString(serverID, VIRTUALSERVER_PASSWORD, "secret")) != ERROR_ok) {
        printf("Error setting server password: %d\n", error);
        return 1;
    }

    /* Flush above two changes to server */
    if((error = ts3server_flushVirtualServerVariable(serverID)) != ERROR_ok) {
        printf("Error flushing server variables: %d\n", error);
        return 1;
    }

    /* Set codec quality of channel(s) */
    uint64* channelList = NULL;
    if ((error = ts3server_getChannelList(serverID, &channelList)) != ERROR_ok) {
        printf("Couldn't get channel list: %d\n", error);
        return 1;
    }
    uint64* channelIDPtr = NULL;
    for (channelIDPtr = channelList; *channelIDPtr != (uint64)NULL; ++channelIDPtr) {
        if ((error = ts3server_setChannelVariableAsInt(serverID, *channelIDPtr, CHANNEL_CODEC_QUALITY, 10)) != ERROR_ok) {
            printf("Couldn't set channel codec quality: %d\n", error);
            ts3server_freeMemory(channelList);
            return 1;
        }
        if ((error = ts3server_flushChannelVariable(serverID, *channelIDPtr)) != ERROR_ok) {
            if (error != ERROR_ok_no_update) {
                printf("Error flushing channel variables: %d\n", error);
                ts3server_freeMemory(channelList);
                return 1;
            }
        }
    }
    ts3server_freeMemory(channelList);

    /* Wait for user input */
    printf("\n--- Press Return to shutdown server and exit ---\n");
    getchar();

    /* Stop virtual server */
    if((error = ts3server_stopVirtualServer(serverID)) != ERROR_ok) {
        printf("Error stopping virtual server: %d\n", error);
        return 1;
    }

    /* Shutdown server lib */
    if((error = ts3server_destroyServerLib()) != ERROR_ok) {
        printf("Error destroying server lib: %d\n", error);
        return 1;
    }

    return 0;
}
