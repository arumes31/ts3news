package clientquery

import (
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
	CLID     int
	Nickname string
	Type     int // 0 = normal voice client, 1 = ServerQuery/ClientQuery client
}

// ClientList returns all clients on the currently selected virtual server.
func (c *Client) ClientList() ([]ClientInfo, error) {
	data, err := c.Command("clientlist")
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
