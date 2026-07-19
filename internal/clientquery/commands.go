package clientquery

import (
	"fmt"
	"strconv"
	"strings"
)

// TeamSpeak ServerQuery/ClientQuery string escaping.
var escaper = strings.NewReplacer(
	"\\", "\\\\",
	"/", "\\/",
	" ", "\\s",
	"|", "\\p",
	"\a", "\\a",
	"\b", "\\b",
	"\f", "\\f",
	"\n", "\\n",
	"\r", "\\r",
	"\t", "\\t",
	"\v", "\\v",
)

var unescaper = strings.NewReplacer(
	"\\\\", "\\",
	"\\/", "/",
	"\\s", " ",
	"\\p", "|",
	"\\a", "\a",
	"\\b", "\b",
	"\\f", "\f",
	"\\n", "\n",
	"\\r", "\r",
	"\\t", "\t",
	"\\v", "\v",
)

// Escape encodes a string for safe inclusion in a ClientQuery command argument.
func Escape(s string) string { return escaper.Replace(s) }

// Unescape decodes a ClientQuery-escaped string back to its raw form.
func Unescape(s string) string { return unescaper.Replace(s) }

// ClientInfo describes one client visible on the server.
type ClientInfo struct {
	CLID            int
	Nickname        string
	UID             string // client_unique_identifier (stable TeamSpeak identity id)
	Type            int    // 0 = normal voice client, 1 = ServerQuery/ClientQuery client
	CID             int    // channel id
	ConnectedTimeMS int64  // client_connected_time (ms since session start)
	IdleTimeMS      int64  // client_idle_time (ms since last input or >1s of voice)
	Talking         bool   // client_flag_talking, when reported by clientinfo
}

// ClientList returns all clients on the currently selected virtual server,
// including each client's unique identifier (-uid).
func (c *Client) ClientList() ([]ClientInfo, error) {
	data, err := c.Command("clientlist -uid")
	if err != nil {
		return nil, err
	}
	var out []ClientInfo
	for _, line := range data {
		// Records are separated by "|", fields by spaces as key=value pairs.
		for _, rec := range strings.Split(line, "|") {
			ci := ClientInfo{CLID: -1, CID: -1}
			for _, field := range strings.Fields(rec) {
				k, v, ok := strings.Cut(field, "=")
				if !ok {
					continue
				}
				switch k {
				case "clid":
					ci.CLID, _ = strconv.Atoi(v)
				case "cid":
					ci.CID, _ = strconv.Atoi(v)
				case "client_nickname":
					ci.Nickname = Unescape(v)
				case "client_unique_identifier":
					ci.UID = Unescape(v)
				case "client_type":
					ci.Type, _ = strconv.Atoi(v)
				}
			}

			// For voice clients, fetch additional session stats (connected time,
			// idle time, and talk flag — the latter two power the Idely subsystem).
			if ci.CLID >= 0 && ci.Type == 0 {
				if info, ierr := c.Command(fmt.Sprintf("clientinfo clid=%d", ci.CLID)); ierr == nil {
					for _, iline := range info {
						for _, f := range strings.Fields(iline) {
							switch {
							case strings.HasPrefix(f, "client_connected_time="):
								ci.ConnectedTimeMS, _ = strconv.ParseInt(f[len("client_connected_time="):], 10, 64)
							case strings.HasPrefix(f, "client_idle_time="):
								ci.IdleTimeMS, _ = strconv.ParseInt(f[len("client_idle_time="):], 10, 64)
							case strings.HasPrefix(f, "client_flag_talking="):
								ci.Talking = f[len("client_flag_talking="):] == "1"
							}
						}
					}
				}
			}

			if ci.CLID >= 0 {
				out = append(out, ci)
			}
		}
	}
	return out, nil
}

// ClientListBasic returns the clients on the current server using only the
// fields available directly from "clientlist" (clid, cid, uid, nickname, type).
// Unlike ClientList it issues no per-client "clientinfo" — that command does not
// exist over ClientQuery, and the fields it would fill (idle/connected time) are
// not synced to ClientQuery for remote clients anyway. Idely uses this and
// measures idleness itself from voice-activity events.
func (c *Client) ClientListBasic() ([]ClientInfo, error) {
	data, err := c.Command("clientlist -uid")
	if err != nil {
		return nil, err
	}
	var out []ClientInfo
	for _, line := range data {
		for _, rec := range strings.Split(line, "|") {
			ci := ClientInfo{CLID: -1, CID: -1}
			for _, field := range strings.Fields(rec) {
				k, v, ok := strings.Cut(field, "=")
				if !ok {
					continue
				}
				switch k {
				case "clid":
					ci.CLID, _ = strconv.Atoi(v)
				case "cid":
					ci.CID, _ = strconv.Atoi(v)
				case "client_nickname":
					ci.Nickname = Unescape(v)
				case "client_unique_identifier":
					ci.UID = Unescape(v)
				case "client_type":
					ci.Type, _ = strconv.Atoi(v)
				}
			}
			if ci.CLID >= 0 {
				out = append(out, ci)
			}
		}
	}
	return out, nil
}

// Use selects a server connection handler (the connected server is id 1).
func (c *Client) Use(schandlerID int) error {
	_, err := c.Command("use " + strconv.Itoa(schandlerID))
	return err
}

// Poke sends a poke (a pop-up notification) to a client.
func (c *Client) Poke(clid int, message string) error {
	_, err := c.Command("clientpoke clid=" + strconv.Itoa(clid) + " msg=" + Escape(message))
	return err
}

// SendPrivateMessage sends a private text message to a single client.
func (c *Client) SendPrivateMessage(clid int, message string) error {
	_, err := c.Command("sendtextmessage targetmode=1 target=" + strconv.Itoa(clid) + " msg=" + Escape(message))
	return err
}

// WhoAmI returns the raw whoami fields (useful to confirm we are connected).
func (c *Client) WhoAmI() ([]string, error) {
	return c.Command("whoami")
}

// WhoAmIInfo is the parsed result of a "whoami" query.
type WhoAmIInfo struct {
	CLID   int    // own client id on the current server (0 = not connected)
	CID    int    // current channel id
	Status string // virtualserver_status, e.g. "connected" / "connecting" (if reported)
}

// WhoAmIInfo runs "whoami" and parses the connection identity. A non-nil error
// means the command itself failed (e.g. "not connected to a server" → id!=0).
func (c *Client) WhoAmIInfo() (WhoAmIInfo, error) {
	data, err := c.WhoAmI()
	if err != nil {
		return WhoAmIInfo{}, err
	}
	var info WhoAmIInfo
	for _, line := range data {
		for _, f := range strings.Fields(line) {
			k, v, ok := strings.Cut(f, "=")
			if !ok {
				continue
			}
			switch k {
			case "clid":
				info.CLID, _ = strconv.Atoi(v)
			case "cid":
				info.CID, _ = strconv.Atoi(v)
			case "virtualserver_status":
				info.Status = v
			}
		}
	}
	return info, nil
}

// IsConnected reports whether the client is currently connected to a server
// (whoami returns a non-zero own client id).
func (c *Client) IsConnected() bool {
	info, err := c.WhoAmIInfo()
	if err != nil {
		return false // "currently not possible" => not connected
	}
	return info.CLID > 0
}

// Disconnect disconnects the client from the current server (it stays running).
func (c *Client) Disconnect() error {
	_, err := c.Command("disconnect")
	return err
}

// SetNickname changes the bot's own nickname on the connected server.
func (c *Client) SetNickname(nickname string) error {
	_, err := c.Command("clientupdate client_nickname=" + Escape(nickname))
	return err
}

// SetDescription updates the client's description field.
func (c *Client) SetDescription(description string) error {
	_, err := c.Command("clientupdate client_description=" + Escape(description))
	return err
}

// SetChannelDescription updates a channel's description field.
func (c *Client) SetChannelDescription(cid int, description string) error {
	_, err := c.Command(fmt.Sprintf("channeledit cid=%d channel_description=%s", cid, Escape(description)))
	return err
}

// SetChannelName updates a channel's name.
func (c *Client) SetChannelName(cid int, name string) error {
	_, err := c.Command(fmt.Sprintf("channeledit cid=%d channel_name=%s", cid, Escape(name)))
	return err
}

// ChannelDetail describes one channel on the server.
type ChannelDetail struct {
	CID       int
	Name      string
	IsDefault bool // channel_flag_default=1 (the "home" channel new clients join)
}

// ChannelList returns every channel on the current server with its name and
// default-channel flag (via "channellist -flags").
func (c *Client) ChannelList() ([]ChannelDetail, error) {
	data, err := c.Command("channellist -flags")
	if err != nil {
		return nil, err
	}
	var out []ChannelDetail
	for _, line := range data {
		for _, rec := range strings.Split(line, "|") {
			cd := ChannelDetail{CID: -1}
			for _, field := range strings.Fields(rec) {
				k, v, ok := strings.Cut(field, "=")
				if !ok {
					continue
				}
				switch k {
				case "cid":
					cd.CID, _ = strconv.Atoi(v)
				case "channel_name":
					cd.Name = Unescape(v)
				case "channel_flag_default":
					cd.IsDefault = v == "1"
				}
			}
			if cd.CID >= 0 {
				out = append(out, cd)
			}
		}
	}
	return out, nil
}

// ClientDBID returns the server-side database id (cldbid) for a connected client,
// needed for server-group operations.
func (c *Client) ClientDBID(clid int) (int, error) {
	data, err := c.Command("clientvariable clid=" + strconv.Itoa(clid) + " client_database_id")
	if err != nil {
		return 0, err
	}
	for _, line := range data {
		for _, f := range strings.Fields(line) {
			if v, ok := strings.CutPrefix(f, "client_database_id="); ok {
				return strconv.Atoi(v)
			}
		}
	}
	return 0, fmt.Errorf("client_database_id not found for clid=%d", clid)
}

// ClientUID returns the stable unique identifier (client_unique_identifier) for a
// connected client. Idely uses it to resolve its own identity so it doesn't count
// itself as a user when scanning for idle channels.
func (c *Client) ClientUID(clid int) (string, error) {
	data, err := c.Command("clientvariable clid=" + strconv.Itoa(clid) + " client_unique_identifier")
	if err != nil {
		return "", err
	}
	for _, line := range data {
		for _, f := range strings.Fields(line) {
			if v, ok := strings.CutPrefix(f, "client_unique_identifier="); ok {
				return Unescape(v), nil
			}
		}
	}
	return "", fmt.Errorf("client_unique_identifier not found for clid=%d", clid)
}

// ClientIdleTime returns the remote client's idle time in milliseconds.
func (c *Client) ClientIdleTime(clid int) (int64, error) {
	data, err := c.Command(fmt.Sprintf("clientvariable clid=%d client_idle_time", clid))
	if err != nil {
		return 0, err
	}
	for _, line := range data {
		for _, f := range strings.Fields(line) {
			if v, ok := strings.CutPrefix(f, "client_idle_time="); ok {
				return strconv.ParseInt(v, 10, 64)
			}
		}
	}
	return 0, fmt.Errorf("client_idle_time not found for clid=%d", clid)
}

// AddServerGroup adds a client (by database id) to a server group. Requires the
// bot's identity to hold the necessary group-management permission.
func (c *Client) AddServerGroup(sgid, cldbid int) error {
	_, err := c.Command(fmt.Sprintf("servergroupaddclient sgid=%d cldbid=%d", sgid, cldbid))
	return err
}

// RegisterTalkStatusEvents subscribes to notifytalkstatuschange events on server
// connection handler 1. After this, the client emits an unsolicited
// "notifytalkstatuschange status=<0|1> ... clid=<n>" line whenever a client
// starts (status=1) or stops (status=0) talking; read them via the Reader.
// This event is available only over ClientQuery (a connected voice client), not
// ServerQuery, which is why Idely runs its own TeamSpeak client.
func (c *Client) RegisterTalkStatusEvents() error {
	_, err := c.Command("clientnotifyregister schandlerid=1 event=notifytalkstatuschange")
	return err
}

// MoveClient moves the client with the given clid into channel cid. The Idely
// client uses this (with its own clid from whoami) to follow the audio bot into
// the channel it is serenading, so it can keep watching for talk activity there.
func (c *Client) MoveClient(clid, cid int) error {
	_, err := c.Command(fmt.Sprintf("clientmove clid=%d cid=%d", clid, cid))
	return err
}
