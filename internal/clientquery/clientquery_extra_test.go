package clientquery

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestCommand(t *testing.T) {
	// Start a local server to mock ClientQuery
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
		
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			cmd := scanner.Text()
			if cmd == "testcommand" {
				fmt.Fprint(conn, "data line 1\ndata line 2\nerror id=0 msg=ok\n")
			} else if cmd == "failcommand" {
				fmt.Fprint(conn, "error id=256 msg=invalid\\sparameter\n")
			}
		}
	}()

	client, err := Dial(l.Addr().String(), 1*time.Second)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer client.Close()

	// 1. Success command
	data, err := client.Command("testcommand")
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}
	if len(data) != 2 || data[0] != "data line 1" {
		t.Errorf("got data %v, want [data line 1, data line 2]", data)
	}

	// 2. Failure command
	_, err = client.Command("failcommand")
	if err == nil {
		t.Error("expected error for failcommand")
	} else if !strings.Contains(err.Error(), "id=256") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestAuth(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	defer l.Close()
	go func() {
		conn, _ := l.Accept()
		defer conn.Close()
		fmt.Fprint(conn, "error id=0 msg=ok\n") // for greeting drain
		scanner := bufio.NewScanner(conn)
		if scanner.Scan() {
			fmt.Fprint(conn, "error id=0 msg=ok\n")
		}
	}()

	client, _ := Dial(l.Addr().String(), 1*time.Second)
	err := client.Auth("secret")
	if err != nil {
		t.Errorf("Auth failed: %v", err)
	}
}
