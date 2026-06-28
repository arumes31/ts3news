package bot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"ts3news/internal/clientquery"
)

// Supervisor owns the long-running lifecycle: it (re)starts the TeamSpeak client
// each cycle, drives one poke cycle via ClientQuery, then disconnects and sleeps
// a random interval. It includes a watchdog (restarts the client if ClientQuery
// never becomes responsive / the client dies) and graceful SIGTERM handling
// (finish the in-progress poke cycle, then exit).
type Supervisor struct {
	bot *Bot
}

func NewSupervisor(b *Bot) *Supervisor { return &Supervisor{bot: b} }

var (
	errClientExited = errors.New("ts3 client process exited")
	errShutdown     = errors.New("shutdown requested")
	errConnectTO    = errors.New("ts3 client did not become responsive in time")
)

// Run blocks until a termination signal is received. On SIGINT/SIGTERM it
// finishes the current cycle (if poking is underway) and then returns.
func (s *Supervisor) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go s.startCommandListener(ctx)

	for {
		err := s.runCycleWithClient(ctx)
		if err != nil {
			if errors.Is(err, errShutdown) {
				log.Println("Shutdown requested during connect; exiting.")
				return nil
			}
			log.Printf("Cycle error (watchdog will restart client shortly): %v", err)
		}

		// Housekeeping: purge users that have not connected for the configured period.
		if n, cerr := s.bot.CleanupDeadUsers(); cerr != nil {
			log.Printf("Dead-user cleanup failed: %v", cerr)
		} else if n > 0 {
			log.Printf("Dead-user cleanup removed %d inactive user(s).", n)
		}

		// Graceful shutdown: the cycle has finished, so exit now.
		select {
		case <-ctx.Done():
			log.Println("Termination signal received; exiting after completing the cycle.")
			return nil
		default:
		}

		// On error, retry soon; on success, sleep a random interval.
		var sleep time.Duration
		if err != nil {
			sleep = 30 * time.Second
			log.Printf("Retrying in %s.", sleep)
		} else {
			sleep = s.randomInterval()
			log.Printf("Sleeping %s until next cycle.", sleep.Round(time.Minute))
		}
		if interrupted := sleepCtx(ctx, sleep); interrupted {
			log.Println("Termination signal received during sleep; exiting.")
			return nil
		}
	}
}

// runCycleWithClient starts the TS3 client, waits for it to connect (watchdog),
// runs one poke cycle, then stops the client. Returns an error if the client
// never became responsive (so the loop restarts it).
func (s *Supervisor) runCycleWithClient(shutdownCtx context.Context) error {
	cfg := s.bot.Cfg
	url := fmt.Sprintf("ts3server://%s?port=%d&nickname=%s", cfg.TS3Host, cfg.TS3Port, cfg.TS3Nickname)

	// cycleCtx is cancelled when the client process exits (watchdog trigger).
	cycleCtx, cancelCycle := context.WithCancel(context.Background())
	defer cancelCycle()

	log.Printf("Launching TS3 client: %s %s", cfg.TS3ClientPath, url)
	// #nosec G204 -- TS3ClientPath is loaded securely from environment configuration
	cmd := exec.Command(cfg.TS3ClientPath, "-nosingleinstance", url)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting ts3 client: %w", err)
	}

	// Single waiter: closes exited and trips the watchdog when the client dies.
	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(exited)
		cancelCycle()
	}()
	defer s.stopClient(cmd, exited)

	// Clear any first-run/promo popups that might appear.
	go clearPopups(cycleCtx)

	c, err := s.connect(shutdownCtx, cycleCtx)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()

	log.Println("TS3 client connected. Running notification cycle...")
	if err := c.Use(1); err != nil {
		log.Printf("Warning: 'use 1' failed: %v", err)
	}

	// Server-group "list" commands deliver their replies via notification events;
	// subscribe before any server-group operations this cycle.
	if s.bot.Cfg.XPServerGroups {
		if err := c.RegisterServerGroupEvents(); err != nil {
			log.Printf("Warning: registering server-group events failed: %v", err)
		}
	}

	if os.Getenv("CQ_PROBE") == "1" {
		probeClientQuery(c)
	}

	// The poke cycle itself is intentionally NOT cancelled by shutdownCtx so an
	// in-progress cycle finishes cleanly (graceful shutdown).
	cycleErr := s.bot.RunCycle(c)

	if err := c.Disconnect(); err != nil {
		log.Printf("Warning: clean disconnect failed: %v", err)
	} else {
		log.Println("Disconnected from server.")
	}
	return cycleErr
}

// connect dials ClientQuery, authenticates, and waits (up to ConnectTimeoutSec)
// for the client to be connected to the server. It aborts early on shutdown or if
// the client process exits.
func (s *Supervisor) connect(shutdownCtx, cycleCtx context.Context) (*clientquery.Client, error) {
	cfg := s.bot.Cfg
	addr := cfg.ClientQueryAddr
	if addr == "" {
		addr = "127.0.0.1:25639"
	}
	deadline := time.Now().Add(time.Duration(cfg.ConnectTimeoutSec) * time.Second)

	log.Printf("Connecting to ClientQuery at %s (timeout %ds)...", addr, cfg.ConnectTimeoutSec)
	var c *clientquery.Client
	for time.Now().Before(deadline) {
		if err := ctxErr(shutdownCtx, cycleCtx); err != nil {
			return nil, err
		}
		cc, err := clientquery.Dial(addr, 2*time.Second)
		if err == nil {
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

	if apiKey := s.bot.getAPIKey(); apiKey != "" {
		log.Println("Authenticating with ClientQuery...")
		if err := c.Auth(apiKey); err != nil {
			log.Printf("Warning: ClientQuery authentication failed: %v", err)
		}
	}

	log.Println("Waiting for TS3 client to be connected to the server...")
	start := time.Now()
	lastLog := time.Time{}
	for time.Now().Before(deadline) {
		if err := ctxErr(shutdownCtx, cycleCtx); err != nil {
			_ = c.Close()
			return nil, err
		}
		// Select the server connection handler (schandler 1) before querying
		// identity. Until the client has actually created the handler, both "use"
		// and "whoami" fail with id=1799 ("invalid server connection handler ID").
		// Polling whoami without this selected the wrong/no handler, so a fully
		// connected client looked "not connected" until the watchdog timed out.
		uerr := c.Use(1)
		var info clientquery.WhoAmIInfo
		var werr error
		if uerr == nil {
			info, werr = c.WhoAmIInfo()
			if werr == nil && info.CLID > 0 {
				log.Printf("TS3 client is connected (clid=%d cid=%d, took %s).", info.CLID, info.CID, time.Since(start).Round(time.Second))
				return c, nil
			}
		}
		// Surface why we're still waiting: log the first poll and then every 15s.
		if time.Since(lastLog) >= 15*time.Second || lastLog.IsZero() {
			switch {
			case uerr != nil:
				log.Printf("  ...still waiting (%s): connection handler not ready: %v", time.Since(start).Round(time.Second), uerr)
			case werr != nil:
				log.Printf("  ...still waiting (%s): whoami failed: %v", time.Since(start).Round(time.Second), werr)
			default:
				log.Printf("  ...still waiting (%s): client up but not on a server yet (clid=%d status=%q)", time.Since(start).Round(time.Second), info.CLID, info.Status)
			}
			lastLog = time.Now()
		}
		if sleepCtx(shutdownCtx, time.Second) {
			_ = c.Close()
			return nil, errShutdown
		}
	}
	_ = c.Close()
	return nil, errConnectTO
}

// stopClient stops the TS3 client gracefully, escalating to SIGKILL if needed.
// It waits on the supplied exited channel (closed by the single cmd.Wait waiter)
// so the process is reaped exactly once.
func (s *Supervisor) stopClient(cmd *exec.Cmd, exited <-chan struct{}) {
	if cmd.Process == nil {
		return
	}
	// Already gone?
	select {
	case <-exited:
		return
	default:
	}

	log.Printf("Stopping TS3 client (PID %d)...", cmd.Process.Pid)
	_ = cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-exited:
		return
	case <-time.After(5 * time.Second):
		log.Println("TS3 client did not exit; sending SIGKILL.")
		_ = cmd.Process.Kill()
		<-exited
	}
}

func (s *Supervisor) randomInterval() time.Duration {
	cfg := s.bot.Cfg
	min, max := cfg.MinIntervalHours, cfg.MaxIntervalHours
	if min < 0 {
		min = 0
	}
	if max < min {
		max = min
	}
	span := max - min + 1
// #nosec G404
	hours := min + rand.IntN(span) // #nosec G404
	return time.Duration(hours) * time.Hour
}

// probeClientQuery issues a battery of raw ClientQuery commands and logs the full
// replies, to diagnose how the server-group admin commands respond. Enabled with
// CQ_PROBE=1.
func probeClientQuery(c *clientquery.Client) {
	cmds := []string{
		"clientnotifyregister schandlerid=1 event=any",
		"ftinitupload clientftfid=1 name=\\/icon_77777777 cid=0 cpw= size=120 overwrite=1 resume=0",
	}
	for _, cmd := range cmds {
		lines, err := c.Raw(cmd, 6*time.Second)
		log.Printf("PROBE %q -> err=%v lines=%d %q", cmd, err, len(lines), lines)
	}
	// Drain any asynchronous notifications (e.g. the file-transfer key/port).
	log.Printf("PROBE drain -> %q", c.DrainRaw(5*time.Second))
}

// clearPopups periodically sends Escape via xdotool to dismiss first-run dialogs.
func clearPopups(ctx context.Context) {
	for i := 0; i < 6; i++ {
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
		_ = exec.Command("xdotool", "key", "--clearmodifiers", "Escape").Run()
	}
}

// ctxErr returns a mapped error if either context is done, else nil.
func ctxErr(shutdownCtx, cycleCtx context.Context) error {
	select {
	case <-shutdownCtx.Done():
		return errShutdown
	case <-cycleCtx.Done():
		return errClientExited
	default:
		return nil
	}
}

// sleepCtx sleeps for d or until ctx is done; returns true if interrupted by ctx.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return true
	case <-t.C:
		return false
	}
}

func parseNotification(line string) (string, map[string]string) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", nil
	}
	name := fields[0]
	m := map[string]string{}
	for _, f := range fields[1:] {
		k, v, ok := strings.Cut(f, "=")
		if ok {
			m[k] = clientquery.Unescape(v)
		}
	}
	return name, m
}

func (s *Supervisor) startCommandListener(ctx context.Context) {
	addr := s.bot.Cfg.ClientQueryAddr
	if addr == "" {
		addr = "127.0.0.1:25639"
	}
	apiKey := s.bot.getAPIKey()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		c, err := clientquery.Dial(addr, 5*time.Second)
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}

		if apiKey != "" {
			_ = c.Auth(apiKey)
		}
		_ = c.Use(1)

		// Register for text messages and pokes
		_, _ = c.Command("clientnotifyregister schandlerid=1 event=notifytextmessage")
		_, _ = c.Command("clientnotifyregister schandlerid=1 event=notifypoke")

		reader := c.Reader()
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			line = strings.Trim(line, "\r\n")
			s.handleNotificationLine(c, line)
		}
		_ = c.Close()
		time.Sleep(5 * time.Second)
	}
}

func (s *Supervisor) handleNotificationLine(c *clientquery.Client, line string) {
	evName, params := parseNotification(line)
	if evName != "notifytextmessage" && evName != "notifypoke" {
		return
	}

	msg := params["msg"]
	if !strings.HasPrefix(strings.TrimSpace(msg), "!abyss") {
		return
	}

	clidStr := params["clid"]
	if clidStr == "" {
		clidStr = params["invokerid"]
	}
	clid, _ := strconv.Atoi(clidStr)
	if clid == 0 {
		return
	}

	uid := params["invokeruid"]
	nick := params["invokername"]
	if uid == "" {
		return
	}

	// Fetch Abyss Summary from database
	var bestDepth, deaths, lifetimeFloors int
	var active bool
	var depth int
	var escrow int64

	// Query best depth, deaths, floors
	_ = s.bot.DB.QueryRow(
		"SELECT abyss_best_depth, abyss_deaths, abyss_lifetime_floors FROM users WHERE client_uid=$1", uid,
	).Scan(&bestDepth, &deaths, &lifetimeFloors)

	// Check if run is active
	err := s.bot.DB.QueryRow(
		"SELECT depth, escrow FROM abyss_active WHERE client_uid=$1", uid,
	).Scan(&depth, &escrow)
	if err == nil {
		active = true
	}

	summary := fmt.Sprintf(
		"⚔️ [b]Abyss Summary for %s[/b] ⚔️\n• Best Depth: Floor %d\n• Lifetime Floors: %d\n• Total Deaths: %d\n• Active Run: %s",
		nick, bestDepth, lifetimeFloors, deaths, "No active run",
	)
	if active {
		summary = fmt.Sprintf(
			"⚔️ [b]Abyss Summary for %s[/b] ⚔️\n• Best Depth: Floor %d\n• Lifetime Floors: %d\n• Total Deaths: %d\n• Active Run: Floor %d (Escrow: %d gold)",
			nick, bestDepth, lifetimeFloors, deaths, depth, escrow,
		)
	}

	if evName == "notifypoke" {
		_ = c.Poke(clid, summary)
	} else {
		_ = c.SendPrivateMessage(clid, summary)
	}
}

