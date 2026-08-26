package bot

import (
	"slices"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestApplyAbyssSkillPriorityReconcilesStoredOrder(t *testing.T) {
	t.Parallel()

	skills := []content.Skill{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	got := abyssSkillPriorityIDs(applyAbyssSkillPriority(skills, []string{"c", "retired", "c", "a"}))
	want := []string{"c", "a", "b"}
	if !slices.Equal(got, want) {
		t.Fatalf("priority = %v, want %v", got, want)
	}
}

func TestAbyssSkillPriorityViewPreservesDefaultRanks(t *testing.T) {
	t.Parallel()

	base := []content.Skill{
		{ID: "a", Name: "Alpha", Type: content.SkillPhysical, Rarity: content.RarityCommon},
		{ID: "b", Name: "Beta", Type: content.SkillMagic, Rarity: content.RarityRare},
	}
	view := abyssSkillPriorityViewForSkills(base, []content.Skill{base[1], base[0]})
	if !view.Customized || len(view.Items) != 2 {
		t.Fatalf("view = %+v, want customized two-item view", view)
	}
	if view.Items[0].ID != "b" || view.Items[0].Priority != 1 || view.Items[0].DefaultPriority != 2 {
		t.Fatalf("first item = %+v, want b with current rank 1 and default rank 2", view.Items[0])
	}
}

func TestAbyssPrioritizedSkillsLoadsPersistedOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(`SELECT value FROM app_meta WHERE key=\$1`).
		WithArgs("abyss_skill_priority:user").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`["b","a"]`))
	skills := []content.Skill{{ID: "a"}, {ID: "b"}}
	got := abyssSkillPriorityIDs((&Bot{DB: db}).abyssPrioritizedSkills("user", skills))
	if !slices.Equal(got, []string{"b", "a"}) {
		t.Fatalf("priority = %v, want [b a]", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAbyssSkillPriority(t *testing.T) {
	t.Parallel()

	skills := []content.Skill{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	tests := []struct {
		name      string
		requested []string
		want      []string
		wantError bool
	}{
		{name: "complete order", requested: []string{"c", "b", "a"}, want: []string{"c", "b", "a"}},
		{name: "partial order appends missing", requested: []string{"b"}, want: []string{"b", "a", "c"}},
		{name: "empty order restores default", want: []string{"a", "b", "c"}},
		{name: "unknown skill", requested: []string{"forged"}, wantError: true},
		{name: "duplicate skill", requested: []string{"a", "a"}, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := validateAbyssSkillPriority(test.requested, skills)
			if (err != nil) != test.wantError {
				t.Fatalf("error = %v, wantError %t", err, test.wantError)
			}
			if !test.wantError && !slices.Equal(got, test.want) {
				t.Fatalf("priority = %v, want %v", got, test.want)
			}
		})
	}
}

func TestFirstReadyAffordableSkillUsesPriority(t *testing.T) {
	t.Parallel()

	skills := []content.Skill{
		{ID: "first", ManaCost: 60},
		{ID: "second", ManaCost: 20},
		{ID: "third", ManaCost: 10},
	}
	cost := func(base int) int { return base }
	selected := firstReadyAffordableSkill(skills, map[string]int{"second": 2}, 30, cost)
	if selected == nil || selected.ID != "third" {
		t.Fatalf("selected = %+v, want third", selected)
	}
	if selected := firstReadyAffordableSkill(skills, nil, 5, cost); selected != nil {
		t.Fatalf("selected = %+v, want nil", selected)
	}
}
