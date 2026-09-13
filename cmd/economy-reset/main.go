// Command economy-reset is an explicit, offline maintenance operation.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
	"ts3news/internal/db"
)

func main() {
	scope := flag.String("scope", "", "economy (currencies and stock, preserves character progression)")
	confirmed := flag.Bool("writers-stopped-and-backup-restored", false, "confirm offline writers and successful backup restore drill")
	flag.Parse()
	if !*confirmed || *scope == "" {
		fmt.Fprintln(os.Stderr, "explicit scope and verified offline restore drill are required")
		os.Exit(2)
	}
	database, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = database.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := db.Migrate(database); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	applied, err := db.ResetEconomy(ctx, database, *scope)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Economy reset version %d applied=%v scope=%s\n", db.EconomyResetVersion, applied, *scope)
}
