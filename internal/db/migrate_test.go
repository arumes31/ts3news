package db

import (
	"testing"

	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func TestEmbeddedMigrationsLoad(t *testing.T) {
	// The source rejects duplicate versions before PostgreSQL is contacted.
	// Independently added migrations must remain loadable after a branch merge.
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := source.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := source.First(); err != nil {
		t.Fatal(err)
	}
}
