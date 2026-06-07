package clientquery

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestCommands(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		
		fmt.Fprint(conn, "error id=0 msg=ok\n") // Greeting drain
		
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case line == "clientlist -uid":
				fmt.Fprint(conn, "clid=1 cid=1 client_nickname=Daniel client_type=0 client_unique_identifier=abc123uniqueid=\nerror id=0 msg=ok\n")
			case strings.HasPrefix(line, "clientinfo clid=1"):
				fmt.Fprint(conn, "client_connected_time=12345\nerror id=0 msg=ok\n")
			case line == "servergrouplist":
				fmt.Fprint(conn, "sgid=1 name=Drifter\\sI type=1 iconid=123\nerror id=0 msg=ok\n")
			default:
				fmt.Fprint(conn, "error id=0 msg=ok\n")
			}
		}
	}()

	client, err := Dial(l.Addr().String(), 1*time.Second)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer client.Close()

	// clientlist
	cls, err := client.ClientList()
	if err != nil {
		t.Errorf("ClientList failed: %v", err)
	}
	if len(cls) != 1 || cls[0].Nickname != "Daniel" {
		t.Errorf("got clients %v", cls)
	}

	// servergrouplist
	sgs, err := client.ServerGroupList()
	if err != nil {
		t.Errorf("ServerGroupList failed: %v", err)
	}
	if len(sgs) != 1 || sgs[0].Name != "Drifter I" {
		t.Errorf("got groups %v", sgs)
	}

	// SendPrivateMessage
	_ = client.SendPrivateMessage(1, "hello")

	// Poke
	_ = client.Poke(1, "wake up")

	// ServerGroupAdd
	_ = client.ServerGroupAdd("New Group")

	// ServerGroupDel
	_ = client.ServerGroupDel(1, true)

	// AddServerGroup
	_ = client.AddServerGroup(1, 1)

	// ServerGroupDelClient
	_ = client.ServerGroupDelClient(1, 1)

	// SetChannelDescription
	_ = client.SetChannelDescription(1, "New Desc")

	// SetNickname
	_ = client.SetNickname("NewNick")

	// UploadIcon
	_, _ = client.UploadIcon([]byte("pngdata"), "localhost")

	// WhoAmI
	_, _ = client.WhoAmI()

	// IsConnected
	_ = client.IsConnected()

	// Disconnect
	_ = client.Disconnect()
}
