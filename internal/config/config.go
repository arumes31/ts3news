package config

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	TS3Host            string
	TS3Port            int
	TS3User            string
	TS3Pass            string
	TS3Identity        string
	TS3Nickname        string
	TS3ServerID        int
	CheckIntervalHours int

	// ClientQuery / poke settings
	ClientQueryAddr string // host:port of the ClientQuery telnet interface
	ClientQueryINI  string // path to clientquery.ini (to read the API key)
	APIKey          string // optional explicit API key (overrides the .ini)
	TargetNick      string // if set, only poke clients with this nickname (testing)
	PokeDelayMS     int    // delay between consecutive pokes, to avoid anti-flood
}

func LoadConfig() *Config {
	// Load variables from config.env (if present) so the bot works both inside
	// Docker (where compose injects the file as real env) and when run directly.
	// Existing environment variables always take precedence over the file.
	loadDotEnv("config.env")

	port, _ := strconv.Atoi(os.Getenv("TS3_PORT"))
	sid, _ := strconv.Atoi(os.Getenv("TS3_SERVER_ID"))
	interval, _ := strconv.Atoi(os.Getenv("CHECK_INTERVAL_HOURS"))
	if interval == 0 {
		interval = 12
	}
	pokeDelay, _ := strconv.Atoi(os.Getenv("POKE_DELAY_MS"))
	if pokeDelay == 0 {
		pokeDelay = 1200
	}

	return &Config{
		TS3Host:            os.Getenv("TS3_HOST"),
		TS3Port:            port,
		TS3User:            os.Getenv("TS3_USER"),
		TS3Pass:            os.Getenv("TS3_PASS"),
		TS3Identity:        os.Getenv("TS3_IDENTITY"),
		TS3Nickname:        envDefault("TS3_NICKNAME", "MrFree"),
		TS3ServerID:        sid,
		CheckIntervalHours: interval,
		ClientQueryAddr:    envDefault("CLIENTQUERY_ADDR", "127.0.0.1:25639"),
		ClientQueryINI:     envDefault("CLIENTQUERY_INI", "/root/.ts3client/clientquery.ini"),
		APIKey:             os.Getenv("TS3_APIKEY"),
		TargetNick:         os.Getenv("TS3_TARGET_NICK"),
		PokeDelayMS:        pokeDelay,
	}
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// loadDotEnv reads a simple KEY=VALUE file and sets any variables that are not
// already present in the environment. Lines starting with '#' and blank lines
// are ignored, as are surrounding quotes around values.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // No file is fine; env vars may be supplied another way.
	}
	defer f.Close()

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
			os.Setenv(key, value)
		}
	}
}
