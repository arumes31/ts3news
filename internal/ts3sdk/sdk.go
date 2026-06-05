package ts3sdk

/*
#cgo CFLAGS: -I${SRCDIR}/../../lib/ts3sdk/client_sdk/linux-x86_64/include
#cgo LDFLAGS: -L${SRCDIR}/../../lib/ts3sdk/client_sdk/linux-x86_64/lib -lteamspeak_sdk_client
#include "ts3_wrapper.c"
int init_ts3();
int spawn_connection_handler();
char* create_identity();
void free_sdk_memory(void* ptr);
int start_connection(int serverConnectionHandlerID, const char* identity, const char* host, int port, const char* nickname, const char* serverPassword);
int poke_client(int serverConnectionHandlerID, int clientID, const char* message);
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type EventHandler func(eventName string, handlerID uint64, newStatus uint64, errorNumber uint64)

var globalHandler EventHandler

//export GoCallbackHandler
func GoCallbackHandler(eventName *C.char, handlerID C.ulonglong, newStatus C.ulonglong, errorNumber C.ulonglong) {
	if globalHandler != nil {
		globalHandler(C.GoString(eventName), uint64(handlerID), uint64(newStatus), uint64(errorNumber))
	}
}

func Init(handler EventHandler) error {
	globalHandler = handler
	res := C.init_ts3()
	if res != 0 {
		return fmt.Errorf("failed to init SDK, code: %d", res)
	}
	return nil
}

func SpawnConnectionHandler() (int, error) {
	id := C.spawn_connection_handler()
	if id == -1 {
		return -1, fmt.Errorf("failed to spawn connection handler")
	}
	return int(id), nil
}

// CreateIdentity asks the SDK to generate a fresh client identity string.
func CreateIdentity() (string, error) {
	c := C.create_identity()
	if c == nil {
		return "", fmt.Errorf("failed to create identity")
	}
	defer C.free_sdk_memory(unsafe.Pointer(c))
	return C.GoString(c), nil
}

func Connect(handlerID int, identity string, host string, port int, nickname string, serverPassword string) error {
	cIdentity := C.CString(identity)
	cHost := C.CString(host)
	cNickname := C.CString(nickname)
	cPassword := C.CString(serverPassword)
	defer C.free(unsafe.Pointer(cIdentity))
	defer C.free(unsafe.Pointer(cHost))
	defer C.free(unsafe.Pointer(cNickname))
	defer C.free(unsafe.Pointer(cPassword))

	res := C.start_connection(C.int(handlerID), cIdentity, cHost, C.int(port), cNickname, cPassword)
	if res != 0 {
		return fmt.Errorf("failed to connect, code: %d", res)
	}
	return nil
}

func PokeClient(handlerID int, clientID int, message string) error {
	cMessage := C.CString(message)
	defer C.free(unsafe.Pointer(cMessage))

	res := C.poke_client(C.int(handlerID), C.int(clientID), cMessage)
	if res != 0 {
		return fmt.Errorf("failed to poke client, code: %d", res)
	}
	return nil
}
