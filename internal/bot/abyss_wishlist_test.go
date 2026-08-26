package bot

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

type fixedWishlistRandom int

func (r fixedWishlistRandom) IntN(n int) int {
	if n <= 0 {
		panic("invalid random bound")
	}
	return int(r) % n
}

func wishlistTestGear(t *testing.T, match func(content.Gear) bool) content.Gear {
	t.Helper()
	for _, gear := range content.AbyssGearCatalog() {
		if match(gear) {
			return gear
		}
	}
	t.Fatal("Abyss catalog has no matching test gear")
	return content.Gear{}
}

func TestNormalizeAbyssWishlistRejectsInvalidDuplicatesAndOverflow(t *testing.T) {
	catalog := content.AbyssGearCatalog()
	if len(catalog) < 4 {
		t.Fatal("Abyss catalog must contain at least four entries")
	}
	state := normalizeAbyssWishlist(abyssWishlistState{
		GearIDs: []string{"missing", catalog[0].ID, catalog[0].ID, catalog[1].ID, catalog[2].ID, catalog[3].ID},
		Pity:    abyssWishlistPityCap + 9,
	})
	if len(state.GearIDs) != abyssWishlistLimit {
		t.Fatalf("normalized IDs = %d, want %d", len(state.GearIDs), abyssWishlistLimit)
	}
	if state.Pity != abyssWishlistPityCap {
		t.Fatalf("normalized pity = %d, want %d", state.Pity, abyssWishlistPityCap)
	}
	empty := normalizeAbyssWishlist(abyssWishlistState{Pity: 17})
	if empty.Pity != 0 {
		t.Fatalf("empty wishlist retained pity %d", empty.Pity)
	}
}

func TestToggleAbyssWishlistResetsPityAndEnforcesLimit(t *testing.T) {
	catalog := content.AbyssGearCatalog()
	state := abyssWishlistState{Pity: 12}
	for i := 0; i < abyssWishlistLimit; i++ {
		var err error
		state, err = toggleAbyssWishlist(state, catalog[i].ID)
		if err != nil {
			t.Fatalf("adding %q: %v", catalog[i].ID, err)
		}
		if state.Pity != 0 {
			t.Fatalf("add retained pity %d", state.Pity)
		}
	}
	if _, err := toggleAbyssWishlist(state, catalog[abyssWishlistLimit].ID); err == nil {
		t.Fatal("adding a fourth target succeeded")
	}
	state.Pity = 9
	state, err := toggleAbyssWishlist(state, catalog[1].ID)
	if err != nil {
		t.Fatalf("removing target: %v", err)
	}
	if state.Pity != 0 || len(state.GearIDs) != abyssWishlistLimit-1 {
		t.Fatalf("remove result = %#v", state)
	}
	if _, err := toggleAbyssWishlist(state, "not-in-catalog"); err == nil {
		t.Fatal("unknown target succeeded")
	}
}

func TestApplyAbyssWishlistGuaranteesThirtiethEligibleRoll(t *testing.T) {
	target := wishlistTestGear(t, func(content.Gear) bool { return true })
	original := target
	original.ID = "rolled-item"
	original.Name = "Rolled Item"
	original.Rarity = content.RarityEternal
	state := abyssWishlistState{GearIDs: []string{target.ID}, Pity: abyssWishlistPityCap - 1}

	got, next, hit := applyAbyssWishlist(original, content.GearDropPoolAbyss, state, "", true, fixedWishlistRandom(0))
	if !hit || got.ID != target.ID {
		t.Fatalf("guarantee = (%q, %v), want target %q", got.ID, hit, target.ID)
	}
	if got.Rarity != original.Rarity {
		t.Fatalf("guaranteed rarity = %v, want rolled %v", got.Rarity, original.Rarity)
	}
	if next.Pity != 0 {
		t.Fatalf("pity after guarantee = %d, want 0", next.Pity)
	}
}

func TestApplyAbyssWishlistPreservesSetPityPriority(t *testing.T) {
	target := wishlistTestGear(t, func(content.Gear) bool { return true })
	original := target
	original.ID = "set-pity-result"
	original.Rarity = content.RarityEternal
	state := abyssWishlistState{GearIDs: []string{target.ID}, Pity: abyssWishlistPityCap - 1}

	got, capped, hit := applyAbyssWishlist(original, content.GearDropPoolAbyss, state, "", false, fixedWishlistRandom(0))
	if hit || got.ID != original.ID || capped.Pity != abyssWishlistPityCap {
		t.Fatalf("set-pity precedence result = (%q, %#v, %v)", got.ID, capped, hit)
	}
	got, next, hit := applyAbyssWishlist(original, content.GearDropPoolAbyss, capped, "", true, fixedWishlistRandom(0))
	if !hit || got.ID != target.ID || next.Pity != 0 {
		t.Fatalf("deferred guarantee result = (%q, %#v, %v)", got.ID, next, hit)
	}
}

func TestApplyAbyssWishlistHonorsPoolCategoryAndBaseRarity(t *testing.T) {
	nonWeapon := wishlistTestGear(t, func(gear content.Gear) bool {
		return !abyssGearMatchesLootCategory(gear.Slot, "weapon")
	})
	original := nonWeapon
	original.ID = "rolled-item"
	original.Rarity = content.RarityCommon
	state := abyssWishlistState{GearIDs: []string{nonWeapon.ID}, Pity: abyssWishlistPityCap - 1}

	if _, unchanged, hit := applyAbyssWishlist(original, content.GearDropPoolStandard, state, "", true, fixedWishlistRandom(0)); hit || unchanged.Pity != state.Pity {
		t.Fatalf("standard pool advanced wishlist: %#v, hit=%v", unchanged, hit)
	}
	if _, capped, hit := applyAbyssWishlist(original, content.GearDropPoolAbyss, state, "weapon", true, fixedWishlistRandom(0)); hit || capped.Pity != abyssWishlistPityCap {
		t.Fatalf("category mismatch result = %#v, hit=%v", capped, hit)
	}
	if nonWeapon.Rarity > content.RarityCommon {
		if _, capped, hit := applyAbyssWishlist(original, content.GearDropPoolAbyss, state, "", true, fixedWishlistRandom(0)); hit || capped.Pity != abyssWishlistPityCap {
			t.Fatalf("rarity mismatch result = %#v, hit=%v", capped, hit)
		}
	}
}

func TestAbyssWishlistViewIsBoundedAndSearchable(t *testing.T) {
	target := wishlistTestGear(t, func(content.Gear) bool { return true })
	view := abyssWishlistViewFor(abyssWishlistState{GearIDs: []string{target.ID}, Pity: 15}, target.ID)
	if len(view.Selected) != 1 || view.Selected[0].ID != target.ID {
		t.Fatalf("selected = %#v", view.Selected)
	}
	if len(view.Candidates) > abyssWishlistPageSize {
		t.Fatalf("candidate count = %d", len(view.Candidates))
	}
	if view.PityPct != 50 || view.Limit != abyssWishlistLimit {
		t.Fatalf("view counters = %#v", view)
	}
	for _, candidate := range view.Candidates {
		if candidate.ID == target.ID {
			t.Fatal("selected target leaked into candidates")
		}
	}
}

func TestLoadAndSaveAbyssWishlist(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	bot := &Bot{DB: database}
	target := wishlistTestGear(t, func(content.Gear) bool { return true })
	state := abyssWishlistState{GearIDs: []string{target.ID}, Pity: 7}
	raw := `{"gear_ids":["` + target.ID + `"],"pity":7}`
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM app_meta WHERE key=$1")).
		WithArgs(abyssWishlistKey("player")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(raw))
	if got := bot.loadAbyssWishlist("player"); !abyssWishlistStatesEqual(got, state) {
		t.Fatalf("loaded state = %#v, want %#v", got, state)
	}
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs(abyssWishlistKey("player"), raw).
		WillReturnResult(sqlmock.NewResult(1, 1))
	if err := saveAbyssWishlist(database, "player", state); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAbyssWishlistTreatsMissingRowAsEmpty(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectQuery("SELECT value FROM app_meta").WillReturnError(sql.ErrNoRows)
	if got := (&Bot{DB: database}).loadAbyssWishlist("missing"); len(got.GearIDs) != 0 || got.Pity != 0 {
		t.Fatalf("missing state = %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
}

func TestEscrowAbyssLootCommitsWishlistStateAtomically(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	target := wishlistTestGear(t, func(content.Gear) bool { return true })
	target.Rarity = content.RarityCommon
	state := abyssWishlistState{GearIDs: []string{target.ID}, Pity: 8}
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO abyss_escrow_loot").
		WithArgs("player", "gear", "drop", sqlmock.AnyArg(), 12).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs(abyssWishlistKey("player"), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	grant := abyssLootGrant{Type: "gear", Gear: &target, WishlistState: &state}
	if !(&Bot{DB: database}).escrowAbyssLoot("player", 12, "drop", grant) {
		t.Fatal("atomic escrow failed")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEscrowAbyssLootRollsBackWhenWishlistSaveFails(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	target := wishlistTestGear(t, func(content.Gear) bool { return true })
	state := abyssWishlistState{GearIDs: []string{target.ID}, Pity: 8}
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO abyss_escrow_loot").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO app_meta").
		WillReturnError(errors.New("write failed"))
	mock.ExpectRollback()
	grant := abyssLootGrant{Type: "gear", Gear: &target, WishlistState: &state}
	if (&Bot{DB: database}).escrowAbyssLoot("player", 12, "drop", grant) {
		t.Fatal("escrow succeeded after wishlist write failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
