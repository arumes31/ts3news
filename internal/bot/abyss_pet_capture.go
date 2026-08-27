package bot

import (
	"fmt"
	"net/http"
	"strings"

	"ts3news/internal/content"
)

const abyssPetCaptureCap = abyssPetBaseCap

type abyssPetCaptureResult string

const (
	abyssPetCaptureRecruited abyssPetCaptureResult = "recruited"
	abyssPetCapturePending   abyssPetCaptureResult = "pending"
	abyssPetCapturePreserved abyssPetCaptureResult = "preserved"
	abyssPetCaptureFull      abyssPetCaptureResult = "full"
)

type abyssPendingPetCaptureView struct {
	Name    string
	Type    string
	Level   int
	HP      int
	MaxHP   int
	STR     int
	DEF     int
	SPD     int
	Loyalty int
}

func abyssPetCaptureLimit(mindControlLevel int) int {
	return min(abyssPetCaptureCap, max(0, mindControlLevel))
}

func abyssCanAttemptPetCapture(owned, mindControlLevel int, pendingAttempted bool) bool {
	return abyssCanAttemptPetCaptureAtLimit(owned, abyssPetCaptureLimit(mindControlLevel), pendingAttempted)
}

func abyssCanAttemptPetCaptureAtLimit(owned, limit int, pendingAttempted bool) bool {
	if limit < abyssPetBaseCap {
		return owned < limit
	}
	return !pendingAttempted
}

func (b *Bot) persistAbyssPetCapture(uid string, pet *content.Mob, limit int) (abyssPetCaptureResult, error) {
	if pet == nil || uid == "" {
		return "", fmt.Errorf("invalid pet capture")
	}
	limit = min(abyssPetMaxCap, max(0, limit))
	if limit == 0 {
		return abyssPetCaptureFull, nil
	}
	tx, err := b.DB.Begin()
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()
	var owner string
	if err := tx.QueryRow("SELECT client_uid FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&owner); err != nil {
		return "", err
	}
	var owned int
	if err := tx.QueryRow("SELECT COUNT(*) FROM user_pets WHERE client_uid=$1", uid).Scan(&owned); err != nil {
		return "", err
	}
	result := abyssPetCaptureRecruited
	if owned < limit {
		profile, encodeErr := encodeAbyssPetProfile(abyssPetProfile{
			Shiny: pet.PetShiny, BossVariant: pet.PetBoss, BarkStyle: "gentle",
		})
		if encodeErr != nil {
			return "", encodeErr
		}
		if _, err := tx.Exec(`INSERT INTO user_pets
			(client_uid,name,mob_type,level,hp,max_hp,str,def,spd,loyalty,autoskills)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, uid, pet.Name, string(pet.Type), pet.Level,
			max(1, pet.Stats.HP), max(1, pet.MaxHP), pet.Stats.STR, pet.Stats.DEF, pet.Stats.SPD,
			min(100, max(1, pet.Loyalty)), profile); err != nil {
			return "", err
		}
	} else if limit >= abyssPetCaptureCap {
		pendingName := pet.Name
		if pet.PetShiny {
			pendingName = "✦ " + pendingName
		}
		inserted, err := tx.Exec(`INSERT INTO abyss_pending_pet_captures
			(client_uid,name,mob_type,level,hp,max_hp,str,def,spd,loyalty)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			ON CONFLICT (client_uid) DO NOTHING`, uid, pendingName, string(pet.Type), pet.Level,
			max(1, pet.Stats.HP), max(1, pet.MaxHP), pet.Stats.STR, pet.Stats.DEF, pet.Stats.SPD,
			min(100, max(1, pet.Loyalty)))
		if err != nil {
			return "", err
		}
		changed, err := inserted.RowsAffected()
		if err != nil {
			return "", err
		}
		if changed == 1 {
			result = abyssPetCapturePending
		} else {
			result = abyssPetCapturePreserved
		}
	} else {
		result = abyssPetCaptureFull
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return result, nil
}

func (b *Bot) abyssPendingPetCapture(uid string) *abyssPendingPetCaptureView {
	view := &abyssPendingPetCaptureView{}
	err := b.DB.QueryRow(`SELECT name,mob_type,level,hp,max_hp,str,def,spd,loyalty
		FROM abyss_pending_pet_captures WHERE client_uid=$1`, uid).Scan(
		&view.Name, &view.Type, &view.Level, &view.HP, &view.MaxHP,
		&view.STR, &view.DEF, &view.SPD, &view.Loyalty,
	)
	if err != nil {
		return nil
	}
	return view
}

func (s *WebServer) handleAbyssPetCaptureResolve(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}
	var req struct {
		ReleasePetID int64 `json:"release_pet_id"`
		Decline      bool  `json:"decline"`
	}
	if readJSON(r, &req) != nil || req.ReleasePetID < 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid capture decision"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var owner string
	if err := tx.QueryRow("SELECT client_uid FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&owner); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	var pending abyssPendingPetCaptureView
	if err := tx.QueryRow(`SELECT name,mob_type,level,hp,max_hp,str,def,spd,loyalty
		FROM abyss_pending_pet_captures WHERE client_uid=$1 FOR UPDATE`, uid).Scan(
		&pending.Name, &pending.Type, &pending.Level, &pending.HP, &pending.MaxHP,
		&pending.STR, &pending.DEF, &pending.SPD, &pending.Loyalty,
	); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "no pending capture"})
		return
	}
	if req.Decline {
		if _, err := tx.Exec("DELETE FROM abyss_pending_pet_captures WHERE client_uid=$1", uid); err != nil || tx.Commit() != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "msg": pending.Name + " was released back into the Abyss."})
		return
	}
	var owned int
	if err := tx.QueryRow("SELECT COUNT(*) FROM user_pets WHERE client_uid=$1", uid).Scan(&owned); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	stableLimit, err := abyssPetStableLimitTx(tx, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	activeSlot := 0
	if owned >= stableLimit {
		if req.ReleasePetID <= 0 {
			writeJSON(w, map[string]any{"ok": false, "error": "choose a companion to release"})
			return
		}
		var releasedName, rawProfile string
		if err := tx.QueryRow(`SELECT name,active_slot,autoskills::text FROM user_pets
			WHERE pet_id=$1 AND client_uid=$2 FOR UPDATE`, req.ReleasePetID, uid).Scan(&releasedName, &activeSlot, &rawProfile); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "companion not found"})
			return
		}
		if decodeAbyssPetProfile(rawProfile).busy(timeNowUTC()) {
			writeJSON(w, map[string]any{"ok": false, "error": "companion is committed to a stable activity"})
			return
		}
		if _, err := tx.Exec("DELETE FROM user_pets WHERE pet_id=$1 AND client_uid=$2", req.ReleasePetID, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		owned--
		if owned >= stableLimit {
			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "pending": true,
				"msg": releasedName + " was released. Choose another companion to make room."})
			return
		}
	}
	pendingProfile, err := encodeAbyssPetProfile(abyssPetProfile{
		Shiny: strings.HasPrefix(pending.Name, "✦ "), BossVariant: pending.Type == string(content.MobBoss), BarkStyle: "gentle",
	})
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "capture profile"})
		return
	}
	if _, err := tx.Exec(`INSERT INTO user_pets
		(client_uid,name,mob_type,level,hp,max_hp,str,def,spd,loyalty,active_slot,autoskills)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, uid, strings.TrimPrefix(pending.Name, "✦ "), pending.Type,
		pending.Level, pending.HP, pending.MaxHP, pending.STR, pending.DEF, pending.SPD,
		pending.Loyalty, activeSlot, pendingProfile); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec("DELETE FROM abyss_pending_pet_captures WHERE client_uid=$1", uid); err != nil || tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("%s joined your stable.", pending.Name)})
}
