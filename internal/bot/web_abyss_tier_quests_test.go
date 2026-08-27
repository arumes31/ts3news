package bot

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssTierQuestsUseSuccessfulBankedRuns(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	mock.ExpectQuery("SELECT COALESCE\\(tier,'normal'\\),COALESCE\\(MAX\\(depth\\),0\\)").
		WithArgs("delver").
		WillReturnRows(sqlmock.NewRows([]string{"tier", "depth"}).
			AddRow("normal", 14).
			AddRow("nightmare", 19).
			AddRow("hell", 31))

	quests, err := (&Bot{DB: db}).abyssTierQuests(context.Background(), "delver")
	if err != nil {
		t.Fatal(err)
	}
	nightmare, _ := abyssTierQuestByKey(quests, "nightmare")
	hell, _ := abyssTierQuestByKey(quests, "hell")
	insanity, _ := abyssTierQuestByKey(quests, "insanity")
	if !nightmare.Complete || hell.Complete || !insanity.Complete {
		t.Fatalf("quests = %#v", quests)
	}
	if hell.progress() != "Best banked 19 / 20" {
		t.Fatalf("hell progress = %q", hell.progress())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssTierViewsExposeQuestCopy(t *testing.T) {
	quests := abyssTierQuestCatalog()
	quests[1].BestDepth = 7
	views := abyssTierViewsWithQuests(quests, nil)
	if views[1].Unlocked || views[1].UnlockQuest != "Bank a Normal run at depth 10+" || views[1].QuestProgress != "Best banked 7 / 10" {
		t.Fatalf("nightmare view = %#v", views[1])
	}
}
