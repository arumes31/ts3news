package bot

import (
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
)

var abyssMaterialFlowCounters sync.Map

var abyssForgeMaterialCostPattern = regexp.MustCompile(`(\d+)\s*(🌫️|🔷|🟣|💠)`)

func recordAbyssForgeMaterialCost(uid, cost string) {
	materialByIcon := map[string]string{"🌫️": "dust", "🔷": "shard", "🟣": "core", "💠": "prism"}
	for _, match := range abyssForgeMaterialCostPattern.FindAllStringSubmatch(cost, -1) {
		amount, _ := strconv.Atoi(match[1])
		if mat := materialByIcon[match[2]]; mat != "" {
			recordAbyssMaterialFlow(uid, mat, "sink", amount)
		}
	}
}

func recordAbyssMaterialFlow(uid, mat, direction string, amount int) {
	if amount <= 0 || (direction != "source" && direction != "sink") {
		return
	}
	key := uid + "\x00" + mat + "\x00" + direction
	value, _ := abyssMaterialFlowCounters.LoadOrStore(key, &atomic.Int64{})
	value.(*atomic.Int64).Add(int64(amount))
}

func abyssMaterialFlow(uid string) map[string]map[string]int64 {
	out := map[string]map[string]int64{}
	prefix := uid + "\x00"
	abyssMaterialFlowCounters.Range(func(rawKey, rawValue any) bool {
		key := rawKey.(string)
		if len(key) <= len(prefix) || key[:len(prefix)] != prefix {
			return true
		}
		remainder := key[len(prefix):]
		cut := -1
		for i := len(remainder) - 1; i >= 0; i-- {
			if remainder[i] == 0 {
				cut = i
				break
			}
		}
		if cut <= 0 {
			return true
		}
		mat, direction := remainder[:cut], remainder[cut+1:]
		if out[mat] == nil {
			out[mat] = map[string]int64{}
		}
		out[mat][direction] = rawValue.(*atomic.Int64).Load()
		return true
	})
	return out
}

func (s *WebServer) handleAbyssMaterialFlow(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "flow": abyssMaterialFlow(uid)})
}
