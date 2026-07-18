package idely

import "time"

// AudioPlayer implements Player by spawning a fresh TS3AudioBot instance for each
// idle channel: it connects a new bot to the server, gives it a random name,
// moves it into the target channel, and loops the lo-fi playlist. Stopping
// disconnects that instance, so each idle channel gets its own concurrent bot.
type AudioPlayer struct {
	bot         *AudioBot
	serverAddr  string        // TS3 server address new bots connect to
	volume      int           // 0..100, applied on Start (<=0 leaves default)
	pluginRef   string        // TS3AudioBot plugin (id/name) to load per bot for talk sensing
	connectWait time.Duration // how long to wait for a spawned bot to connect
	logf        func(format string, args ...any)
}

// AudioPlayerConfig configures an AudioPlayer.
type AudioPlayerConfig struct {
	Bot         *AudioBot
	ServerAddr  string
	Volume      int
	PluginRef   string        // load this plugin on each spawned bot ("" = none)
	ConnectWait time.Duration // default 20s
	Logf        func(format string, args ...any)
}

// NewAudioPlayer builds an AudioPlayer.
func NewAudioPlayer(c AudioPlayerConfig) *AudioPlayer {
	logf := c.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	wait := c.ConnectWait
	if wait <= 0 {
		wait = 20 * time.Second
	}
	return &AudioPlayer{bot: c.Bot, serverAddr: c.ServerAddr, volume: c.Volume, pluginRef: c.PluginRef, connectWait: wait, logf: logf}
}

// Start spawns a new audio-bot instance, waits for it to connect, names it,
// moves it into cid, and loops tracks. It returns the instance id as the handle.
// On any load-bearing failure it disconnects the half-spawned bot so instances
// don't leak.
func (p *AudioPlayer) Start(cid int, name string, tracks []string) (int, error) {
	if len(tracks) == 0 {
		return 0, errNoTracks
	}
	id, err := p.bot.Connect(p.serverAddr)
	if err != nil {
		return 0, err
	}
	if err := p.bot.WaitConnected(id, p.connectWait); err != nil {
		_ = p.bot.Disconnect(id)
		return 0, err
	}
	// Load the talk-sensor plugin so this bot can report when a user talks.
	// Best-effort: playback still works without it (just no talk-stop).
	if p.pluginRef != "" {
		if err := p.bot.LoadPlugin(id, p.pluginRef); err != nil {
			p.logf("idely: loading talk-sensor plugin on bot %d failed (no talk-stop): %v", id, err)
		}
	}
	if name != "" {
		if err := p.bot.SetName(id, name); err != nil {
			p.logf("idely: set name %q failed (continuing): %v", name, err)
		}
	}
	if err := p.bot.Move(id, cid); err != nil {
		_ = p.bot.Disconnect(id)
		return 0, err
	}
	if p.volume > 0 {
		if err := p.bot.Volume(id, p.volume); err != nil {
			p.logf("idely: set volume failed (continuing): %v", err)
		}
	}
	if err := p.bot.Play(id, tracks[0]); err != nil {
		_ = p.bot.Disconnect(id)
		return 0, err
	}
	for _, t := range tracks[1:] {
		if err := p.bot.Enqueue(id, t); err != nil {
			p.logf("idely: enqueue %q failed (continuing): %v", t, err)
		}
	}
	if err := p.bot.Loop(id, true); err != nil {
		p.logf("idely: enable loop failed (continuing): %v", err)
	}
	return id, nil
}

// Stop halts playback on the instance and disconnects it.
func (p *AudioPlayer) Stop(handle int) error {
	err := p.bot.Stop(handle)
	if derr := p.bot.Disconnect(handle); derr != nil {
		p.logf("idely: disconnect bot %d failed: %v", handle, derr)
	}
	return err
}
