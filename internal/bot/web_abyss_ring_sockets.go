package bot

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"

	"ts3news/internal/content"
)

const (
	abyssRingSocketMinimum = 1
	abyssRingSocketMaximum = 3
	abyssRingSocketCost    = 5
)

func isAbyssRingSlot(slot content.GearSlot) bool {
	return slot == content.SlotFinger1 || slot == content.SlotFinger2
}

// rerollAbyssRingSockets returns a copied ring with a new bounded socket count
// and gem order. Existing gems are never removed, replaced, or downgraded.
func rerollAbyssRingSockets(gear content.Gear, randomIntN func(int) int) (content.Gear, error) {
	if !isAbyssRingSlot(gear.Slot) {
		return content.Gear{}, errors.New("socket rerolling is limited to rings")
	}
	if gear.Unidentified {
		return content.Gear{}, errors.New("identify this ring before rerolling its sockets")
	}
	if len(gear.Gemstones) > abyssRingSocketMaximum {
		return content.Gear{}, fmt.Errorf("extract gems until at most %d remain before rerolling", abyssRingSocketMaximum)
	}

	minimum := max(abyssRingSocketMinimum, len(gear.Gemstones))
	result := gear
	result.Sockets = minimum + randomIntN(abyssRingSocketMaximum-minimum+1)
	result.Gemstones = append([]string(nil), gear.Gemstones...)
	for index := len(result.Gemstones) - 1; index > 0; index-- {
		other := randomIntN(index + 1)
		result.Gemstones[index], result.Gemstones[other] = result.Gemstones[other], result.Gemstones[index]
	}
	return result, nil
}

// handleAbyssRerollRingSockets rerolls a ring's socket count and occupied
// positions for five Void Shards. The item, material debit, and undo snapshot
// share one transaction under the player's Abyss mutation lock.
func (s *WebServer) handleAbyssRerollRingSockets(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var request forgeItemReq
	if !readForgeItemReq(w, r, &request) {
		return
	}
	tx, gear, rawData, ok := s.beginForgeTx(w, uid, request.InvID, request.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	rerolled, err := rerollAbyssRingSockets(gear, rand.IntN)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"shard": abyssRingSocketCost}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Void Shards (need 5)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, request.InvID, request.Slot, rawData, "ring socket reroll")
	if !saveForgeItem(w, tx, uid, request.InvID, request.Slot, rerolled) {
		return
	}
	detail := fmt.Sprintf("%s: %d→%d sockets", gear.Name, gear.Sockets, rerolled.Sockets)
	if !s.finishForge(w, tx, uid, "reroll ring sockets", detail, "5🔷") {
		return
	}
	suffix := "s"
	if rerolled.Sockets == 1 {
		suffix = ""
	}
	writeJSON(w, map[string]any{
		"ok": true, "previous_sockets": gear.Sockets, "sockets": rerolled.Sockets,
		"gemstones": rerolled.Gemstones, "materials": s.bot.loadMaterials(uid),
		"msg": fmt.Sprintf("💍 Rerolled %s to %d socket%s; every fitted gem was preserved.",
			rerolled.Name, rerolled.Sockets, suffix),
	})
}
