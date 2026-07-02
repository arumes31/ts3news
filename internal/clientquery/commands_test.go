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
	defer func() { _ = l.Close() }()

	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		
		_, _ = fmt.Fprint(conn, "error id=0 msg=ok\n") // Greeting drain
		
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case line == "clientlist -uid":
				_, _ = fmt.Fprint(conn, "clid=1 cid=1 client_nickname=Daniel client_type=0 client_unique_identifier=abc123uniqueid=\nerror id=0 msg=ok\n")
			case strings.HasPrefix(line, "clientinfo clid=1"):
				_, _ = fmt.Fprint(conn, "client_connected_time=12345\nerror id=0 msg=ok\n")
			case line == "servergrouplist":
				_, _ = fmt.Fprint(conn, "sgid=1 name=Drifter\\sI type=1 iconid=123\nerror id=0 msg=ok\n")
			default:
				_, _ = fmt.Fprint(conn, "error id=0 msg=ok\n")
			}
		}
	}()

	client, err := Dial(l.Addr().String(), 1*time.Second)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer func() { _ = client.Close() }()

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

// TestIconWireNameUnsigned verifies that icon file names on the wire use the
// UNSIGNED CRC32 (/icon_<uint32>), not the signed int32 form. A high-CRC icon
// (id >= 2^31) previously got a negative name, so the server stored it under a
// different name than IconExists queried — the icon then rendered as broken.
func TestIconWireNameUnsigned(t *testing.T) {
	// Payload whose CRC32 lands in the high half (>= 2^31), so signed != unsigned.
	var data []byte
	var id uint32
	for i := 0; i < 4096; i++ {
		data = []byte(fmt.Sprintf("iconpayload-%d", i))
		if IconID(data) >= 1<<31 {
			id = IconID(data)
			break
		}
	}
	if id == 0 {
		t.Fatal("could not find a payload with a high-half CRC32")
	}
	wantName := fmt.Sprintf("/icon_%d", id) // unsigned decimal, no minus sign

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer func() { _ = l.Close() }()

	lines := make(chan string, 8)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = fmt.Fprint(conn, "error id=0 msg=ok\n") // greeting drain
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "ftgetfileinfo") || strings.HasPrefix(line, "ftinitupload") {
				lines <- line
			}
			// Report the icon as absent so UploadIcon proceeds to ftinitupload,
			// then let it fail fast (no notifystartupload) — we only assert names.
			_, _ = fmt.Fprint(conn, "error id=1 msg=fail\n")
		}
	}()

	client, err := Dial(l.Addr().String(), 1*time.Second)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer func() { _ = client.Close() }()

	go func() { _, _ = client.UploadIcon(data, "localhost") }()

	deadline := time.After(3 * time.Second)
	sawExists, sawUpload := false, false
	for !(sawExists && sawUpload) {
		select {
		case line := <-lines:
			if !strings.Contains(line, Escape(wantName)) {
				t.Errorf("wire name not unsigned:\n got: %q\nwant substring: %q", line, Escape(wantName))
			}
			if strings.Contains(line, "/icon_-") {
				t.Errorf("wire name is signed (negative): %q", line)
			}
			if strings.HasPrefix(line, "ftgetfileinfo") {
				sawExists = true
			}
			if strings.HasPrefix(line, "ftinitupload") {
				sawUpload = true
			}
		case <-deadline:
			t.Fatalf("did not observe both ftgetfileinfo and ftinitupload (exists=%v upload=%v)", sawExists, sawUpload)
		}
	}
}

// TestSetChannelNameWire verifies the exact command line sent to TeamSpeak when
// renaming a channel, including ServerQuery escaping of spaces in the new name.
func TestSetChannelNameWire(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer func() { _ = l.Close() }()

	got := make(chan string, 1)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = fmt.Fprint(conn, "error id=0 msg=ok\n") // greeting drain
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "channeledit") {
				got <- line
			}
			_, _ = fmt.Fprint(conn, "error id=0 msg=ok\n")
		}
	}()

	client, err := Dial(l.Addr().String(), 1*time.Second)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.SetChannelName(42, "Screaming Guerilla"); err != nil {
		t.Fatalf("SetChannelName failed: %v", err)
	}

	select {
	case line := <-got:
		want := `channeledit cid=42 channel_name=Screaming\sGuerilla`
		if line != want {
			t.Errorf("wire command mismatch:\n got: %q\nwant: %q", line, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no channeledit command received")
	}
}
