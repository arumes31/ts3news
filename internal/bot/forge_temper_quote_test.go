package bot

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/content"
)

func TestTemperQuoteUsesPityAndMatchingGuard(t *testing.T) {
	for _, tc := range []struct {
		name, guard string
		pity        int
		want        float64
	}{
		{"pity", "", 3, temperChance(10, 3)},
		{"matching guard", "inv:7", 0, 1},
		{"other item guard", "inv:8", 0, temperChance(10, 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, mock, done := newForge2TestServer(t)
			defer done()
			mock.ExpectQuery("SELECT temper_fail_stacks").WithArgs("player").WillReturnRows(sqlmock.NewRows([]string{"stacks"}).AddRow(tc.pity))
			rows := sqlmock.NewRows([]string{"value"})
			if tc.guard != "" {
				rows.AddRow(tc.guard)
			}
			mock.ExpectQuery("SELECT value FROM app_meta").WithArgs(forge4TemperGuardKey("player")).WillReturnRows(rows)
			chance, text, pity, err := s.temperQuoteChance(context.Background(), "player", content.Gear{Temper: 10}, abyssForgeQuoteRequest{InvID: 7})
			if err != nil || chance != tc.want || text == "" || pity == "" {
				t.Fatalf("quote: %v %q %q %v", chance, text, pity, err)
			}
			if tc.guard == "inv:7" && !strings.Contains(pity, "guard") {
				t.Fatal("guard not explained")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestTemperQuoteRejectsUnavailablePity(t *testing.T) {
	s, mock, done := newForge2TestServer(t)
	defer done()
	mock.ExpectQuery("SELECT temper_fail_stacks").WithArgs("player").WillReturnError(errors.New("unavailable"))
	if _, _, _, err := s.temperQuoteChance(context.Background(), "player", content.Gear{}, abyssForgeQuoteRequest{}); err == nil {
		t.Fatal("invented odds after read failure")
	}
}
