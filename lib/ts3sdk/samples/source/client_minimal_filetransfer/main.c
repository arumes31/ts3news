/*
 * TeamSpeak SDK client minimal sample for filetransfer
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

#include <teamspeak/public_definitions.h>
#include <teamspeak/public_errors.h>
#include <teamspeak/clientlib.h>

#define DEFAULT_VIRTUAL_SERVER 1
#define NAME_BUFSIZE 1024
#define CHANNEL_PASSWORD_BUFSIZE 1024

#ifdef _WIN32
#define SLEEP(x) Sleep(x)
#define strdup(x) _strdup(x)
#else
#define SLEEP(x) usleep(x*1000)
#endif

char* gProgramPath;

void emptyInputBuffer() {
    int c;
    while((c = getchar()) != '\n' && c != EOF);
}

uint64 enterChannelID() {
    uint64 channelID;
    int n;

    printf("\nEnter channel ID: ");
    n = scanf("%llu", (unsigned long long*)&channelID);
    emptyInputBuffer();
    if(n == 0) {
        printf("Invalid input. Please enter a number.\n\n");
        return 0;
    }
    return channelID;
}

anyID enterTransferID() {
    anyID transferID;
    int n;

    printf("\nEnter transferID: ");
    n = scanf("%hu", (anyID*)&transferID);
    emptyInputBuffer();
    if(n == 0) {
        printf("Invalid input. Please enter a number.\n\n");
        return 0;
    }
    return transferID;
}

uint64 enterBWLimit(const char* section) {
    uint64 limit;
    int n;

    printf("Enter bandwidth limit for %s (or u for unlimited): ", section);
    n = scanf("%llu", (unsigned long long*)&limit);
    emptyInputBuffer();
    if(n == 0) {
        return BANDWIDTH_LIMIT_UNLIMITED;
    }
    return limit;
}

void enterName(const char* text, char *name) {
    char *s;
    printf("\n%s: ", text);
    fgets(name, NAME_BUFSIZE, stdin);
    s = strrchr(name, '\n');
    if(s) {
        *s = '\0';
    }
}

void enterPassword(char *password) {
    char *s;
    printf("\nEnter password: ");
    fgets(password, CHANNEL_PASSWORD_BUFSIZE, stdin);
    s = strrchr(password, '\n');
    if(s) {
        *s = '\0';
    }
}

char* programPath(char* programInvocation){
    char* path;
    char* end;
    int length;
    char pathsep;

    if(programInvocation == NULL) return strdup("");

#ifdef _WIN32
    pathsep = '\\';
#else
    pathsep = '/';
#endif

    end = strrchr(programInvocation, pathsep);
    if(!end) return strdup("");

    length = (end - programInvocation)+2;
    path = (char*) malloc(length);
    strncpy(path, programInvocation, length-1);
    path[length-1] = 0;

    return path;
}

/*
 * Callback for connection status change.
 * Connection status switches through the states STATUS_DISCONNECTED, STATUS_CONNECTING, STATUS_CONNECTED and STATUS_CONNECTION_ESTABLISHED.
 *
 * Parameters:
 *   serverConnectionHandlerID - Server connection handler ID
 *   newStatus                 - New connection status, see the enum ConnectStatus in clientlib_publicdefinitions.h
 *   errorNumber               - Error code. Should be zero when connecting or actively disconnection.
 *                               Contains error state when losing connection.
 */
void onConnectStatusChangeEvent(uint64 serverConnectionHandlerID, int newStatus, unsigned int errorNumber) {
    printf("Connect status changed: %llu %d %u\n", (unsigned long long)serverConnectionHandlerID, newStatus, errorNumber);
    /* Failed to connect ? */
    if(newStatus == STATUS_DISCONNECTED && errorNumber == ERROR_failed_connection_initialisation) {
        printf("Looks like there is no server running, terminate!\n");
    }
}

/*
 * This event is called when another client starts whispering to own client. Own client can decide to accept or deny
 * receiving the whisper by adding the sending client to the whisper allow list. If not added, whispering is blocked.
 *
 * Parameters:
 *   serverConnectionHandlerID - Server connection handler ID
 *   clientID                  - ID of the whispering client
 */
void onIgnoredWhisperEvent(uint64 serverConnectionHandlerID, anyID clientID) {
    unsigned int error;

    /* Add sending client to whisper allow list so own client will hear the voice data.
     * It is sufficient to add a clientID only once, not everytime this event is called. However it won't
     * hurt to add the same clientID to the allow list repeatedly, but is is not necessary. */
    if((error = ts3client_allowWhispersFrom(serverConnectionHandlerID, clientID)) != ERROR_ok) {
        printf("Error setting client on whisper allow list: %u\n", error);
    } else {
        printf("Added client %d to whisper allow list\n", clientID);
    }
}

/*
 * This event is called when the server determines an error.
 *
 * Parameters:
 *   serverConnectionHandlerID - Server connection handler ID
 *   errorMessage              - String containing a verbose error message
 *   error                     - Error code as explained on public_errors.h
 *   returnCode                - Return code if it has been set by the Client Lib function call which caused this error event
 *   extraMessage              - Can contain additional information about the occurred error, otherwise it is an empty string
 */

void onServerErrorEvent(uint64 serverConnectionHandlerID, const char* errorMessage, unsigned int error, const char* returnCode, const char* extraMessage) {
    printf("Error for server %llu: %s %s\n", (unsigned long long)serverConnectionHandlerID, errorMessage, extraMessage ? extraMessage : "");
}

/*
 * This event is periodically called when a filetransfer is active
 *
 * Parameters:
 *   transferID                - Transfer ID for which filetransfer this event was called
 *   status                    - Filetransfer status code as explained on public_errors.h
 *   statusMessage             - String containing a verbose status message
 *   remotefileSize            - Size in bytes of the remote file
 *   scHandlerID               - Server connection handler ID
 */

void onFileTransferStatusEvent(anyID transferID, unsigned int status, const char* statusMessage, uint64 remotefileSize, uint64 scHandlerID) {
    printf("onFileTransferStatusEvent transferID: %d status: %d statusMessage: %s\n", transferID, status, statusMessage);
}

/*
 * This event is called when the fileList was requested (ts3client_requestFileList)
 *
 * Parameters:
 *   serverConnectionHandlerID - Server connection handler ID
 *   channelID                 - To which channel ID the remote file contains
 *   path                      - Path of the remote file or directory
 *   name                      - Name of the remote file or directory
 *   size                      - Size in bytes of the remote file. If it is a directory this value is 0
 *   datetime                  - Timestamp of the remote file or directory
 *   type                      - Type of the remote item (0 is a folder or 1 is a file)
 *   incompleteSize            - If the file is not completely uploaded yet, this value contains the current available size
 *   returnCode                - Return code if it has been set by the Client Lib function call which caused this error event
 */

void onFileListEvent(uint64 serverConnectionHandlerID, uint64 channelID, const char* path, const char* name, uint64 size, uint64 datetime, int type, uint64 incompletesize, const char* returnCode) {
    printf("onFileListEvent channelID: %llu  path: %s filename: %s type:%s\n", channelID, path, name, type == 1 ? "file" : "dir");
}

/*
 * This event is called when the fileList request is finished
 *
 * Parameters:
 *   serverConnectionHandlerID - Server connection handler ID
 *   channelID                 - Server has finished sending events regarding this channel ID
 *   path                      - Path of the remote directory
 */

void onFileListFinishedEvent(uint64 serverConnectionHandlerID, uint64 channelID, const char* path) {
    printf("onFileListFinishedEvent: %llu\n", (unsigned long long)channelID);
}

/*
 * This event is called when a requestFileInfo call was completed on the server
 *
 * Parameters:
 *   serverConnectionHandlerID - Server connection handler ID
 *   channelID                 - Server has finished sending events regarding this channel ID
 *   name                      - name of the file
 *   size                      - size in bytes of the file
 *   datetime                  - file date/time in unix time
 */

void onFileInfoEvent(uint64 serverConnectionHandlerID, uint64 channelID, const char* name, uint64 size, uint64 datetime){
    printf("onFileInfoEvent channelID: %llu  filename: %s size:%llu date:%llu\n", channelID, name, size, datetime);
}

/*
 * Print all channels of the given virtual server
 */
void showChannels(uint64 serverConnectionHandlerID) {
    uint64 *ids;
    int i;
    unsigned int error;

    printf("\nList of channels on virtual server %llu:\n", (unsigned long long)serverConnectionHandlerID);
    if((error = ts3client_getChannelList(serverConnectionHandlerID, &ids)) != ERROR_ok) {  /* Get array of channel IDs */
        printf("Error getting channel list: %d\n", error);
        return;
    }
    if(!ids[0]) {
        printf("No channels\n\n");
        ts3client_freeMemory(ids);
        return;
    }
    for(i=0; ids[i]; i++) {
        char* name;
        if((error = ts3client_getChannelVariableAsString(serverConnectionHandlerID, ids[i], CHANNEL_NAME, &name)) != ERROR_ok) {  /* Query channel name */
            printf("Error getting channel name: %d\n", error);
            break;
        }
        printf("%llu - %s\n", (unsigned long long)ids[i], name);
        ts3client_freeMemory(name);
    }
    printf("\n");

    ts3client_freeMemory(ids);  /* Release array */
}

void showChannelDir(uint64 serverConnectionHandlerID) {
    unsigned int error;
    uint64 channelID = enterChannelID();
    if(channelID) {
        if((error = ts3client_requestFileList(serverConnectionHandlerID, channelID, "", "/", "")) != ERROR_ok) {  /* Requesting the root directory of this channel ID */
             printf("Error getting channel dir: %d\n", error);
        }
    }
}

void uploadFile(uint64 serverConnectionHandlerID) {
    anyID transferID;
    int overwrite = 1;
    int resume = 0;
    char filename[NAME_BUFSIZE];
    uint64 channelID = enterChannelID();
    printf("Uploading from predefined path: %s\n", gProgramPath);
    enterName("Enter filename to upload (<serverPath><filename> like /testfile.txt)", filename);
    if(channelID) {
        if(ts3client_sendFile(serverConnectionHandlerID, channelID, "", filename, overwrite, resume, gProgramPath, &transferID, NULL) == ERROR_ok) {
            printf("Sending file with transferID: %d\n", transferID);
        }
    }
}

void downloadFile(uint64 serverConnectionHandlerID) {
    anyID transferID;
    int overwrite = 1;
    int resume = 0;
    uint64 channelID;
    char filename[NAME_BUFSIZE];
    channelID = enterChannelID();
    printf("Downloading in predefined path: %s\n", gProgramPath);
    enterName("Enter filename to download (<serverPath><filename> like /testfile.txt)", filename);

    if(ts3client_requestFile(serverConnectionHandlerID, channelID, "", filename, overwrite, resume, gProgramPath, &transferID, NULL) == ERROR_ok) {
        printf("Recieving file with transferID: %d\n", transferID);
    }
}

void deleteFile(uint64 serverConnectionHandlerID) {
    unsigned int error;
    uint64 channelID;
    char* files[2];
    char filename[NAME_BUFSIZE];
    channelID = enterChannelID();
    enterName("Enter filename to delete (<serverPath><filename> like /testfile.txt)", filename);

    files[0] = filename;
    files[1] = 0;

    if(channelID) {
        if((error = ts3client_requestDeleteFile(serverConnectionHandlerID, channelID, "", (const char**)files, NULL)) != ERROR_ok) {
            printf("Error deleting file: %d\n", error);
        }
    }
}

void renameFile(uint64 serverConnectionHandlerID) {
    unsigned int error;
    uint64 channelID;
    char oldName[NAME_BUFSIZE];
    char newName[NAME_BUFSIZE];

    channelID = enterChannelID();
    enterName("Enter old name (<serverPath><filename> like /testfile.txt)", oldName);
    enterName("Enter new name (<serverPath><filename> like /new_testfile.txt)", newName);

    if(channelID) {
        if((error = ts3client_requestRenameFile(serverConnectionHandlerID, channelID, "", channelID, "", oldName, newName, NULL)) != ERROR_ok) {
            printf("Error renaming file: %d\n", error);
        }
    }
}

void createDirectory(uint64 serverConnectionHandlerID) {
    unsigned int error;
    uint64 channelID;
    char dirName[NAME_BUFSIZE];

    channelID = enterChannelID();
    enterName("Enter new directory name (<serverPath> like /subdir)", dirName);

    if(channelID) {
        if((error = ts3client_requestCreateDirectory(serverConnectionHandlerID, channelID, "", dirName, NULL)) != ERROR_ok) {
            printf("Error renaming file: %d\n", error);
        }
    }
}

void fileInfo(uint64 serverConnectionHandlerID) {
    uint64 channelID;
    char filename[NAME_BUFSIZE];
    unsigned int error;

    channelID = enterChannelID();
    enterName("Enter filename to get info on (<serverPath><filename> like /testfile.txt)", filename);

    if((error=ts3client_requestFileInfo(serverConnectionHandlerID, channelID, "", filename, NULL)) != ERROR_ok) {
        printf("error getting file info: %d\n", error);
    }
}

void bandwidth(uint64 serverConnectionHandlerID) {
    unsigned int error;
    uint64 instanceUpLimit;
    uint64 instanceDownLimit;
    uint64 schUpLimit;
    uint64 schDownLimit;

    if((error=ts3client_getInstanceSpeedLimitUp(&instanceUpLimit)) != ERROR_ok){
        printf("error during ts3client_getInstanceSpeedLimitUp: %d\n", error);
        instanceUpLimit=0;
    }

    if((error=ts3client_getInstanceSpeedLimitDown(&instanceDownLimit)) != ERROR_ok){
        printf("error during ts3client_getInstanceSpeedLimitDown: %d\n", error);
        instanceDownLimit=0;
    }

    if((error=ts3client_getServerConnectionHandlerSpeedLimitUp(serverConnectionHandlerID, &schUpLimit)) != ERROR_ok){
        printf("error during ts3client_getServerConnectionHandlerSpeedLimitUp: %d\n", error);
        schUpLimit=0;
    }

    if((error=ts3client_getServerConnectionHandlerSpeedLimitDown(serverConnectionHandlerID, &schDownLimit)) != ERROR_ok){
        printf("error during ts3client_getServerConnectionHandlerSpeedLimitDown: %d\n", error);
        schDownLimit=0;
    }

    printf("current limits: instanceUp: %llu instanceDown: %llu schUp: %llu schDown: %llu\n", instanceUpLimit, instanceDownLimit, schUpLimit, schDownLimit);

    instanceUpLimit = enterBWLimit("instanceUp");
    if((error=ts3client_setInstanceSpeedLimitUp(instanceUpLimit)) != ERROR_ok){
        printf("error during ts3client_setInstanceSpeedLimitUp: %d\n", error);
    }

    instanceDownLimit = enterBWLimit("instanceDown");
    if((error=ts3client_setInstanceSpeedLimitDown(instanceDownLimit)) != ERROR_ok){
        printf("error during ts3client_setInstanceSpeedLimitDown: %d\n", error);
    }

    schUpLimit = enterBWLimit("schUp");
    if((error=ts3client_setServerConnectionHandlerSpeedLimitUp(serverConnectionHandlerID, schUpLimit)) != ERROR_ok){
        printf("error during ts3client_setServerConnectionHandlerSpeedLimitUp: %d\n", error);
    }

    schDownLimit = enterBWLimit("schDown");
    if((error=ts3client_setServerConnectionHandlerSpeedLimitDown(serverConnectionHandlerID, schDownLimit)) != ERROR_ok){
        printf("error during ts3client_setServerConnectionHandlerSpeedLimitDown: %d\n", error);
    }
}

void cancelTransfer(uint64 serverConnectionHandlerID){
    anyID transferID;
    unsigned int error;

    transferID = enterTransferID();

    if(transferID){
        if((error=ts3client_haltTransfer(serverConnectionHandlerID, transferID, 1, NULL))!=ERROR_ok){
            printf("error during ts3client_haltTransfer: %d\n", error);
        }
    }
}

void transferStats(uint64 serverConnectionHandlerID){
    unsigned int error;
    uint64 bytesRecievedBandwidth;
    uint64 bytesSentBandwidth;
    uint64 bytesRecieved;
    uint64 bytesSent;
        
    if((error=ts3client_getConnectionVariableAsUInt64(serverConnectionHandlerID, 0, CONNECTION_FILETRANSFER_BANDWIDTH_RECEIVED, &bytesRecievedBandwidth))!=ERROR_ok){
        printf("error getting bytesRecievedBandwidth: %d\n", error);
        bytesRecievedBandwidth=0;
    }

    if((error=ts3client_getConnectionVariableAsUInt64(serverConnectionHandlerID, 0, CONNECTION_FILETRANSFER_BANDWIDTH_SENT, &bytesSentBandwidth))!=ERROR_ok){
        printf("error getting bytesSentBandwidth: %d\n", error);
        bytesSentBandwidth=0;
    }

    if((error=ts3client_getConnectionVariableAsUInt64(serverConnectionHandlerID, 0, CONNECTION_FILETRANSFER_BYTES_RECEIVED_TOTAL, &bytesRecieved))!=ERROR_ok){
        printf("error getting bytesRecieved: %d\n", error);
        bytesRecieved=0;
    }

    if((error=ts3client_getConnectionVariableAsUInt64(serverConnectionHandlerID, 0, CONNECTION_FILETRANSFER_BYTES_SENT_TOTAL, &bytesSent))!=ERROR_ok){
        printf("error getting bytesSent: %d\n", error);
        bytesSent=0;
    }

    printf("Transfer statistics: BW recv: %llu BW send %llu bytes recv: %llu bytes sent: %llu\n", bytesRecievedBandwidth, bytesSentBandwidth, bytesRecieved, bytesSent);
}

void showHelp() {
    printf("\n");
    printf("[q] - Disconnect from server\n");
    printf("[h] - Show this help\n");
    printf("[c] - Show channels\n");
    printf("[s] - Show directory of a channel\n");
    printf("[u] - Upload file\n");
    printf("[d] - Download file\n");
    printf("[x] - Delete file\n");
    printf("[r] - Rename file\n");
    printf("[f] - create directory file\n");
    printf("[i] - get file information\n");
    printf("[k] - cancel transfer\n");
    printf("[l] - edit transfer bandwidth limits\n");
    printf("[j] - get connection transfer stats\n");
}

int main(int argc, char **argv) {
    uint64 scHandlerID;
    unsigned int error;
    char *version;
    char *identity;
    short abort = 0;

    /* Create struct for callback function pointers */
    struct ClientUIFunctions funcs;

    /* Initialize all callbacks with NULL */
    memset(&funcs, 0, sizeof(struct ClientUIFunctions));

    /* Callback function pointers */
    /* It is sufficient to only assign those callback functions you are using. When adding more callbacks, add those function pointers here. */
    funcs.onConnectStatusChangeEvent    = onConnectStatusChangeEvent;
    funcs.onServerErrorEvent            = onServerErrorEvent;
    funcs.onFileListEvent               = onFileListEvent;
    funcs.onFileListFinishedEvent       = onFileListFinishedEvent;
    funcs.onFileTransferStatusEvent     = onFileTransferStatusEvent; // crash when not defined!!!
    funcs.onFileInfoEvent               = onFileInfoEvent;
    funcs.onIgnoredWhisperEvent         = onIgnoredWhisperEvent;

    /* Initialize client lib with callbacks */
    /* Resource path points to the SDK\bin directory to locate the soundbackends*/
    gProgramPath = programPath(argv[0]);
    error = ts3client_initClientLib(&funcs, NULL, LogType_FILE | LogType_CONSOLE | LogType_USERLOGGING, NULL, gProgramPath);

    if(error != ERROR_ok) {
        char* errormsg;
        if(ts3client_getErrorMessage(error, &errormsg) == ERROR_ok) {
            printf("Error initialzing serverlib: %s\n", errormsg);
            ts3client_freeMemory(errormsg);
        }
        return 1;
    }

    /* Spawn a new server connection handler using the default port and store the server ID */
    if((error = ts3client_spawnNewServerConnectionHandler(0, &scHandlerID)) != ERROR_ok) {
        printf("Error spawning server connection handler: %d\n", error);
        return 1;
    }

    /* Create a new client identity */
    /* In your real application you should do this only once, store the assigned identity locally and then reuse it. */
    if((error = ts3client_createIdentity(&identity)) != ERROR_ok) {
        printf("Error creating identity: %d\n", error);
        return 1;
    }


    /* Connect to server on localhost:9987 with nickname "client", no default channel, no default channel password and server password "secret" */
    if((error = ts3client_startConnection(scHandlerID, identity, "localhost", 9987, "client", NULL, "", "secret")) != ERROR_ok) {
        printf("Error connecting to server: %d\n", error);
        return 1;
    }

    ts3client_freeMemory(identity);  /* Release dynamically allocated memory */
    identity = NULL;

    printf("Client lib initialized and running\n");

    /* Query and print client lib version */
    if((error = ts3client_getClientLibVersion(&version)) != ERROR_ok) {
        printf("Failed to get clientlib version: %d\n", error);
        return 1;
    }
    printf("Client lib version: %s\n", version);
    ts3client_freeMemory(version);  /* Release dynamically allocated memory */
    version = NULL;

    SLEEP(500);

    /* Simple commandline interface */
    printf("\nTeamSpeak 3 client commandline interface\n");
    showHelp();

    /* Wait for user input */
    while(!abort) {
        int c = getc(stdin);
        switch(c) {
        case 'q':
            printf("\nDisconnecting from server...\n");
            abort = 1;
            break;
        case 'h':
            showHelp();
            break;
        case 'c':
            showChannels(DEFAULT_VIRTUAL_SERVER);
            break;
        case 's':
            showChannelDir(DEFAULT_VIRTUAL_SERVER);
            break;
        case 'u':
            uploadFile(DEFAULT_VIRTUAL_SERVER);
            break;
        case 'x':
            deleteFile(DEFAULT_VIRTUAL_SERVER);
            break;
        case 'r':
            renameFile(DEFAULT_VIRTUAL_SERVER);
            break;
        case 'd':
            downloadFile(DEFAULT_VIRTUAL_SERVER);
            break;
        case 'f':
            createDirectory(DEFAULT_VIRTUAL_SERVER);
            break;
        case 'i':
            fileInfo(DEFAULT_VIRTUAL_SERVER);
            break;
        case 'k':
            cancelTransfer(DEFAULT_VIRTUAL_SERVER);
            break;
        case 'l':
            bandwidth(DEFAULT_VIRTUAL_SERVER);
            break;
        case 'j':
            transferStats(DEFAULT_VIRTUAL_SERVER);
            break;
        }

        SLEEP(50);
    }

    /* Disconnect from server */
    if((error = ts3client_stopConnection(scHandlerID, "leaving")) != ERROR_ok) {
        printf("Error stopping connection: %d\n", error);
        return 1;
    }

    SLEEP(200);

    /* Destroy server connection handler */
    if((error = ts3client_destroyServerConnectionHandler(scHandlerID)) != ERROR_ok) {
        printf("Error destroying clientlib: %d\n", error);
        return 1;
    }

    /* Shutdown client lib */
    if((error = ts3client_destroyClientLib()) != ERROR_ok) {
        printf("Failed to destroy clientlib: %d\n", error);
        return 1;
    }

    free(gProgramPath);
    return 0;
}
