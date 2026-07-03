package clientquery

import (
	"errors"
	"fmt"
	"hash/crc32"
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
	for _, ev := range []string{"notifyservergrouplist", "notifyservergroupclientlist", "notifychannelgrouplist", "notifystartupload", "notifyfilelist"} {
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

// IconExists reports whether a non-empty icon file for id is present in the server's
// icon filebase (the /icons/ subfolder of channel 0). It confirms an icon really
// landed before a group is pointed at it, so a lost or truncated upload can't leave a
// dangling i_icon_id (which the client renders as "icon … not found").
//
// The check is made against the directory listing (ftgetfilelist, via IconFileList),
// NOT ftgetfileinfo. On the live server ftgetfileinfo returns a false "not found" for
// icons that are demonstrably present — they are listed by ftgetfilelist and reported
// as already-existing (error 2050) by ftinitupload — which made every upload look like
// it had failed and left groups without their icon. Listing the folder is the query
// that actually works here.
//
// A size of 0 counts as absent, so a truncated 0-byte upload is treated as
// not-yet-present and retried rather than referenced as a broken icon.
func (c *Client) IconExists(id uint32) bool {
	files, err := c.IconFileList()
	if err != nil {
		return false
	}
	for _, f := range files {
		if f.ID == id {
			return f.Size > 0
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
		// Wait a tiny bit and check multiple times to allow async file writing on remote hosts.
		var exists bool
		for checkAttempt := 0; checkAttempt < 5; checkAttempt++ {
			if c.IconExists(id) {
				exists = true
				break
			}
			time.Sleep(150 * time.Millisecond)
		}
		if exists {
			return id, nil
		}
		if lastErr == nil {
			lastErr = fmt.Errorf("icon %d: transfer reported success but file not on server", id)
		}
	}
	// Diagnostic: show what the filebase actually holds so we can tell "nothing
	// landed" (transfer rejected) from "landed under an unexpected name".
	if files, ferr := c.IconFileList(); ferr == nil {
		present := false
		var thisSize int64
		names := make([]string, 0, 8)
		for _, f := range files {
			if f.ID == id {
				present = true
				thisSize = f.Size
			}
			if len(names) < 8 {
				names = append(names, f.Name)
			}
		}
		log.Printf("icon %d: filebase has %d icon(s), thisPresent=%v thisSize=%d, sample=%v", id, len(files), present, thisSize, names)
	} else {
		log.Printf("icon %d: filebase list after failed upload errored: %v", id, ferr)
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

	// The notify arrives near-instantly on a direct local link, but can take
	// several seconds when the ClientQuery is running inside a Docker container
	// relaying through the TS3 client to a remote server. 5 s is long enough for
	// the relay round-trip while still bounding each attempt tightly enough that
	// a lost handshake (which never arrives) doesn't stall the cycle for long.
	// UploadIcon retries on timeout rather than blocking indefinitely here.
	kv, ok := c.ReadNotify("notifystartupload", 5*time.Second)
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
		// An empty transfer IP means "use the server you're connected to". The client
		// relays the upload to the voice server's file-transfer port, so fall back to
		// the configured TS3 host (config.env TS3_HOST); nothing listens locally.
		host = hostFallback
	}
	log.Printf("icon: ft handshake %d: status=%q ftkey.len=%d port=%s ip=%q seekpos=%q proto=%q host=%s",
		id, kv["status"], len(key), port, kv["ip"], kv["seekpos"], kv["proto"], host)

	if err := rawUpload(net.JoinHostPort(host, port), key, data); err != nil {
		return 0, fmt.Errorf("icon file transfer: %w", err)
	}
	log.Printf("icon: uploaded %d (%d bytes) to %s:%s", id, len(data), host, port)
	return id, nil
}

// rawUpload performs the TeamSpeak file-transfer handshake: connect, send the
// transfer key, then send the payload. The key and payload are written separately —
// the sequence the official client uses.
//
// NOTE: on the live server this currently persists a 0-byte file regardless of the
// transfer framing (single vs. split writes, read-to-EOF or immediate close, remote
// vs. local target were all verified). The server accepts the connection and reads
// the bytes but does not commit them, so the fault is server-side file transfer, not
// this handshake. IconExists (size > 0) is what surfaces the failure instead of
// letting a broken 0-byte icon be referenced.
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
	return nil
}

// errEmptyResultSet is TeamSpeak's "database empty result set" error id, returned
// by list commands (ftgetfilelist, channelgrouplist, …) when there is simply
// nothing to list. For icon cleanup that is a normal, non-fatal outcome.
const errEmptyResultSet = 1281

// isEmptyResult reports whether err is the benign "empty result set" reply.
func isEmptyResult(err error) bool {
	var ce *CommandError
	return errors.As(err, &ce) && ce.ID == errEmptyResultSet
}

// eachRecord splits a set of reply lines into individual "|"-separated records and
// calls fn with each record's parsed (unescaped) key=value fields.
func eachRecord(data []string, fn func(map[string]string)) {
	for _, line := range data {
		for _, rec := range strings.Split(line, "|") {
			m := map[string]string{}
			for _, f := range strings.Fields(rec) {
				if k, v, ok := strings.Cut(f, "="); ok {
					m[k] = Unescape(v)
				}
			}
			if len(m) > 0 {
				fn(m)
			}
		}
	}
}

// IconFile is one icon file present in the server's icon filebase.
type IconFile struct {
	Name string // filebase name, e.g. "icon_2168812048"
	ID   uint32 // the unsigned icon id parsed from the name
	Size int64  // file size in bytes (0 = truncated/empty upload)
}

// iconIDFromSigned normalises a signed icon value to the UNSIGNED form used by the
// filebase icon file names. The i_icon_id permission and the negative-named files
// left by the old signed-name bug carry the id as a signed int32, while the file
// names use the unsigned CRC32. Masking off the low 32 bits maps both the signed and
// unsigned spellings of the same 32 bits onto the one unsigned id — it is an explicit
// bit-reinterpretation, not a lossy narrowing, and the mask bounds the value to the
// uint32 range so the final conversion is provably in range (no unchecked truncation).
func iconIDFromSigned(v int64) uint32 {
	return uint32(v & 0xFFFFFFFF)
}

// IconFileList returns every icon file in the server's icon filebase. TeamSpeak
// stores icons in the "/icons/" subfolder of channel 0 (the root only holds avatars
// and the icons folder itself), so that is the path listed here. Names are parsed
// leniently: both the correct unsigned form ("icon_2168812048") and the negative
// form left by the old signed-name bug ("icon_-2126155248") map to the same unsigned
// id, so historical broken uploads are still matched and cleaned.
func (c *Client) IconFileList() ([]IconFile, error) {
	data, err := c.Command("ftgetfilelist cid=0 cpw= path=/icons/")
	if err != nil {
		if isEmptyResult(err) {
			return nil, nil // no files stored yet
		}
		return nil, err
	}
	var out []IconFile
	eachRecord(data, func(m map[string]string) {
		name, ok := m["name"]
		if !ok || !strings.HasPrefix(name, "icon_") {
			return
		}
		n, perr := strconv.ParseInt(strings.TrimPrefix(name, "icon_"), 10, 64)
		if perr != nil {
			return
		}
		var size int64
		if s, ok := m["size"]; ok {
			size, _ = strconv.ParseInt(s, 10, 64)
		}
		out = append(out, IconFile{Name: name, ID: iconIDFromSigned(n), Size: size})
	})
	return out, nil
}

// DeleteFile removes a file from the icon filebase (channel 0) by its name
// (e.g. "icon_2168812048"). A missing file is treated as success.
func (c *Client) DeleteFile(name string) error {
	_, err := c.Command("ftdeletefile cid=0 cpw= name=" + Escape("/"+name))
	if err != nil && isEmptyResult(err) {
		return nil
	}
	return err
}

// ReferencedIconIDs collects every icon id currently in use anywhere on the server,
// as the UNSIGNED filebase id: server groups, channel groups, channels and the
// virtual server itself. It is the "keep" set for icon garbage collection.
//
// Any query failure (other than an empty result set) is returned as an error so the
// caller can skip deletion entirely — deleting with an incomplete reference set
// risks removing an icon that is actually still in use.
func (c *Client) ReferencedIconIDs() (map[uint32]struct{}, error) {
	refs := map[uint32]struct{}{}
	add := func(v int) {
		// The i_icon_id permission stores the id as a signed int32; the filebase names
		// it unsigned. Normalise to the unsigned form used by the file names.
		if u := iconIDFromSigned(int64(v)); u != 0 {
			refs[u] = struct{}{}
		}
	}

	sgs, err := c.ServerGroupList()
	if err != nil {
		return nil, fmt.Errorf("servergrouplist: %w", err)
	}
	for _, g := range sgs {
		add(g.IconID)
	}

	for _, spec := range []struct{ cmd, key string }{
		{"channelgrouplist", "iconid"},
		{"channellist -icon", "channel_icon_id"},
	} {
		ids, err := c.recordIconIDs(spec.cmd, spec.key)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", firstWord(spec.cmd), err)
		}
		for _, v := range ids {
			add(v)
		}
	}

	sv, err := c.serverIconID()
	if err != nil {
		return nil, fmt.Errorf("serverinfo: %w", err)
	}
	add(sv)

	return refs, nil
}

// recordIconIDs runs a list command and extracts the integer icon id under key from
// every record. An empty result set yields no ids (not an error).
func (c *Client) recordIconIDs(cmd, key string) ([]int, error) {
	data, err := c.Command(cmd)
	if err != nil {
		if isEmptyResult(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []int
	eachRecord(data, func(m map[string]string) {
		if v, ok := m[key]; ok {
			if n, perr := strconv.Atoi(v); perr == nil {
				out = append(out, n)
			}
		}
	})
	return out, nil
}

// serverIconID returns the virtual server's own icon id (0 if none). A missing key
// is treated as "no icon"; only a command error is propagated. ClientQuery has no
// "serverinfo" command, so the value is read from the cached server variable.
func (c *Client) serverIconID() (int, error) {
	data, err := c.Command("servervariable virtualserver_icon_id")
	if err != nil {
		return 0, err
	}
	if v, ok := firstKV(data)["virtualserver_icon_id"]; ok {
		n, _ := strconv.Atoi(v)
		return n, nil
	}
	return 0, nil
}
