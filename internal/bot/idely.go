package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"ts3news/internal/clientquery"
	"ts3news/internal/config"
	"ts3news/internal/idely"
)

// audioExts are the track file types Idely will hand to TS3AudioBot when scanning
// a track directory.
var audioExts = []string{"*.mp3", "*.ogg", "*.opus", "*.flac", "*.wav", "*.m4a"}

// idelyTracks resolves the playlist: an explicit IDELY_TRACKS list wins;
// otherwise the tracks directory is scanned so a whole downloaded library is
// picked up automatically.
func idelyTracks(cfg *config.Config) []string {
	if len(cfg.IdelyTracks) > 0 {
		return cfg.IdelyTracks
	}
	if cfg.IdelyTracksDir == "" {
		return nil
	}
	tracks := scanTracks(cfg.IdelyTracksDir)
	log.Printf("Idely: scanned %d track(s) from %s", len(tracks), cfg.IdelyTracksDir)
	return tracks
}

// idelyNameSet turns the audio-bot name pool into a lookup set so the detector
// can recognise (and ignore) the bots Idely spawns.
func idelyNameSet(names []string) map[string]bool {
	if len(names) == 0 {
		return nil
	}
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// idelyAllowCIDs turns the configured channel whitelist into a set (nil when
// unrestricted, so every channel is eligible).
func idelyAllowCIDs(cfg *config.Config) map[int]bool {
	if len(cfg.IdelyOnlyCIDs) == 0 {
		return nil
	}
	m := make(map[int]bool, len(cfg.IdelyOnlyCIDs))
	for _, cid := range cfg.IdelyOnlyCIDs {
		m[cid] = true
	}
	return m
}

// scanTracks returns the base filenames of playable audio files in dir, sorted
// for determinism. Base names are used because TS3AudioBot resolves them against
// its own media path (which points at the same shared /audio volume).
func scanTracks(dir string) []string {
	var out []string
	for _, ext := range audioExts {
		matches, _ := filepath.Glob(filepath.Join(dir, ext))
		for _, m := range matches {
			out = append(out, filepath.Base(m))
		}
	}
	sort.Strings(out)
	return out
}

// runIdely owns the lifecycle of the Idely idle-music subsystem. It keeps a
// dedicated, always-on TeamSpeak client running (its own identity, profile and
// ClientQuery port, separate from the main poke-bot client which is torn down
// between cycles) and, while that client is connected, runs the detection
// service that dispatches the TS3AudioBot sidecar to serenade dormant channels.
//
// It restarts the client + service if either dies, and exits cleanly on ctx
// cancellation. It is started as a goroutine from Supervisor.Run.
func (s *Supervisor) runIdely(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		if err := s.runIdelyOnce(ctx); err != nil && ctx.Err() == nil {
			log.Printf("Idely: %v", err)
		}
		if sleepCtx(ctx, 20*time.Second) {
			return
		}
	}
}

// runIdelyOnce launches the Idely client, connects, and runs the service until
// the client dies or ctx is cancelled. Setup failures are returned; a normal
// end (client exit / shutdown) returns nil so the caller retries or stops.
func (s *Supervisor) runIdelyOnce(ctx context.Context) error {
	cfg := s.bot.Cfg

	// cycleCtx is cancelled when the Idely client process exits (watchdog).
	cycleCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	url := fmt.Sprintf("ts3server://%s?port=%d&nickname=%s", cfg.TS3Host, cfg.TS3Port, cfg.IdelyNickname)
	log.Printf("Idely: launching dedicated client (HOME=%s): %s", cfg.IdelyHome, cfg.IdelyClientPath)
	// #nosec G204 -- IdelyClientPath is loaded from trusted environment configuration.
	cmd := exec.Command(cfg.IdelyClientPath, "-nosingleinstance", url)
	// A separate HOME gives this client its own ~/.ts3client profile (identity,
	// ClientQuery port and API key), so it coexists with the main bot's client.
	cmd.Env = append(os.Environ(), "HOME="+cfg.IdelyHome)
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting Idely client: %w", err)
	}
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited); cancel() }()
	defer s.stopClient(cmd, exited)

	// Fall back to the standard ClientQuery API key (from clientquery.ini) when no
	// Idely-specific key is set — the own-container profile is the baked default.
	apiKey := cfg.IdelyAPIKey
	if apiKey == "" {
		apiKey = s.bot.getAPIKey()
	}

	poll, err := idelyConnect(ctx, cycleCtx, cfg.IdelyClientQueryAddr, apiKey, cfg.ConnectTimeoutSec)
	if err != nil {
		return fmt.Errorf("connecting Idely ClientQuery: %w", err)
	}
	defer func() { _ = poll.Close() }()

	audioBot := idely.NewAudioBot(idely.AudioBotConfig{
		BaseURL: cfg.IdelyAPIURL,
		UID:     cfg.IdelyAPIUser,
		Token:   cfg.IdelyAPIToken,
	})
	// Disconnect any audio bots left over from a previous Idely instance (a hard
	// kill doesn't let Idely tear its bots down, so they'd keep playing forever).
	// TS3AudioBot is dedicated to Idely, so every instance is ours to reap.
	if ids, lerr := audioBot.List(); lerr == nil && len(ids) > 0 {
		for _, id := range ids {
			_ = audioBot.Disconnect(id)
		}
		log.Printf("Idely: cleaned up %d orphaned audio bot(s) from a previous run.", len(ids))
	}
	engine := idely.NewEngine(idely.EngineConfig{
		Detect: idely.Config{
			IdleThreshold: time.Duration(cfg.IdelyIdleMinutes) * time.Minute,
			ActiveGrace:   time.Duration(cfg.IdelyActiveGraceSeconds) * time.Second,
			ExcludeUIDs:   s.idelyExclusions(poll),
			ExcludeNicks:  idelyNameSet(cfg.IdelyNames),
			AllowCIDs:     idelyAllowCIDs(cfg),
		},
		Player: idely.NewAudioPlayer(idely.AudioPlayerConfig{
			Bot:        audioBot,
			ServerAddr: fmt.Sprintf("%s:%d", cfg.TS3Host, cfg.TS3Port),
			Volume:     cfg.IdelyVolume,
			PluginRef:  cfg.IdelyPluginRef,
			Logf:       log.Printf,
		}),
		Tracks:       idelyTracks(cfg),
		Names:        cfg.IdelyNames,
		MaxBots:      cfg.IdelyMaxBots,
		TalkCooldown: time.Duration(cfg.IdelyTalkCooldownSec) * time.Second,
		Logf:         log.Printf,
	})

	// ClientQuery cannot read remote clients' idle time, so Idely measures it
	// itself from voice activity: the tracker ages every client and is reset when
	// they talk. Both the poll loop and the talk listener feed it.
	tracker := idely.NewActivityTracker(nil)

	// Real-time talk events arrive on a dedicated connection so they don't race
	// the polling connection's command replies.
	talk := make(chan int, 16)
	go s.idelyTalkListener(cycleCtx, tracker, talk)

	svc := &idely.Service{
		Engine: engine,
		Snapshot: func() ([]idely.Client, error) {
			raw, err := idely.SnapshotClientQuery(poll)
			if err != nil {
				return nil, err
			}
			return tracker.Observe(raw), nil
		},
		TalkStarts:   talk,
		PollInterval: time.Duration(cfg.IdelyPollSeconds) * time.Second,
		Logf:         log.Printf,
	}
	// Stop-on-talk: poll each serenading bot's IdelyTalkSensor plugin; if it
	// heard voice within the window, a real user is talking there.
	if cfg.IdelyTalkStopMS > 0 {
		svc.TalkCheck = func(handle int) bool {
			ms, err := audioBot.VoiceAge(handle)
			return err == nil && ms >= 0 && ms <= cfg.IdelyTalkStopMS
		}
	}
	log.Printf("Idely: connected; watching for idle channels (idle>=%dm, poll=%ds).", cfg.IdelyIdleMinutes, cfg.IdelyPollSeconds)
	svc.Run(cycleCtx)
	return nil
}

// idelyExclusions builds the set of identities that must never count as real
// users: the configured audio-bot UID, any extra exclusions, and the Idely
// client's own UID (resolved live via whoami + clientvariable).
func (s *Supervisor) idelyExclusions(c *clientquery.Client) map[string]bool {
	cfg := s.bot.Cfg
	excl := map[string]bool{}
	for _, u := range cfg.IdelyExcludeUIDs {
		if u != "" {
			excl[u] = true
		}
	}
	if cfg.IdelyBotUID != "" {
		excl[cfg.IdelyBotUID] = true
	}
	if info, err := c.WhoAmIInfo(); err == nil && info.CLID > 0 {
		if uid, uerr := c.ClientUID(info.CLID); uerr == nil && uid != "" {
			excl[uid] = true
		} else if uerr != nil {
			log.Printf("Idely: could not resolve own UID (self may count as a user): %v", uerr)
		}
	}
	return excl
}

// idelyConnect dials the Idely client's ClientQuery, authenticates, selects
// server handler 1, and waits until the client is connected to the server.
func idelyConnect(shutdownCtx, cycleCtx context.Context, addr, apiKey string, timeoutSec int) (*clientquery.Client, error) {
	if addr == "" {
		addr = "127.0.0.1:25640"
	}
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	log.Printf("Idely: connecting to ClientQuery at %s (timeout %ds)...", addr, timeoutSec)

	var c *clientquery.Client
	for time.Now().Before(deadline) {
		if err := ctxErr(shutdownCtx, cycleCtx); err != nil {
			return nil, err
		}
		if cc, err := clientquery.Dial(addr, 2*time.Second); err == nil {
			c = cc
			break
		}
		if sleepCtx(shutdownCtx, time.Second) {
			return nil, errShutdown
		}
	}
	if c == nil {
		return nil, errConnectTO
	}

	if apiKey != "" {
		if err := c.Auth(apiKey); err != nil {
			log.Printf("Idely: ClientQuery authentication failed: %v", err)
		}
	}

	for time.Now().Before(deadline) {
		if err := ctxErr(shutdownCtx, cycleCtx); err != nil {
			_ = c.Close()
			return nil, err
		}
		if c.Use(1) == nil {
			if info, err := c.WhoAmIInfo(); err == nil && info.CLID > 0 {
				return c, nil
			}
		}
		if sleepCtx(shutdownCtx, time.Second) {
			_ = c.Close()
			return nil, errShutdown
		}
	}
	_ = c.Close()
	return nil, errConnectTO
}

// idelyTalkListener maintains a dedicated ClientQuery connection subscribed to
// notifytalkstatuschange and forwards the clids of clients that start talking
// (status=1) onto talk. It reconnects on failure until ctx is cancelled. The
// talk-start feed is a best-effort fast path; the service's idle-time polling is
// the guaranteed stop mechanism.
func (s *Supervisor) idelyTalkListener(ctx context.Context, tracker *idely.ActivityTracker, talk chan<- int) {
	cfg := s.bot.Cfg
	addr := cfg.IdelyClientQueryAddr
	if addr == "" {
		addr = "127.0.0.1:25639"
	}
	apiKey := cfg.IdelyAPIKey
	if apiKey == "" {
		apiKey = s.bot.getAPIKey()
	}
	for {
		if ctx.Err() != nil {
			return
		}
		c, err := clientquery.Dial(addr, 5*time.Second)
		if err != nil {
			if sleepCtx(ctx, 10*time.Second) {
				return
			}
			continue
		}
		if apiKey != "" {
			_ = c.Auth(apiKey)
		}
		_ = c.Use(1)
		// Subscribe to all channels so talk events for the target channel are
		// delivered even when the Idely client is parked elsewhere (best-effort).
		_, _ = c.Command("channelsubscribeall")
		if err := c.RegisterTalkStatusEvents(); err != nil {
			log.Printf("Idely: talk-status registration failed: %v", err)
			_ = c.Close()
			if sleepCtx(ctx, 10*time.Second) {
				return
			}
			continue
		}

		// Close the connection when ctx ends so the blocking ReadString unblocks.
		go func() { <-ctx.Done(); _ = c.Close() }()

		reader := c.Reader()
		for {
			line, rerr := reader.ReadString('\n')
			if t := strings.Trim(line, "\r\n"); t != "" {
				if clid, ok := parseTalkStart(t); ok {
					tracker.Touch(clid) // reset this client's idle timer
					select {
					case talk <- clid:
					default: // channel full; polling will still catch activity
					}
				}
			}
			if rerr != nil {
				break
			}
		}
		_ = c.Close()
		if sleepCtx(ctx, 5*time.Second) {
			return
		}
	}
}

// parseTalkStart returns the clid from a notifytalkstatuschange line, and ok=true
// only when it signals talking started (status=1).
func parseTalkStart(line string) (int, bool) {
	name, params := parseNotification(line)
	if name != "notifytalkstatuschange" || params["status"] != "1" {
		return 0, false
	}
	clid, err := strconv.Atoi(params["clid"])
	if err != nil || clid == 0 {
		return 0, false
	}
	return clid, true
}
