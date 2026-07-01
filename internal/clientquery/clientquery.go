// Package clientquery is a small client for the TeamSpeak 3 ClientQuery plugin,
// which exposes a telnet interface (default 127.0.0.1:25639) to remote-control a
// running TeamSpeak client. The bot uses it to enumerate clients and send pokes
// or text messages.
package clientquery

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

type Client struct {
	conn   net.Conn
	reader *bufio.Reader
}

// Dial connects to the ClientQuery telnet interface and consumes the greeting.
func Dial(addr string, timeout time.Duration) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	c := &Client{conn: conn, reader: bufio.NewReader(conn)}
	// The server sends a multi-line greeting ("TS3 Client", "Welcome ...", ...).
	// Drain whatever is immediately available so it does not pollute command replies.
	_ = c.conn.SetReadDeadline(time.Now().Add(1500 * time.Millisecond))
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			break
		}
		if strings.HasPrefix(line, "error ") {
			break
		}
	}
	_ = c.conn.SetReadDeadline(time.Time{})
	return c, nil
}

func (c *Client) Close() error { return c.conn.Close() }

// ErrDuplicateEntry is TeamSpeak's error id for an insert that already exists —
// e.g. adding a client to a server group it is already a member of. Callers can
// treat it as a benign no-op rather than a failure.
const ErrDuplicateEntry = 2561

// CommandError is returned when a ClientQuery command completes with a non-zero
// error id. Callers can inspect ID to handle specific conditions (via errors.As).
type CommandError struct {
	Cmd string
	ID  int
	Msg string
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("command %q failed: id=%d msg=%s", e.Cmd, e.ID, e.Msg)
}

// Command sends a single command and returns the data lines (everything before
// the terminating "error id=... msg=..." line). It returns an error if the
// command's error id is non-zero.
func (c *Client) Command(cmd string) ([]string, error) {
	if _, err := fmt.Fprint(c.conn, cmd+"\n"); err != nil {
		return nil, err
	}
	var data []string
	_ = c.conn.SetReadDeadline(time.Now().Add(8 * time.Second))
	defer func() { _ = c.conn.SetReadDeadline(time.Time{}) }()
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return data, fmt.Errorf("reading reply to %q: %w", firstWord(cmd), err)
		}
		// TeamSpeak's query protocol separates lines with "\n\r", which leaves a
		// leading carriage return on every line after the first; trim both ends.
		line = strings.Trim(line, "\r\n")
		if strings.HasPrefix(line, "error ") {
			id, msg := parseError(line)
			if id != 0 {
				return data, &CommandError{Cmd: firstWord(cmd), ID: id, Msg: msg}
			}
			return data, nil
		}
		if line != "" {
			data = append(data, line)
		}
	}
}

// Auth authenticates with the ClientQuery API key.
func (c *Client) Auth(apiKey string) error {
	_, err := c.Command("auth apikey=" + Escape(apiKey))
	if err != nil {
		return fmt.Errorf("auth command failed: %w", err)
	}
	return nil
}

// DrainRaw reads and returns any lines that arrive within timeout (used to
// capture asynchronous notification events). It always returns when the timeout
// elapses.
func (c *Client) DrainRaw(timeout time.Duration) []string {
	var lines []string
	_ = c.conn.SetReadDeadline(time.Now().Add(timeout))
	defer func() { _ = c.conn.SetReadDeadline(time.Time{}) }()
	for {
		line, err := c.reader.ReadString('\n')
		if t := strings.Trim(line, "\r\n"); t != "" {
			lines = append(lines, t)
		}
		if err != nil {
			return lines
		}
	}
}

// ReadNotify reads lines until one starts with prefix (returning its parsed
// key=value fields) or the timeout elapses.
func (c *Client) ReadNotify(prefix string, timeout time.Duration) (map[string]string, bool) {
	deadline := time.Now().Add(timeout)
	_ = c.conn.SetReadDeadline(deadline)
	defer func() { _ = c.conn.SetReadDeadline(time.Time{}) }()
	for time.Now().Before(deadline) {
		line, err := c.reader.ReadString('\n')
		if t := strings.Trim(line, "\r\n"); strings.HasPrefix(t, prefix) {
			return firstKV([]string{t}), true
		}
		if err != nil {
			return nil, false
		}
	}
	return nil, false
}

// Raw sends a command and returns every line received until a terminating
// "error ..." line or the timeout, for diagnostics. It never desyncs the caller
// because it consumes through the error line when present.
func (c *Client) Raw(cmd string, timeout time.Duration) ([]string, error) {
	if _, err := fmt.Fprint(c.conn, cmd+"\n"); err != nil {
		return nil, err
	}
	var lines []string
	_ = c.conn.SetReadDeadline(time.Now().Add(timeout))
	defer func() { _ = c.conn.SetReadDeadline(time.Time{}) }()
	for {
		line, err := c.reader.ReadString('\n')
		if line != "" {
			lines = append(lines, strings.Trim(line, "\r\n"))
		}
		if err != nil {
			return lines, err
		}
		if strings.HasPrefix(strings.Trim(line, "\r\n"), "error ") {
			return lines, nil
		}
	}
}

func firstWord(s string) string {
	if i := strings.IndexByte(s, ' '); i >= 0 {
		return s[:i]
	}
	return s
}

func parseError(line string) (int, string) {
	var id int
	var msg string
	for _, tok := range strings.Fields(line) {
		switch {
		case strings.HasPrefix(tok, "id="):
			_, _ = fmt.Sscanf(tok[3:], "%d", &id)
		case strings.HasPrefix(tok, "msg="):
			msg = Unescape(tok[4:])
		}
	}
	return id, msg
}

func (c *Client) Reader() *bufio.Reader {
	return c.reader
}

