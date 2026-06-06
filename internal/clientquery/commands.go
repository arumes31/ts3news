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

func Escape(s string) string   { return escaper.Replace(s) }
func Unescape(s string) string { return unescaper.Replace(s) }

// Client describes one client visible on the server.
type ClientInfo struct {
	CLID            int
	Nickname        string
	UID             string // client_unique_identifier (stable TeamSpeak identity id)
	Type            int    // 0 = normal voice client, 1 = ServerQuery/ClientQuery client
	ConnectedTimeMS int64  // client_connected_time (ms since session start)
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
			ci := ClientInfo{CLID: -1}
			for _, field := range strings.Fields(rec) {
				k, v, ok := strings.Cut(field, "=")
				if !ok {
					continue
				}
				switch k {
				case "clid":
					ci.CLID, _ = strconv.Atoi(v)
				case "client_nickname":
					ci.Nickname = Unescape(v)
				case "client_unique_identifier":
					ci.UID = Unescape(v)
				case "client_type":
					ci.Type, _ = strconv.Atoi(v)
				}
			}

			// For voice clients, fetch additional session stats (connected time)
			if ci.CLID >= 0 && ci.Type == 0 {
				if info, ierr := c.Command(fmt.Sprintf("clientinfo clid=%d", ci.CLID)); ierr == nil {
					for _, iline := range info {
						for _, f := range strings.Fields(iline) {
							if val, ok := strings.CutPrefix(f, "client_connected_time="); ok {
								ci.ConnectedTimeMS, _ = strconv.ParseInt(val, 10, 64)
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

// IsConnected reports whether the client is currently connected to a server
// (whoami returns a non-zero own client id).
func (c *Client) IsConnected() bool {
	data, err := c.WhoAmI()
	if err != nil {
		return false // "currently not possible" => not connected
	}
	for _, line := range data {
		for _, f := range strings.Fields(line) {
			if v, ok := strings.CutPrefix(f, "clid="); ok {
				return v != "0" && v != ""
			}
		}
	}
	return false
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

// AddServerGroup adds a client (by database id) to a server group. Requires the
// bot's identity to hold the necessary group-management permission.
func (c *Client) AddServerGroup(sgid, cldbid int) error {
	_, err := c.Command(fmt.Sprintf("servergroupaddclient sgid=%d cldbid=%d", sgid, cldbid))
	return err
}
