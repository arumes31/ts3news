package bot

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
	"ts3news/internal/clientquery"
	"ts3news/internal/config"
	"ts3news/internal/icons"
	"ts3news/internal/leveling"
)

func TestPrestigeAssignsNewlyCreatedGroupID(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	name := prestigeGroupName(1)
	png, err := icons.Icon(1, leveling.NumTiers, leveling.NumTiers, iconSizePx)
	if err != nil {
		t.Fatal(err)
	}
	iconID := clientquery.IconID(png)
	assigned := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = fmt.Fprintln(conn, "error id=0 msg=ok")
		scanner := bufio.NewScanner(conn)
		created := false
		for scanner.Scan() {
			command := scanner.Text()
			switch {
			case strings.HasPrefix(command, "clientvariable "):
				_, _ = fmt.Fprintln(conn, "client_database_id=77")
			case command == "servergrouplist":
				if created {
					_, _ = fmt.Fprintf(conn, "sgid=42 name=%s iconid=1\n", clientquery.Escape(name))
				}
			case strings.HasPrefix(command, "servergroupadd name="):
				created = true
			case strings.HasPrefix(command, "ftgetfilelist "):
				_, _ = fmt.Fprintf(conn, "name=icon_%d size=100\n", iconID)
			case strings.HasPrefix(command, "servergroupaddclient "):
				assigned <- command
			}
			_, _ = fmt.Fprintln(conn, "error id=0 msg=ok")
		}
	}()
	client, err := clientquery.Dial(listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()
	(&Bot{Cfg: &config.Config{}}).applyPrestigeGroup(client, 5, "uid", "Player", 1)
	select {
	case command := <-assigned:
		if command != "servergroupaddclient sgid=42 cldbid=77" {
			t.Fatalf("assignment=%s", command)
		}
	case <-time.After(time.Second):
		t.Fatal("no assignment")
	}
}
