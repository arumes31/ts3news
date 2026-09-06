package bot

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

type abyssClassPending struct {
	UID     string           `json:"uid"`
	Class   string           `json:"class"`
	Owner   string           `json:"owner"`
	Seed    [2]uint64        `json:"seed"`
	Depth   int              `json:"depth"`
	Kills   []abyssClassKill `json:"kills"`
	Cleared bool             `json:"cleared"`
}

func abyssClassPendingKey(p abyssClassPending, raw []byte) string {
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("abyss_class_pending:%s:%x", p.UID, sum[:16])
}
func (b *Bot) deliverAbyssClassPending(ctx context.Context, p abyssClassPending) (int64, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return 0, err
	}
	key := abyssClassPendingKey(p, raw)
	// Journal before touching totals. A failed credit transaction can be replayed
	// after process restart; different partial results have separate immutable keys.
	if _, err = b.DB.ExecContext(ctx, "INSERT INTO app_meta (key,value) VALUES ($1,$2) ON CONFLICT (key) DO NOTHING", key, string(raw)); err != nil {
		return 0, err
	}
	delta, err := b.grantAbyssClassCredit(ctx, p.UID, p.Class, p.Owner, p.Seed, p.Depth, p.Kills, p.Cleared)
	if err != nil {
		return 0, err
	}
	if _, err = b.DB.ExecContext(ctx, "DELETE FROM app_meta WHERE key=$1", key); err != nil {
		return delta, err
	}
	return delta, nil
}
func (b *Bot) flushAbyssClassPending(ctx context.Context, uid string) {
	rows, err := b.DB.QueryContext(ctx, "SELECT key,value FROM app_meta WHERE starts_with(key,$1) ORDER BY key LIMIT 20", "abyss_class_pending:"+uid+":")
	if err != nil {
		return
	}
	pending := []abyssClassPending{}
	for rows.Next() {
		var key, raw string
		if rows.Scan(&key, &raw) != nil {
			continue
		}
		var p abyssClassPending
		if json.Unmarshal([]byte(raw), &p) == nil && p.UID == uid && p.Depth > 0 && key == abyssClassPendingKey(p, []byte(raw)) {
			pending = append(pending, p)
		}
	}
	readErr := rows.Err()
	rows.Close()
	if readErr != nil {
		return
	}
	for _, p := range pending {
		if _, err := b.deliverAbyssClassPending(ctx, p); err != nil {
			log.Printf("abyss class XP journal retry failed: %v", err)
			return
		}
	}
}

var abyssClassPendingRetries sync.Map

func (b *Bot) retryAbyssClassPending(p abyssClassPending) {
	// Before the first successful journal insert, recovery is process-lifetime
	// only. Do not start workers for test/simulation Bots without a live config.
	if b.Cfg == nil {
		return
	}
	raw, _ := json.Marshal(p)
	key := struct {
		Bot *Bot
		Key string
	}{b, abyssClassPendingKey(p, raw)}
	if _, loaded := abyssClassPendingRetries.LoadOrStore(key, true); loaded {
		return
	}
	go func() {
		defer abyssClassPendingRetries.Delete(key)
		for {
			time.Sleep(30 * time.Second)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, err := b.deliverAbyssClassPending(ctx, p)
			cancel()
			if err == nil {
				return
			}
		}
	}()
}
