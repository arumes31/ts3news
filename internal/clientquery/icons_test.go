package clientquery

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// startMockTS3 spins up a fake ClientQuery endpoint that answers a fixed set of
// commands from replies, then returns a connected Client. capture (if non-nil)
// receives every command line the client sends.
func startMockTS3(t *testing.T, replies map[string]string, capture chan<- string) *Client {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

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
			if capture != nil {
				capture <- line
			}
			if body, ok := replies[line]; ok && body != "" {
				_, _ = fmt.Fprint(conn, body+"\n")
			}
			_, _ = fmt.Fprint(conn, "error id=0 msg=ok\n")
		}
	}()

	c, err := Dial(l.Addr().String(), 1*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestIconFileListParsesSignedAndUnsignedNames(t *testing.T) {
	c := startMockTS3(t, map[string]string{
		// High-CRC unsigned name, a small orphan, a legacy negative-named file, and a
		// non-icon file that must be ignored.
		"ftgetfilelist cid=0 cpw= path=/icons/": `cid=0 path=\/ name=icon_2168812048 size=348 datetime=1 type=1|name=icon_100 size=100 datetime=1 type=1|name=icon_-64882566 size=50 datetime=1 type=1|name=avatar_abc size=9 type=1`,
	}, nil)

	files, err := c.IconFileList()
	if err != nil {
		t.Fatalf("IconFileList: %v", err)
	}
	got := map[string]uint32{}
	for _, f := range files {
		got[f.Name] = f.ID
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 icon files (avatar ignored), got %d: %v", len(got), got)
	}
	if got["icon_2168812048"] != 2168812048 {
		t.Errorf("unsigned name id = %d, want 2168812048", got["icon_2168812048"])
	}
	if got["icon_100"] != 100 {
		t.Errorf("small id = %d, want 100", got["icon_100"])
	}
	// uint32(int32(-64882566)) == 4230084730 — the legacy negative name maps to the
	// same unsigned id the file would carry under the fixed naming.
	if got["icon_-64882566"] != 4230084730 {
		t.Errorf("legacy negative name id = %d, want 4230084730", got["icon_-64882566"])
	}
}

func TestReferencedIconIDsAcrossAllSources(t *testing.T) {
	c := startMockTS3(t, map[string]string{
		// Group icon stored as the SIGNED int32 (negative) — must normalise to the
		// unsigned filebase id 2168812048. iconid=0 must be excluded.
		"servergrouplist":                      `sgid=1 name=Peasant\sI type=1 iconid=-2126155248|sgid=2 name=Guest type=1 iconid=0`,
		"channelgrouplist":                     `cgid=1 name=Channel\sAdmin type=0 iconid=0|cgid=2 name=Op type=0 iconid=500`,
		"channellist -icon":                    `cid=1 pid=0 channel_name=Default channel_icon_id=0|cid=2 channel_name=Foo channel_icon_id=777`,
		"servervariable virtualserver_icon_id": `virtualserver_icon_id=0`,
	}, nil)

	refs, err := c.ReferencedIconIDs()
	if err != nil {
		t.Fatalf("ReferencedIconIDs: %v", err)
	}
	want := []uint32{2168812048, 500, 777}
	for _, w := range want {
		if _, ok := refs[w]; !ok {
			t.Errorf("reference set missing %d; have %v", w, refs)
		}
	}
	if _, ok := refs[0]; ok {
		t.Errorf("reference set must not contain 0 (no icon)")
	}
	if len(refs) != len(want) {
		t.Errorf("reference set size = %d, want %d: %v", len(refs), len(want), refs)
	}
}

// TestReferencedIconIDsFailsClosed verifies that a failure to enumerate one source
// aborts the whole reference gathering, so the caller never deletes on partial data.
func TestReferencedIconIDsFailsClosed(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })
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
			case line == "servergrouplist":
				_, _ = fmt.Fprint(conn, "sgid=1 name=A type=1 iconid=10\nerror id=0 msg=ok\n")
			case line == "channelgrouplist":
				// Hard error (not an empty-result set) — must be fatal.
				_, _ = fmt.Fprint(conn, "error id=512 msg=command\\snot\\sfound\n")
			default:
				_, _ = fmt.Fprint(conn, "error id=0 msg=ok\n")
			}
		}
	}()
	c, err := Dial(l.Addr().String(), 1*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	if _, err := c.ReferencedIconIDs(); err == nil {
		t.Fatal("expected error when a reference source fails, got nil")
	}
}

func TestDeleteFileWire(t *testing.T) {
	cap := make(chan string, 8)
	c := startMockTS3(t, map[string]string{}, cap)

	if err := c.DeleteFile("icon_2168812048"); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	want := `ftdeletefile cid=0 cpw= name=\/icon_2168812048`
	deadline := time.After(2 * time.Second)
	for {
		select {
		case line := <-cap:
			if strings.HasPrefix(line, "ftdeletefile") {
				if line != want {
					t.Fatalf("delete wire:\n got: %q\nwant: %q", line, want)
				}
				return
			}
		case <-deadline:
			t.Fatal("no ftdeletefile command observed")
		}
	}
}

// TestIconGCSelection ties it together: only filebase icons absent from the
// reference set are selected for deletion.
func TestIconGCSelection(t *testing.T) {
	c := startMockTS3(t, map[string]string{
		"ftgetfilelist cid=0 cpw= path=/icons/": `cid=0 path=\/ name=icon_2168812048 size=1 type=1|name=icon_100 size=1 type=1|name=icon_-64882566 size=1 type=1`,
		"servergrouplist":                       `sgid=1 name=Peasant\sI type=1 iconid=-2126155248`,
		"channelgrouplist":                      `cgid=2 name=Op type=0 iconid=500`,
		"channellist -icon":                     `cid=2 channel_name=Foo channel_icon_id=777`,
		"servervariable virtualserver_icon_id":  `virtualserver_icon_id=0`,
	}, nil)

	files, err := c.IconFileList()
	if err != nil {
		t.Fatalf("IconFileList: %v", err)
	}
	refs, err := c.ReferencedIconIDs()
	if err != nil {
		t.Fatalf("ReferencedIconIDs: %v", err)
	}
	var del []string
	for _, f := range files {
		if _, used := refs[f.ID]; !used {
			del = append(del, f.Name)
		}
	}
	want := map[string]bool{"icon_100": true, "icon_-64882566": true}
	if len(del) != len(want) {
		t.Fatalf("delete set = %v, want keys %v", del, want)
	}
	for _, name := range del {
		if !want[name] {
			t.Errorf("unexpected deletion of %s (should be kept)", name)
		}
	}
}
