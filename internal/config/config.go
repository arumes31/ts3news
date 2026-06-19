package config

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

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
}

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
