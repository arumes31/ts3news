package clientquery

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestSendChannelMessageRequiresCurrentChannel verifies that channel messages
// are only sent while the controlled client is in the requested channel, since
// ClientQuery's targetmode=2 ignores the target and uses the current channel.
func TestSendChannelMessageRequiresCurrentChannel(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer func() { _ = l.Close() }()

	var mu sync.Mutex
	var sent []string
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = fmt.Fprint(conn, "error id=0 msg=ok\n")
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case line == "whoami":
				_, _ = fmt.Fprint(conn, "clid=5 cid=3\nerror id=0 msg=ok\n")
			case strings.HasPrefix(line, "sendtextmessage"):
				mu.Lock()
				sent = append(sent, line)
				mu.Unlock()
				_, _ = fmt.Fprint(conn, "error id=0 msg=ok\n")
			default:
				_, _ = fmt.Fprint(conn, "error id=0 msg=ok\n")
			}
		}
	}()

	client, err := Dial(l.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.SendChannelMessage(4, "elsewhere"); !errors.Is(err, ErrNotInChannel) {
		t.Fatalf("SendChannelMessage(other channel) error = %v, want ErrNotInChannel", err)
	}
	if err := client.SendChannelMessage(3, "here"); err != nil {
		t.Fatalf("SendChannelMessage(current channel) error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(sent) != 1 || !strings.Contains(sent[0], "target=3") {
		t.Fatalf("sent = %q, want exactly one message to channel 3", sent)
	}
}

func TestClientDBIDForUIDRejectsReassignedClientID(t *testing.T) {
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
		_, _ = fmt.Fprint(conn, "error id=0 msg=ok\n")
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			if strings.HasPrefix(scanner.Text(), "clientvariable clid=7 ") {
				_, _ = fmt.Fprint(conn, `clid=7 client_unique_identifier=current\/uid= client_database_id=42`+"\n")
			}
			_, _ = fmt.Fprint(conn, "error id=0 msg=ok\n")
		}
	}()
	client, err := Dial(l.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer func() { _ = client.Close() }()

	if dbid, err := client.ClientDBIDForUID(7, "current/uid="); err != nil || dbid != 42 {
		t.Fatalf("ClientDBIDForUID(matching) = %d, %v; want 42, nil", dbid, err)
	}
	if _, err := client.ClientDBIDForUID(7, "original-uid"); !errors.Is(err, ErrIdentityMismatch) {
		t.Fatalf("ClientDBIDForUID(reassigned) error = %v, want ErrIdentityMismatch", err)
	}
}
