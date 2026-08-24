package bot

import (
	"net/http"
	"sort"
	"time"

	"ts3news/internal/content"
)

type abyssTreeNodeQuote struct {
	ID     int `json:"id"`
	Points int `json:"points"`
}

type abyssTreeMutationQuote struct {
	Action        string               `json:"action"`
	Affected      []int                `json:"affected_nodes"`
	NodeCosts     []abyssTreeNodeQuote `json:"node_costs"`
	AffectedCount int                  `json:"affected_count"`
	PointTotal    int                  `json:"point_total"`
	GoldTotal     int64                `json:"gold_total"`
	TokenTotal    int                  `json:"token_total"`
	Free          bool                 `json:"free"`
}

func (s *WebServer) handleAbyssTreeRefundPreview(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var request struct {
		NodeID int `json:"node_id"`
	}
	if readJSON(r, &request) != nil || content.AbyssTree().Node(request.NodeID) == nil {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown node"})
		return
	}
	allocated, err := s.bot.loadTreeAllocatedContext(r.Context(), uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	allocatedSet := make(map[int]bool, len(allocated))
	for _, id := range allocated {
		allocatedSet[id] = true
	}
	if !allocatedSet[request.NodeID] {
		writeJSON(w, map[string]any{"ok": false, "error": "node is not allocated"})
		return
	}
	tree := content.AbyssTree()
	quote := buildAbyssTreeRefundQuote(tree, allocated, request.NodeID, abyssNodeOfTheDay(time.Now()))
	writeJSON(w, map[string]any{"ok": true, "quote": quote})
}

func buildAbyssTreeRefundQuote(
	tree *content.AbyssTreeData,
	allocated []int,
	target int,
	dayID int,
) abyssTreeMutationQuote {
	affected := treeRefundSet(tree, allocated, target)
	quote := abyssTreeMutationQuote{Action: "cascade_refund", Affected: affected, AffectedCount: len(affected)}
	for _, id := range affected {
		points := 1
		if node := tree.Node(id); node != nil {
			points = treeNodeCostFor(node, dayID)
		}
		quote.PointTotal += points
		quote.NodeCosts = append(quote.NodeCosts, abyssTreeNodeQuote{ID: id, Points: points})
	}
	quote.GoldTotal = int64(quote.PointTotal) * abyssTreeRefundGoldPerPoint
	sort.Slice(quote.NodeCosts, func(i, j int) bool { return quote.NodeCosts[i].ID < quote.NodeCosts[j].ID })
	return quote
}

func (s *WebServer) handleAbyssTreeRespecQuote(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	allocated, err := s.bot.loadTreeAllocatedContext(r.Context(), uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if len(allocated) == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "nothing to respec"})
		return
	}
	quote := abyssTreeMutationQuote{
		Action: "full_respec", Affected: append([]int(nil), allocated...),
		AffectedCount: len(allocated), PointTotal: s.bot.treeSpentEx(uid, allocated),
		TokenTotal: abyssTreeRespecTokens, Free: s.bot.abyssFreeRespecAvailable(uid),
	}
	if quote.Free {
		quote.TokenTotal = 0
	}
	sort.Ints(quote.Affected)
	writeJSON(w, map[string]any{"ok": true, "quote": quote})
}
