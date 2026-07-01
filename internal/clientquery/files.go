package clientquery

import (
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

// ServerGroup is one server group as returned by servergrouplist.
type ServerGroup struct {
	ID     int
	Name   string
	Type   int // 1 = regular server group
	IconID int
}

// firstKV parses the first data line into an unescaped key=value map.
func firstKV(data []string) map[string]string {
	m := map[string]string{}
	if len(data) == 0 {
		return m
	}
	for _, f := range strings.Fields(data[0]) {
		k, v, ok := strings.Cut(f, "=")
		if ok {
			m[k] = Unescape(v)
		}
	}
	return m
}

// RegisterServerGroupEvents subscribes to the notification events that carry the
// replies to server-group "list" commands. ClientQuery delivers servergrouplist /
// servergroupclientlist results via these notifications, so without registering
// first those commands never return (i/o timeout).
func (c *Client) RegisterServerGroupEvents() error {
	for _, ev := range []string{"notifyservergrouplist", "notifyservergroupclientlist", "notifystartupload"} {
		if _, err := c.Command("clientnotifyregister schandlerid=1 event=" + ev); err != nil {
			return err
		}
	}
	return nil
}

// ServerGroupList returns all server groups (sgid, name, type). Requires a prior
// RegisterServerGroupEvents on this connection.
func (c *Client) ServerGroupList() ([]ServerGroup, error) {
	data, err := c.Command("servergrouplist")
	if err != nil {
		return nil, err
	}
	var out []ServerGroup
	for _, line := range data {
		for _, rec := range strings.Split(line, "|") {
			sg := ServerGroup{ID: -1}
			for _, f := range strings.Fields(rec) {
				k, v, ok := strings.Cut(f, "=")
				if !ok {
					continue
				}
				switch k {
				case "sgid":
					sg.ID, _ = strconv.Atoi(v)
				case "name":
					sg.Name = Unescape(v)
				case "type":
					sg.Type, _ = strconv.Atoi(v)
				case "iconid":
					sg.IconID, _ = strconv.Atoi(v)
				}
			}
			if sg.ID >= 0 {
				out = append(out, sg)
			}
		}
	}
	return out, nil
}

// ServerGroupAdd creates a regular server group. The ClientQuery reply does not
// reliably echo the new sgid (the data arrives via the notifyservergrouplist
// event as the full list), so callers should re-list and look the group up by
// name afterwards rather than relying on a returned id.
func (c *Client) ServerGroupAdd(name string) error {
	_, err := c.Command("servergroupadd name=" + Escape(name) + " type=1")
	return err
}

// ServerGroupAddPerm sets a single permission on a server group.
func (c *Client) ServerGroupAddPerm(sgid int, permsid string, value int) error {
	_, err := c.Command(fmt.Sprintf(
		"servergroupaddperm sgid=%d permsid=%s permvalue=%d permnegated=0 permskip=0",
		sgid, permsid, value))
	return err
}

// ServerGroupDelClient removes a client (by db id) from a server group.
func (c *Client) ServerGroupDelClient(sgid, cldbid int) error {
	_, err := c.Command(fmt.Sprintf("servergroupdelclient sgid=%d cldbid=%d", sgid, cldbid))
	return err
}

// ServerGroupMemberCount returns how many clients are members of a server group.
func (c *Client) ServerGroupMemberCount(sgid int) (int, error) {
	data, err := c.Command(fmt.Sprintf("servergroupclientlist sgid=%d", sgid))
	if err != nil {
		// Empty result set => zero members.
		if strings.Contains(err.Error(), "1281") || strings.Contains(strings.ToLower(err.Error()), "empty") {
			return 0, nil
		}
		return 0, err
	}
	count := 0
	for _, line := range data {
		for _, rec := range strings.Split(line, "|") {
			if strings.Contains(rec, "cldbid=") {
				count++
			}
		}
	}
	return count, nil
}

// ServerGroupDel deletes a server group (force=1 removes it even if non-empty).
func (c *Client) ServerGroupDel(sgid int, force bool) error {
	f := 0
	if force {
		f = 1
	}
	_, err := c.Command(fmt.Sprintf("servergroupdel sgid=%d force=%d", sgid, f))
	return err
}

// IconID computes the TeamSpeak icon id (unsigned CRC32) for icon image bytes.
func IconID(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

// IconExists reports whether the icon file for id is actually present in the
// server's icon filebase (channel 0). Used to confirm an icon really landed
// before a group is pointed at it, so a lost upload can't leave a dangling
// i_icon_id (which the client renders as "icon … not found"). ftgetfileinfo
// returns the file's size when present; any error or missing size means absent.
func (c *Client) IconExists(id uint32) bool {
	name := fmt.Sprintf("/icon_%d", id)
	data, err := c.Command(fmt.Sprintf("ftgetfileinfo cid=0 cpw= name=%s", Escape(name)))
	if err != nil {
		return false
	}
	if sz, ok := firstKV(data)["size"]; ok {
		if n, _ := strconv.Atoi(sz); n > 0 {
			return true
		}
	}
	return false
}

// UploadIcon ensures a group icon (PNG bytes) is present in the server's icon
// storage and returns its icon id. hostFallback is used when the server does not
// report a transfer IP (typical for ClientQuery on the same host as the server).
//
// Rank icons are deterministic — the id is the CRC32 of the PNG — so the same icon
// is requested for every group at that rank and again on every level-up. Once a
// given icon is on the server we skip the (failure-prone) upload handshake entirely
// and reuse it; that is what keeps rank icons stable rather than re-uploading (and
// intermittently failing) each time a group is recreated. New icons are uploaded
// with a few retries, since the notifystartupload handshake is occasionally dropped
// on a busy same-host ClientQuery link and a lost handshake is transient.
func (c *Client) UploadIcon(data []byte, hostFallback string) (uint32, error) {
	id := IconID(data)
	if c.IconExists(id) {
		return id, nil // already on the server — reuse, no upload needed
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if _, err := c.uploadIconOnce(data, hostFallback); err != nil {
			lastErr = err
		} else {
			lastErr = nil
		}
		// The filebase is the source of truth. A transfer can report success yet fail
		// to persist (truncated), so confirm the file is actually there before we let
		// a group reference it — otherwise the group gets a dangling id (broken icon).
		if c.IconExists(id) {
			return id, nil
		}
		if lastErr == nil {
			lastErr = fmt.Errorf("icon %d: transfer reported success but file not on server", id)
		}
	}
	return 0, lastErr
}

// uploadIconOnce performs a single ftinitupload + file transfer. UploadIcon wraps it
// with retries and confirms the result against the filebase, so this reports only
// what the transfer replies said (the handshake is asynchronous and can be lost).
func (c *Client) uploadIconOnce(data []byte, hostFallback string) (uint32, error) {
	id := IconID(data)
	name := fmt.Sprintf("/icon_%d", id)

	// The ftinitupload command reply is empty (error id=0). The transfer key/port
	// are delivered asynchronously via a notifystartupload event, so read that.
	if _, err := c.Command(fmt.Sprintf(
		"ftinitupload clientftfid=1 name=%s cid=0 cpw= size=%d overwrite=1 resume=0",
		Escape(name), len(data))); err != nil {
		if strings.Contains(err.Error(), "2050") || strings.Contains(strings.ToLower(err.Error()), "exist") {
			return id, nil // server itself reports the file already exists
		}
		return 0, err
	}

	// Short timeout: a present notify arrives near-instantly on the local link, and a
	// lost one never comes — UploadIcon retries rather than blocking the cycle here.
	kv, ok := c.ReadNotify("notifystartupload", 2*time.Second)
	if !ok {
		return 0, fmt.Errorf("icon %d: upload handshake missing", id)
	}
	if st := kv["status"]; st != "" && st != "0" {
		// Non-zero status usually means the server already has the file; the caller's
		// IconExists check confirms it either way.
		return id, nil
	}
	key := kv["ftkey"]
	port := kv["port"]
	if key == "" || port == "" {
		return 0, fmt.Errorf("notifystartupload missing key/port: %v", kv)
	}
	host := strings.Trim(kv["ip"], ", ")
	if host == "" || host == "0.0.0.0" {
		host = hostFallback
	}

	if err := rawUpload(net.JoinHostPort(host, port), key, data); err != nil {
		return 0, fmt.Errorf("icon file transfer: %w", err)
	}
	log.Printf("icon: uploaded %d (%d bytes) to %s:%s", id, len(data), host, port)
	return id, nil
}

// rawUpload performs the TeamSpeak file-transfer handshake: connect, send the
// transfer key, then stream the payload.
func rawUpload(addr, key string, data []byte) error {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))

	if _, err := conn.Write([]byte(key)); err != nil {
		return err
	}
	if _, err := conn.Write(data); err != nil {
		return err
	}
	// Wait for the server to actually consume the payload before tearing the socket
	// down. conn.Write only means "buffered locally", not "received": closing right
	// after it can truncate the transfer, leaving the icon file unstored while the
	// group still carries the i_icon_id — which clients render as a broken icon.
	// Half-close our write side so the server sees end-of-data, then drain until it
	// closes its side (it does that once the file is stored).
	if tcp, ok := conn.(*net.TCPConn); ok {
		if err := tcp.CloseWrite(); err != nil {
			return err
		}
	}
	if _, err := io.Copy(io.Discard, conn); err != nil {
		return err
	}
	return nil
}
