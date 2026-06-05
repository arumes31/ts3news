/*
 * TeamSpeak SDK server sample
 *
 * Copyright (c) TeamSpeak-Systems
 */

//#define MINIMAL_EXAMPLE

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

#ifndef MINIMAL_EXAMPLE

char FILE_BASE[] = "sdk_files";

/* Small helper function to compare string endings */
int endsWith(const char *str, const char *suffix)
{
    size_t lenstr;
    size_t lensuffix;

    if (!str || !suffix) return 0;
    lenstr = strlen(str);
    lensuffix = strlen(suffix);
    if (lensuffix >  lenstr) return 0;
    return strncmp(str + lenstr - lensuffix, suffix, lensuffix) == 0;
}


   /*
    * Callback triggered when a file transfer status changes
    *
    * Parameters:
    *   data          - The paramaters of the file transfer
    */

void onFileTransferEvent(const struct FileTransferCallbackExport* data){

    printf("onFileTransferEvent clientID: %hu, transferID: %hu, remoteTransferID: %hu, status: %u, msg: %s, remoteFileSize: %llu, bytes: %llu, isSender: %i\n",
        data->clientID, data->transferID, data->remoteTransferID, data->status, data->statusMessage, data->remotefileSize, data->bytes, data->isSender);

}

   /*
    * Callback triggered before a client tries to upload a file
    *
    * Parameters:
    *   serverID        - ID of the virtual server on which upload is requested.
    *   client          - Struct of client parameters like ident, nickname etc. who is uploading the file. See public_definitions.h.
    *   params          - The paramaters of the file upload. See server_commands_file_transfer.h
    *
    * Note: You can deny the upload by returning ERROR_permissions
    */
unsigned int permFileTransferInitUpload(uint64 serverID, const struct ClientMiniExport* client, const struct ts3sc_ftinitupload* params){

    /*log to screen*/
    printf("permFileTransferInitUpload filename: %s, size: %llu, channel: %llu, overwrite: %d, resume: %d\n", params->d.fileName, params->d.fileSize, params->d.channelID, params->d.overwrite, params->d.resume);

    /*just for fun, lets not permit uploading of .txt files*/
    if (endsWith(params->d.fileName, ".txt")){
        return ERROR_permissions;
    }

    /*note we also have the client parameter, so we could deny based on who is uploading*/

    return ERROR_ok;
}

   /*
    * Callback triggered before a client tries to upload a file
    *
    * Parameters:
    *   serverID        - ID of the virtual server on which upload is requested.
    *   client          - Struct of client parameters like ident, nickname etc. who is uploading the file. See public_definitions.h.
    *   params          - The paramaters of the file download. See server_commands_file_transfer.h
    *
    * Note: You can deny the download by returning ERROR_permissions
    */
unsigned int permFileTransferInitDownload(uint64 serverID, const struct ClientMiniExport* client, const struct ts3sc_ftinitdownload* params){
    /*log to screen*/
    printf("permFileTransferInitDownload filename: %s, channel: %llu\n", params->d.fileName, params->d.channelID);

    /*just for fun, lets not permit downloading of .txt files*/
    if (endsWith(params->d.fileName, ".txt")){
        return ERROR_permissions;
    }

    return ERROR_ok;
}

   /*
    * Callback triggered before a client tries to get file information
    *
    * Parameters:
    *   serverID        - ID of the virtual server on which upload is requested.
    *   client          - Struct of client parameters like ident, nickname etc. who is uploading the file. See public_definitions.h.
    *   params          - The paramaters of the information request. See server_commands_file_transfer.h
    *
    * Note: You can deny the whole request returning ERROR_permissions
    */
unsigned int permFileTransferGetFileInfo(uint64 serverID, const struct ClientMiniExport* client, const struct ts3sc_ftgetfileinfo* params){
    int idx;

    /*log to screen*/
    printf("permFileTransferGetFileInfo \n");
    for (idx= 0; idx < params->r_size; ++idx){
        printf("  channel: %llu name:%s\n", params->r[idx].channelID, params->r[idx].fileName);
    }
    printf("\n");

    return ERROR_ok;
}

   /*
    * Callback triggered before a client tries to get a directory listing
    *
    * Parameters:
    *   serverID        - ID of the virtual server on which upload is requested.
    *   client          - Struct of client parameters like ident, nickname etc. who is uploading the file. See public_definitions.h.
    *   params          - The paramaters of the information request. See server_commands_file_transfer.h
    *
    * Note: You can deny the whole request returning ERROR_permissions
    */
unsigned int permFileTransferGetFileList(uint64 serverID, const struct ClientMiniExport* client, const struct ts3sc_ftgetfilelist* params){
    /*log to screen*/
    printf("permFileTransferGetFileList path: %s, channel: %llu\n", params->d.path, params->d.channelID);

    return ERROR_ok;
}

   /*
    * Callback triggered before a client tries to delete files
    *
    * Parameters:
    *   serverID        - ID of the virtual server on which upload is requested.
    *   client          - Struct of client parameters like ident, nickname etc. who is uploading the file. See public_definitions.h.
    *   params          - The paramaters of the delete request. See server_commands_file_transfer.h
    *
    * Note: You can deny the whole request returning ERROR_permissions
    */
unsigned int permFileTransferDeleteFile(uint64 serverID, const struct ClientMiniExport* client, const struct ts3sc_ftdeletefile* params){
    int idx;

    /*log to screen*/
    printf("permFileTransferDeleteFile channel: %llu\n", params->d.channelID);
    for (idx= 0; idx < params->r_size; ++idx){
        printf("  name:%s\n", params->r[idx].fileName);
    
        /*just for fun, lets not permit deleting of .txt files*/
        if (endsWith(params->r[idx].fileName, ".txt")){
            return ERROR_permissions;
        }
    }
    printf("\n");

    return ERROR_ok;
}

   /*
    * Callback triggered before a client tries to create a sub directory
    *
    * Parameters:
    *   serverID        - ID of the virtual server on which upload is requested.
    *   client          - Struct of client parameters like ident, nickname etc. who is uploading the file. See public_definitions.h.
    *   params          - The paramaters of the request. See server_commands_file_transfer.h
    *
    * Note: You can deny the whole request returning ERROR_permissions
    */
unsigned int permFileTransferCreateDirectory(uint64 serverID, const struct ClientMiniExport* client, const struct ts3sc_ftcreatedir* params){
    /*log to screen*/
    printf("permFileTransferCreateDirectory dirname: %s, channel: %llu\n", params->d.dirname, params->d.channelID);

    /*just for fun, lets not permit creation of .txt dirs*/
    if (endsWith(params->d.dirname, ".txt")){
        return ERROR_permissions;
    }

    return ERROR_ok;
}

   /*
    * Callback triggered before a client tries to rename a file
    *
    * Parameters:
    *   serverID        - ID of the virtual server on which upload is requested.
    *   client          - Struct of client parameters like ident, nickname etc. who is uploading the file. See public_definitions.h.
    *   params          - The paramaters of the rename request. See server_commands_file_transfer.h
    *
    * Note: You can deny the whole request returning ERROR_permissions
    */
unsigned int permFileTransferRenameFile(uint64 serverID, const struct ClientMiniExport* client, const struct ts3sc_ftrenamefile* params){
    /*log to screen*/
    if (params->m.has_toChannelID){
        printf("permFileTransferRenameFile oldname: %s, old channel: %llu new name: %s, new channel: %llu\n", params->d.oldFileName, params->d.fromChannelID, params->d.newFileName, params->d.toChannelID);
    } else {
        printf("permFileTransferRenameFile oldname: %s, old channel: %llu new name: %s\n", params->d.oldFileName, params->d.fromChannelID, params->d.newFileName);
    }

    return ERROR_ok;
}

   /*
    * Callback triggered after most file transfer request permissions have passed. It exists to let the server change the file
    * name and/or directory of the request for special purposes.
    *
    * Parameters:
    *   serverID        - ID of the virtual server on which upload is requested.
    *   invokerClientID - id of the client that is doing the request
    *   original        - Struct of the request params like original file name, what kind of action is taken and . See public_definitions.h.
    *   result          - Struct with the transformed parameters. See public_definitions.h.
    */
unsigned int onTransformFilePath(uint64 serverID, anyID invokerClientID, const struct TransformFilePathExport* original, struct TransformFilePathExportReturns* result){

    const char* action;
    size_t filenamelen;
    size_t appendlen = strlen(".example");

    switch(original->action){
    case(FT_INIT_SERVER):
        action = "FT_INIT_SERVER";
        break;
    case(FT_INIT_CHANNEL):
        action = "FT_INIT_CHANNEL";
        break;
    case(FT_UPLOAD):
        action = "FT_UPLOAD";
        break;
    case(FT_DOWNLOAD):
        action = "FT_DOWNLOAD";
        break;
    case(FT_DELETE):
        action = "FT_DELETE";
        break;
    case(FT_CREATEDIR):
        action = "FT_CREATEDIR";
        break;
    case(FT_RENAME):
        action = "FT_RENAME";
        break;
    case(FT_FILELIST):
        action = "FT_FILELIST";
        break;
    case(FT_FILEINFO):
        action = "FT_FILELIST";
        break;
    default:
        action = "unknown action";
    }

    printf("onTransformFilePath filename: %s, action: %s\n", original->filename, action);

    /*for FT_INIT_SERVER and FT_INIT_CHANNEL we use our own directory naming for the server/channel*/
    if (original->action == FT_INIT_SERVER){
        sprintf(result->channelPath, "%s/myvs_%llu", FILE_BASE, (unsigned long long)serverID);
        return ERROR_ok;
    }
    if (original->action == FT_INIT_CHANNEL){
        sprintf(result->channelPath, "%s/myvs_%llu/mychan_%llu", FILE_BASE, serverID, original->channel);
        return ERROR_ok;
    }

    /* Here we can alter the filename and file path to the data on the server. The default values are already filled in in the result variable.
     *
     * For this example we will append ".example" to .doc files
     */

    if (endsWith(original->filename, ".doc")){
        filenamelen = strlen(original->filename);
        if ((int)(filenamelen+appendlen) >= original->transformedFileNameMaxSize) {
            /* the filename we want to return is larger than allowed. We return an error*/
            return ERROR_parameter_invalid_size;
        }
        sprintf(result->transformedFileName, "%s.example", original->filename);
    }

    return ERROR_ok;
}


#endif
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
    funcs.onUserLoggingMessageEvent        = onUserLoggingMessageEvent;
    funcs.onAccountingErrorEvent           = onAccountingErrorEvent;
#ifndef MINIMAL_EXAMPLE
    funcs.permFileTransferInitUpload       = permFileTransferInitUpload;
    funcs.permFileTransferInitDownload     = permFileTransferInitDownload;
    funcs.permFileTransferGetFileInfo      = permFileTransferGetFileInfo;
    funcs.permFileTransferGetFileList      = permFileTransferGetFileList;
    funcs.permFileTransferDeleteFile       = permFileTransferDeleteFile;
    funcs.permFileTransferCreateDirectory  = permFileTransferCreateDirectory;
    funcs.permFileTransferRenameFile       = permFileTransferRenameFile;
    funcs.onFileTransferEvent              = onFileTransferEvent;
    funcs.onTransformFilePath              = onTransformFilePath;
#endif

    /* Initialize server lib with callbacks */
    if((error = ts3server_initServerLib(&funcs, LogType_FILE | LogType_CONSOLE | LogType_USERLOGGING, NULL, argc, (const char* const *)argv)) != ERROR_ok) {
        char* errormsg;
        if(ts3server_getGlobalErrorMessage(error, &errormsg) == ERROR_ok) {
            printf("Error initialzing serverlib: %s\n", errormsg);
            ts3server_freeMemory(errormsg);
        }
        return 1;
    }

    /* The call below is the only call that needs to be made to enable file transfers on the server.
       There are how ever a lot of callbacks that can be hooked in to (above) to change
       permissions or change file names */

    /* Here we initialize file transfers to store everything on disk in a tree starting at "sdk_files".
       All files for virtual server with id 1 will be in "sdk_files/virtualserver_1"
       Files in channel 1 in virtual server 1 will be in "sdk_files/virtualserver_1/channel_1"
       These directories will automatically be created by the server if they do not exist. It is the 
       responsibility of the application (not serverlib) to delete these directories when you are done
       with them.
       
       We supply NULL to the ips param. This is equivalent to:
       char* ips[2]= {"0.0.0.0", NULL}; 
       If the system is ipv6 capable it is equivalent to:
       char* ips[3]= {"0.0.0.0", "::", NULL};

       The port is free to choose. TeamSpeak defaults to 30033. Feel free to listen on an other port.

       Finally we set no limits for the download and upload bandwidth.
       */

    /* Initialize server file transfers */
    if ((error=ts3server_enableFileManager(FILE_BASE, NULL, 30033, BANDWIDTH_LIMIT_UNLIMITED, BANDWIDTH_LIMIT_UNLIMITED)) != ERROR_ok){
        char* errormsg;
        if(ts3server_getGlobalErrorMessage(error, &errormsg) == ERROR_ok) {
            printf("Error initialzing filemanager: %s\n", errormsg);
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
        printf("Error setting server welcome message: %d\n", error);
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
