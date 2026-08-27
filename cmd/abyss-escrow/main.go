// Command abyss-escrow creates and verifies active Abyss escrow snapshots and
// runs non-destructive restore drills against the current PostgreSQL schema.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ts3news/internal/config"
	databasepkg "ts3news/internal/db"

	_ "github.com/lib/pq"
)

const defaultSnapshotLimit = int64(256 << 20)

type commandOptions struct {
	mode     string
	path     string
	timeout  time.Duration
	maxBytes int64
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "abyss-escrow:", err)
		os.Exit(1)
	}
}

func run(parent context.Context, args []string, stdout, stderr io.Writer) error {
	options, err := parseOptions(args, stderr)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, options.timeout)
	defer cancel()

	switch options.mode {
	case "verify":
		snapshot, err := readSnapshot(options.path, options.maxBytes)
		if err != nil {
			return err
		}
		return printSnapshotSummary(stdout, "verified", snapshot)
	case "backup", "check", "drill":
		// These modes need the database; verification intentionally does not.
	default:
		return fmt.Errorf("unsupported mode %q", options.mode)
	}

	database, err := openDatabase(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	switch options.mode {
	case "check":
		report, err := databasepkg.CheckAbyssEscrowIntegrity(ctx, database)
		if err != nil {
			return err
		}
		if err := writeJSON(stdout, report); err != nil {
			return err
		}
		if !report.Healthy() {
			return errors.New("live Abyss escrow integrity check failed")
		}
		return nil
	case "backup":
		report, err := databasepkg.CheckAbyssEscrowIntegrity(ctx, database)
		if err != nil {
			return err
		}
		snapshot, err := databasepkg.ExportAbyssEscrowSnapshot(ctx, database, time.Now())
		if err != nil {
			return err
		}
		if err := writeSnapshot(options.path, snapshot); err != nil {
			return err
		}
		if err := printSnapshotSummary(stdout, "written", snapshot); err != nil {
			return err
		}
		if err := databasepkg.ValidateAbyssEscrowSnapshot(snapshot); err != nil {
			return fmt.Errorf("snapshot preserved but relational validation failed: %w", err)
		}
		if !report.Healthy() {
			return errors.New("snapshot preserved but live Abyss escrow integrity check failed")
		}
		return nil
	case "drill":
		snapshot, err := readSnapshot(options.path, options.maxBytes)
		if err != nil {
			return err
		}
		counts, err := databasepkg.DrillAbyssEscrowRestore(ctx, database, snapshot)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "restore drill passed and rolled back: active=%d loot=%d sessions=%d members=%d checksum=%s\n", counts.Active, counts.Loot, counts.Sessions, counts.Members, snapshot.Checksum)
		return err
	}
	return nil
}

func parseOptions(args []string, stderr io.Writer) (commandOptions, error) {
	options := commandOptions{}
	flags := flag.NewFlagSet("abyss-escrow", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&options.mode, "mode", "check", "check, backup, verify, or drill")
	flags.StringVar(&options.path, "file", "", "snapshot file (required for backup, verify, and drill)")
	flags.DurationVar(&options.timeout, "timeout", 45*time.Second, "operation deadline")
	flags.Int64Var(&options.maxBytes, "max-bytes", defaultSnapshotLimit, "maximum accepted snapshot size")
	if err := flags.Parse(args); err != nil {
		return commandOptions{}, err
	}
	options.mode = strings.ToLower(strings.TrimSpace(options.mode))
	options.path = strings.TrimSpace(options.path)
	if flags.NArg() != 0 {
		return commandOptions{}, errors.New("positional arguments are not supported")
	}
	if options.timeout <= 0 {
		return commandOptions{}, errors.New("timeout must be positive")
	}
	if options.maxBytes <= 0 {
		return commandOptions{}, errors.New("max-bytes must be positive")
	}
	switch options.mode {
	case "check", "backup", "verify", "drill":
	default:
		return commandOptions{}, fmt.Errorf("unsupported mode %q", options.mode)
	}
	if options.mode != "check" && options.path == "" {
		return commandOptions{}, fmt.Errorf("-file is required for %s mode", options.mode)
	}
	return options, nil
}

func openDatabase(ctx context.Context) (*sql.DB, error) {
	cfg := config.LoadConfig()
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	database, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}
	return database, nil
}

func writeSnapshot(path string, snapshot databasepkg.AbyssEscrowSnapshot) (err error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("creating snapshot %q: %w", path, err)
	}
	complete := false
	defer func() {
		if !complete {
			_ = file.Close()
			_ = os.Remove(path)
		}
	}()
	if err := databasepkg.EncodeAbyssEscrowSnapshot(file, snapshot); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("syncing snapshot %q: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("closing snapshot %q: %w", path, err)
	}
	complete = true
	return nil
}

func readSnapshot(path string, maxBytes int64) (databasepkg.AbyssEscrowSnapshot, error) {
	file, err := os.Open(path)
	if err != nil {
		return databasepkg.AbyssEscrowSnapshot{}, fmt.Errorf("opening snapshot %q: %w", path, err)
	}
	defer func() { _ = file.Close() }()
	return databasepkg.DecodeAbyssEscrowSnapshot(file, maxBytes)
}

func printSnapshotSummary(writer io.Writer, verb string, snapshot databasepkg.AbyssEscrowSnapshot) error {
	_, err := fmt.Fprintf(writer, "snapshot %s: active=%d loot=%d sessions=%d members=%d checksum=%s\n", verb, snapshot.Counts.Active, snapshot.Counts.Loot, snapshot.Counts.Sessions, snapshot.Counts.Members, snapshot.Checksum)
	if err != nil {
		return fmt.Errorf("writing snapshot summary: %w", err)
	}
	return nil
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("writing integrity report: %w", err)
	}
	return nil
}
