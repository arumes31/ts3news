package bot

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
)

func TestRiftUltimateOwnershipIncludesInactiveCatalogEntriesAndReportsErrors(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	b := &Bot{DB: db}
	mock.ExpectQuery("SELECT ultimate_id FROM user_ultimate_skills WHERE client_uid=\\$1 ORDER BY obtained, ultimate_id").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"ultimate_id"}).AddRow("ULT_REVIVAL").AddRow("removed"))
	names, err := b.riftOwnedUltimates(context.Background(), "owner")
	if err != nil || len(names) != 1 || names[0] != "Divine Revival" {
		t.Fatalf("ownership: %v %v", names, err)
	}
	mock.ExpectQuery("SELECT ultimate_id").WithArgs("empty").WillReturnRows(sqlmock.NewRows([]string{"ultimate_id"}))
	names, err = b.riftOwnedUltimates(context.Background(), "empty")
	if err != nil || names == nil || len(names) != 0 {
		t.Fatal("known empty ownership must remain distinct from unavailable")
	}
	mock.ExpectQuery("SELECT ultimate_id").WithArgs("failed").WillReturnError(errors.New("offline"))
	if _, err = b.riftOwnedUltimates(context.Background(), "failed"); err == nil {
		t.Fatal("query failure reported as no ownership")
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
