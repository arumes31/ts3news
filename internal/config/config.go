// Package config loads bot configuration from environment variables (and an
// optional .env file) into a typed Config struct.
package config

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

// Config holds every environment-configurable setting the bot reads at
// startup: TS3 connection details, feature flags, and integration secrets.
type Config struct {
	TS3Host     string
	TS3Port     int
	TS3User     string
	TS3Pass     string
	TS3Identity string
	TS3Nickname string
	TS3ServerID int

	// ClientQuery / poke settings
	ClientQueryAddr string // host:port of the ClientQuery telnet interface
	ClientQueryINI  string // path to clientquery.ini (to read the API key)
	APIKey          string // optional explicit API key (overrides the .ini)
	TargetNick      string // if set, only poke clients with this nickname (testing)
	PokeDelayMS     int    // delay between consecutive pokes, to avoid anti-flood

	// Database
	DatabaseURL     string // PostgreSQL connection URL
	ResendAfterDays int    // re-allow sending a game to a user this many days after the last send (0 = never expire)
	DeadUserDays    int    // purge users not seen for this many days (0 = never)

	// Game sources
	EnableGameNews   bool
	EnableGamerPower bool
	EnableEpic       bool
	EnableReddit     bool
	ITADKey          string   // IsThereAnyDeal API key; empty disables ITAD
	DRMFilter        []string // platforms to keep: "steam","epic","gog"

	// Message flavour
	EnableYouTubeTrailer bool
	EnableGreetings      bool
	EnableHolidayThemes  bool
	DynamicNickname      bool // change the bot's TS3 nickname based on the announced game
	EnableChannelRename  bool // rename occupied channels from the name pool each cycle

	// Leveling
	EnableLeveling    bool
	LevelGroups       string // "level:groupID,level:groupID" milestones -> existing server group
	CheaperMoreXP     bool   // true: cheaper games grant more XP; false (default): pricier games do
	XPServerGroups    bool   // auto-create one server group per level tier, with a generated icon
	EnableXPModifiers bool   // streaks, crits, loot boxes, login bonus, parties, server mult, decay, artifacts

	// i_group_show_name_in_tree for the auto-created XP groups: 0=hidden, 1=before
	// nickname, 2=after nickname. A negative value leaves the permission untouched.
	XPGroupShowNameInTree int
	// i_group_show_name_in_tree for the auto-created title groups (same encoding).
	TitleGroupShowNameInTree int

	// Icon housekeeping
	CleanupIcons       bool // each cycle, delete filebase icons no longer referenced by any group/channel/server
	CleanupIconsDryRun bool // log what CleanupIcons would delete without deleting (live-test aid)

	// RPG settings
	EnableRPG  bool // controls the entire RPG combat loop and mechanics
	RPGBaseHP  int  // Base HP to tune win-rates (default 100)
	RPGBaseSTR int  // Base STR to tune win-rates (default 10)
	RPGBaseDEF int  // Base DEF to tune win-rates (default 5)

	// Supervisor / client lifecycle
	TS3ClientPath     string // path to ts3client_linux_amd64
	MinIntervalHours  int
	MaxIntervalHours  int
	ConnectTimeoutSec int // how long to wait for the client to connect each cycle (watchdog)

	// i18n
	Lang string // BCP 47 locale ID, e.g. "en_US", "de_DE"

	// WebUI
	WebEnable     bool   // run the player web portal (armoury, battler, arcade, shop)
	WebListenAddr string // host:port the web server listens on (e.g. ":18081")
	WebBaseURL    string // public base URL used to build per-user login links

	// The Abyss (endless push-your-luck PvE dungeon web game)
	EnableAbyss bool // serve the Abyss page/APIs and PM its deep-link each cycle

	// Idely — idle-music subsystem. A second, always-on TeamSpeak client
	// (separate identity/profile/ClientQuery port) watches every occupied channel;
	// when all users in a channel have idled past IdelyIdleMinutes it dispatches a
	// TS3AudioBot sidecar to join and play lo-fi, stopping when anyone talks/moves.
	EnableIdely bool // master switch for the whole subsystem
	// IdelyOnly makes this process run ONLY the Idely subsystem (no poke cycle).
	// The ClientQuery plugin port (25639) is hard-coded and localhost-only, so the
	// Idely client cannot share a host with the main bot's client — it runs in its
	// own container with IDELY_ONLY=1.
	IdelyOnly bool

	// The dedicated Idely TeamSpeak client (detector/eyes).
	IdelyClientPath      string // ts3client binary (defaults to TS3ClientPath)
	IdelyHome            string // HOME for its isolated profile (own settings.db/identity)
	IdelyIdentity        string // its TS3 identity blob (empty => profile default)
	IdelyNickname        string // its own display name on the server
	IdelyClientQueryAddr string // host:port of its ClientQuery interface (must differ from the main bot's)
	IdelyAPIKey          string // its ClientQuery API key
	IdelyLobbyCID        int    // channel the Idely client parks in

	// Detection tuning.
	IdelyPollSeconds        int      // how often to scan idle state
	IdelyIdleMinutes        int      // per-user idle threshold before a channel is "dormant"
	IdelyActiveGraceSeconds int      // idle at/below this => user is "active again" (stop)
	IdelyExcludeUIDs        []string // extra identities never counted as real users
	IdelyOnlyCIDs           []int    // if non-empty, only serenade these channel ids

	// TS3AudioBot sidecar (voice).
	IdelyAPIURL          string   // TS3AudioBot WebAPI base URL
	IdelyAPIUser         string   // Basic-auth username (the uid that ran !api token)
	IdelyAPIToken        string   // Basic-auth token from !api token
	IdelyBotID           int      // TS3AudioBot bot instance id
	IdelyBotUID          string   // the audio bot's TS3 identity uid (excluded from idle checks)
	IdelyVolume          int      // playback volume 0..100
	IdelyMaxBots         int      // cap on concurrent audio bots (one per idle channel); 0 = unlimited
	IdelyTalkStopMS      int      // stop a channel when its audio bot heard voice within this many ms (needs the IdelyTalkSensor plugin)
	IdelyTalkCooldownSec int      // after stopping for talk, don't re-serenade the channel for this long (avoids flapping)
	IdelyPluginRef       string   // TS3AudioBot plugin id/name to load on each spawned bot (the talk sensor); "" disables
	IdelyNames           []string // pool of random names for the audio bot
	IdelyTracks          []string // explicit lo-fi resources; empty => scan IdelyTracksDir
	IdelyTracksDir       string   // directory of track files to scan (as both Idely and the audio bot see it)
}

// LoadConfig reads bot configuration from the environment (and config.env, if
// present), applying defaults for any unset values.
func LoadConfig() *Config {
	// Load variables from config.env (if present) so the bot works both inside
	// Docker (where compose injects the file as real env) and when run directly.
	// Existing environment variables always take precedence over the file.
	loadDotEnv("config.env")

	port, _ := strconv.Atoi(os.Getenv("TS3_PORT"))
	sid, _ := strconv.Atoi(os.Getenv("TS3_SERVER_ID"))

	return &Config{
		TS3Host:     os.Getenv("TS3_HOST"),
		TS3Port:     port,
		TS3User:     os.Getenv("TS3_USER"),
		TS3Pass:     os.Getenv("TS3_PASS"),
		TS3Identity: os.Getenv("TS3_IDENTITY"),
		TS3Nickname: envDefault("TS3_NICKNAME", "MrFree"),
		TS3ServerID: sid,

		ClientQueryAddr: envDefault("CLIENTQUERY_ADDR", "127.0.0.1:25639"),
		ClientQueryINI:  envDefault("CLIENTQUERY_INI", "/root/.ts3client/clientquery.ini"),
		APIKey:          os.Getenv("TS3_APIKEY"),
		TargetNick:      os.Getenv("TS3_TARGET_NICK"),
		PokeDelayMS:     envInt("POKE_DELAY_MS", 1200),

		DatabaseURL:     os.Getenv("DATABASE_URL"),
		ResendAfterDays: envInt("RESEND_AFTER_DAYS", 60),
		DeadUserDays:    envInt("DEAD_USER_DAYS", 180),

		EnableGameNews:   envBool("ENABLE_GAME_NEWS", true),
		EnableGamerPower: envBool("ENABLE_GAMERPOWER", true),
		EnableEpic:       envBool("ENABLE_EPIC", true),
		EnableReddit:     envBool("ENABLE_REDDIT", true),
		ITADKey:          os.Getenv("ITAD_API_KEY"),
		DRMFilter:        envList("DRM_FILTER", []string{"steam", "epic"}),

		EnableYouTubeTrailer: envBool("ENABLE_YOUTUBE_TRAILER", true),
		EnableGreetings:      envBool("ENABLE_GREETINGS", true),
		EnableHolidayThemes:  envBool("ENABLE_HOLIDAY_THEMES", true),
		DynamicNickname:      envBool("DYNAMIC_NICKNAME", true),
		EnableChannelRename:  envBool("ENABLE_CHANNEL_RENAME", true),
		EnableLeveling:       envBool("ENABLE_LEVELING", true),
		LevelGroups:          os.Getenv("LEVEL_GROUPS"),
		CheaperMoreXP:        envBool("CHEAPER_MORE_XP", false),
		XPServerGroups:       envBool("XP_SERVER_GROUPS", false),
		EnableXPModifiers:    envBool("ENABLE_XP_MODIFIERS", true),

		XPGroupShowNameInTree:    envInt("XP_GROUP_SHOW_NAME_IN_TREE", 2),
		TitleGroupShowNameInTree: envInt("TITLE_GROUP_SHOW_NAME_IN_TREE", 1),

		CleanupIcons:       envBool("CLEANUP_ICONS", true),
		CleanupIconsDryRun: envBool("CLEANUP_ICONS_DRYRUN", false),

		EnableRPG:  envBool("ENABLE_RPG", true),
		RPGBaseHP:  envInt("RPG_BASE_HP", 100),
		RPGBaseSTR: envInt("RPG_BASE_STR", 10),
		RPGBaseDEF: envInt("RPG_BASE_DEF", 5),

		TS3ClientPath:     envDefault("TS3_CLIENT_PATH", "/opt/ts3/ts3client_linux_amd64"),
		MinIntervalHours:  envInt("MIN_INTERVAL_HOURS", 1),
		MaxIntervalHours:  envInt("MAX_INTERVAL_HOURS", 12),
		ConnectTimeoutSec: envInt("CONNECT_TIMEOUT_SEC", 120),

		Lang: envDefault("LANG", "en_US"),

		WebEnable:     envBool("WEB_ENABLE", true),
		WebListenAddr: envDefault("WEB_LISTEN_ADDR", ":18081"),
		WebBaseURL:    envDefault("WEB_BASE_URL", "http://localhost:18081"),

		EnableAbyss: envBool("ENABLE_ABYSS", true),

		EnableIdely: envBool("ENABLE_IDELY", false),
		IdelyOnly:   envBool("IDELY_ONLY", false),

		IdelyClientPath: envDefault("IDELY_CLIENT_PATH", envDefault("TS3_CLIENT_PATH", "/opt/ts3/ts3client_linux_amd64")),
		IdelyHome:       envDefault("IDELY_HOME", "/root"),
		IdelyIdentity:   os.Getenv("IDELY_IDENTITY"),
		IdelyNickname:   envDefault("IDELY_NICKNAME", "Idely"),
		// Own-container topology: the Idely client uses the standard hard-coded
		// ClientQuery plugin port in its own network namespace.
		IdelyClientQueryAddr: envDefault("IDELY_CLIENTQUERY_ADDR", "127.0.0.1:25639"),
		IdelyAPIKey:          os.Getenv("IDELY_APIKEY"),
		IdelyLobbyCID:        envInt("IDELY_LOBBY_CID", 0),

		IdelyPollSeconds:        envInt("IDELY_POLL_SECONDS", 5),
		IdelyIdleMinutes:        envInt("IDELY_IDLE_MINUTES", 15),
		IdelyActiveGraceSeconds: envInt("IDELY_ACTIVE_GRACE_SECONDS", 6),
		IdelyExcludeUIDs:        envList("IDELY_EXCLUDE_UIDS", nil),
		IdelyOnlyCIDs:           envIntList("IDELY_ONLY_CIDS"),

		IdelyAPIURL:          envDefault("IDELY_API_URL", "http://ts3audiobot:58913"),
		IdelyAPIUser:         os.Getenv("IDELY_API_USER"),
		IdelyAPIToken:        os.Getenv("IDELY_API_TOKEN"),
		IdelyBotID:           envInt("IDELY_BOT_ID", 0),
		IdelyBotUID:          os.Getenv("IDELY_BOT_UID"),
		IdelyVolume:          envInt("IDELY_VOLUME", 100),
		IdelyMaxBots:         envInt("IDELY_MAX_BOTS", 4),
		IdelyTalkStopMS:      envInt("IDELY_TALK_STOP_MS", 2500),
		IdelyTalkCooldownSec: envInt("IDELY_TALK_COOLDOWN_SEC", 60),
		// TS3AudioBot's `plugin load` resolves by index, not name, so default to
		// the first plugin (0). Override with IDELY_PLUGIN if you load others.
		IdelyPluginRef: envDefault("IDELY_PLUGIN", "0"),
		IdelyNames: envList("IDELY_NAMES", []string{
			"lofi.radio", "Idle Vibes", "Afk Lounge", "sleepy.fm",
			"Chillhop", "night owl", "vinyl dreams", "Static & Rain",
		}),
		// By default Idely scans IdelyTracksDir for track files (so a whole
		// downloaded library is picked up automatically). Set IDELY_TRACKS to an
		// explicit comma-separated list to override the scan.
		IdelyTracks:    envList("IDELY_TRACKS", nil),
		IdelyTracksDir: envDefault("IDELY_TRACKS_DIR", "/audio"),
	}
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch v {
	case "":
		return def
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// envIntList parses a comma-separated list of integers, skipping blanks and
// non-numeric entries. An unset/empty variable yields nil.
func envIntList(key string) []int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil
	}
	var out []int
	for _, p := range strings.Split(v, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			out = append(out, n)
		}
	}
	return out
}

// envList parses a comma-separated, lowercased list; empty falls back to def.
func envList(key string, def []string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	var out []string
	for _, p := range strings.Split(v, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return def
	}
	return out
}

// loadDotEnv reads a simple KEY=VALUE file and sets any variables that are not
// already present in the environment.
func loadDotEnv(path string) {
	// #nosec G304 -- Path is a hardcoded or explicitly trusted environment variable
	f, err := os.Open(path)
	if err != nil {
		return // No file is fine; env vars may be supplied another way.
	}
	defer func() { _ = f.Close() }()

	log.Printf("Loading configuration from %s", path)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}
