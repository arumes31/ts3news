package idely

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// AudioBot drives a TS3AudioBot server over its HTTP WebAPI. TS3AudioBot is a
// separate service (the sidecar) that can host MANY bot instances at once — each
// is its own voice client connection. Idely uses that to serenade several idle
// channels simultaneously: one spawned bot instance per channel.
//
// The WebAPI turns chat commands into URLs: a space becomes "/", and command
// grouping "(!" becomes "(/". Per-instance commands go to
// /api/bot/use/<id>/(/command/args); spawning a new instance is
// /api/bot/connect/to/<address>. Authentication is HTTP Basic with
// base64("<uid>:<token>").
type AudioBot struct {
	baseURL string
	authHdr string
	http    *http.Client
}

// AudioBotConfig configures an AudioBot manager.
type AudioBotConfig struct {
	BaseURL string        // TS3AudioBot WebAPI base URL
	UID     string        // Basic-auth username: the TS3 uid that owns the token
	Token   string        // Basic-auth token from `!api token` (empty => anonymous)
	Timeout time.Duration // per-request timeout (default 10s)
}

// NewAudioBot builds an AudioBot manager from config.
func NewAudioBot(cfg AudioBotConfig) *AudioBot {
	to := cfg.Timeout
	if to == 0 {
		to = 10 * time.Second
	}
	var authHdr string
	if cfg.Token != "" {
		tok := strings.Replace(cfg.Token, ":ts3ab:", ":", 1)
		authHdr = "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.UID+":"+tok))
	}
	return &AudioBot{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		authHdr: authHdr,
		http:    &http.Client{Timeout: to},
	}
}

// get issues a GET to an already-built API path and returns the body. A non-2xx
// status is an error carrying the (JSON) body.
func (a *AudioBot) get(apiPath string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, a.baseURL+apiPath, nil)
	if err != nil {
		return nil, err
	}
	if a.authHdr != "" {
		req.Header.Set("Authorization", a.authHdr)
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ts3audiobot request %q: %w", apiPath, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, fmt.Errorf("ts3audiobot %q -> HTTP %d: %s", apiPath, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// command runs a chat command (arguments URL-escaped) on bot instance botID.
func (a *AudioBot) command(botID int, segments ...string) ([]byte, error) {
	escaped := make([]string, len(segments))
	for i, s := range segments {
		escaped[i] = url.PathEscape(s)
	}
	return a.get(fmt.Sprintf("/api/bot/use/%d/(/%s)", botID, strings.Join(escaped, "/")))
}

// botStatus is the subset of TS3AudioBot's bot-info JSON we care about.
type botStatus struct {
	Id     int
	Status int // 0 offline, 1 connecting, 2 connected
}

// List returns the ids of all current bot instances on the TS3AudioBot server.
func (a *AudioBot) List() ([]int, error) {
	body, err := a.get("/api/bot/list")
	if err != nil {
		return nil, err
	}
	var bots []struct{ Id int }
	if jerr := json.Unmarshal(body, &bots); jerr != nil {
		return nil, fmt.Errorf("ts3audiobot list: %w", jerr)
	}
	ids := make([]int, 0, len(bots))
	for _, b := range bots {
		ids = append(ids, b.Id)
	}
	return ids, nil
}

// Connect spawns a new bot instance connected to serverAddr and returns its id.
func (a *AudioBot) Connect(serverAddr string) (int, error) {
	body, err := a.get("/api/bot/connect/to/" + url.PathEscape(serverAddr))
	if err != nil {
		return 0, err
	}
	var st botStatus
	if jerr := json.Unmarshal(body, &st); jerr != nil {
		return 0, fmt.Errorf("ts3audiobot connect: bad response %q: %w", strings.TrimSpace(string(body)), jerr)
	}
	return st.Id, nil
}

// WaitConnected polls until bot instance botID reports connected (Status 2) or
// the timeout elapses.
func (a *AudioBot) WaitConnected(botID int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		body, err := a.command(botID, "bot", "info")
		if err == nil {
			var st botStatus
			if json.Unmarshal(body, &st) == nil && st.Status == 2 {
				return nil
			}
		}
		time.Sleep(750 * time.Millisecond)
	}
	return fmt.Errorf("ts3audiobot bot %d did not connect within %s", botID, timeout)
}

// LoadPlugin loads a TS3AudioBot plugin (by id or name) for instance botID, e.g.
// the IdelyTalkSensor. The first load compiles the plugin; later loads for other
// instances reuse the compiled assembly.
func (a *AudioBot) LoadPlugin(botID int, ref string) error {
	_, err := a.command(botID, "plugin", "load", ref)
	return err
}

// SetName sets instance botID's displayed nickname (the "random name").
func (a *AudioBot) SetName(botID int, name string) error {
	_, err := a.command(botID, "bot", "name", name)
	return err
}

// Move moves instance botID into channel cid.
func (a *AudioBot) Move(botID, cid int) error {
	_, err := a.command(botID, "bot", "move", fmt.Sprintf("%d", cid))
	return err
}

// Play starts playing resource on instance botID, replacing anything playing.
func (a *AudioBot) Play(botID int, resource string) error {
	_, err := a.command(botID, "play", resource)
	return err
}

// Enqueue appends a resource to instance botID's queue.
func (a *AudioBot) Enqueue(botID int, resource string) error {
	_, err := a.command(botID, "add", resource)
	return err
}

// Loop enables/disables looping of instance botID's whole queue. Best-effort.
func (a *AudioBot) Loop(botID int, on bool) error {
	mode := "off"
	if on {
		mode = "all"
	}
	_, err := a.command(botID, "repeat", mode)
	return err
}

// Volume sets instance botID's playback volume (0..100). Best-effort.
func (a *AudioBot) Volume(botID, pct int) error {
	_, err := a.command(botID, "volume", fmt.Sprintf("%d", pct))
	return err
}

// Stop halts playback and clears the queue on instance botID.
func (a *AudioBot) Stop(botID int) error {
	_, err := a.command(botID, "stop")
	return err
}

// Disconnect disconnects instance botID from the server (removing the bot).
func (a *AudioBot) Disconnect(botID int) error {
	_, err := a.command(botID, "bot", "disconnect")
	return err
}

// VoiceAge asks the IdelyTalkSensor plugin how many milliseconds ago the bot
// last heard voice in its channel (a real user talking). It returns -1 if the
// plugin has never heard voice, or an error if the plugin/command is absent.
func (a *AudioBot) VoiceAge(botID int) (int, error) {
	body, err := a.command(botID, "idely", "voiceage")
	if err != nil {
		return 0, err
	}
	// The command returns a string; TS3AudioBot may wrap it as {"Value":"…"}.
	var wrapped struct{ Value string }
	s := strings.TrimSpace(string(body))
	if json.Unmarshal(body, &wrapped) == nil && wrapped.Value != "" {
		s = strings.TrimSpace(wrapped.Value)
	}
	s = strings.Trim(s, `"`)
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("ts3audiobot voiceage: bad value %q", s)
	}
	return n, nil
}
