package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"ts3news/internal/content"
)

const abyssTreePlanMaxNodes = 6000

type abyssTreePlanDraft struct {
	Name       string `json:"name"`
	IDs        []int  `json:"ids"`
	LayoutHash string `json:"layout_hash"`
	UpdatedAt  string `json:"updated_at"`
}

type abyssTreePlanAnalysis struct {
	IDs           []int              `json:"ids"`
	Missing       []int              `json:"missing"`
	Added         []int              `json:"added"`
	Removed       []int              `json:"removed"`
	CurrentCost   int                `json:"current_cost"`
	PlannedCost   int                `json:"planned_cost"`
	Available     int                `json:"available_points"`
	Connected     bool               `json:"connected"`
	MaxDepthGate  int                `json:"max_depth_gate"`
	DepthRecord   int                `json:"depth_record"`
	CurrentStats  content.Stats      `json:"current_stats"`
	PlannedStats  content.Stats      `json:"planned_stats"`
	StatDelta     content.Stats      `json:"stat_delta"`
	CurrentPct    map[string]float64 `json:"current_pct"`
	PlannedPct    map[string]float64 `json:"planned_pct"`
	PctDelta      map[string]float64 `json:"pct_delta"`
	NodeCosts     map[int]int        `json:"node_costs"`
	Warnings      []string           `json:"warnings"`
	Valid         bool               `json:"valid"`
	LayoutHash    string             `json:"layout_hash"`
	SchemaVersion int                `json:"schema_version"`
}

func abyssTreeDraftsKey(uid string) string { return "abyss_tree_plan_drafts_" + uid }

func treeNodeRequiredDepth(node *content.TreeNode) int {
	if node == nil {
		return 0
	}
	if node.ID == treeNodeVictorsTrophy {
		return 25
	}
	if node.Type == content.TreeNodeKeystone {
		return 30
	}
	if node.Ring > 20 {
		return 20
	}
	if node.Ring > 10 {
		return 10
	}
	return 0
}

func analyzeAbyssTreePlan(tree *content.AbyssTreeData, active, requested []int, points, bestDepth, memory, dayID int) abyssTreePlanAnalysis {
	analysis := abyssTreePlanAnalysis{
		Available: points, DepthRecord: bestDepth, Connected: true,
		LayoutHash: tree.TopologyHash(), SchemaVersion: content.TreeCatalogSchemaVersion,
		CurrentPct: make(map[string]float64), PlannedPct: make(map[string]float64), PctDelta: make(map[string]float64),
		NodeCosts: make(map[int]int),
	}
	activeSet := make(map[int]bool, len(active))
	for _, id := range active {
		if tree.Node(id) != nil {
			activeSet[id] = true
		}
	}
	plannedSet := make(map[int]bool, len(requested))
	for _, id := range requested {
		if plannedSet[id] {
			continue
		}
		plannedSet[id] = true
		node := tree.Node(id)
		if node == nil {
			analysis.Missing = append(analysis.Missing, id)
			continue
		}
		analysis.IDs = append(analysis.IDs, id)
		analysis.PlannedStats = analysis.PlannedStats.Add(node.Stats)
		for key, value := range node.Pct {
			analysis.PlannedPct[key] += value
		}
		if id != memory {
			cost := treeNodeCostFor(node, dayID)
			analysis.NodeCosts[id] = cost
			analysis.PlannedCost += cost
		} else {
			analysis.NodeCosts[id] = 0
		}
		if gate := treeNodeRequiredDepth(node); gate > analysis.MaxDepthGate {
			analysis.MaxDepthGate = gate
		}
	}
	for id := range activeSet {
		node := tree.Node(id)
		analysis.CurrentStats = analysis.CurrentStats.Add(node.Stats)
		for key, value := range node.Pct {
			analysis.CurrentPct[key] += value
		}
		if id != memory {
			analysis.CurrentCost += treeNodeCostFor(node, dayID)
		}
		if !plannedSet[id] {
			analysis.Removed = append(analysis.Removed, id)
		}
	}
	for _, id := range analysis.IDs {
		if !activeSet[id] {
			analysis.Added = append(analysis.Added, id)
		}
	}
	analysis.Connected = len(analysis.IDs) == 0 || treeConnectedSet(tree, plannedSet)
	analysis.StatDelta = subtractTreeStats(analysis.PlannedStats, analysis.CurrentStats)
	for key, value := range analysis.CurrentPct {
		analysis.PctDelta[key] -= value
	}
	for key, value := range analysis.PlannedPct {
		analysis.PctDelta[key] += value
	}
	for key, value := range analysis.PctDelta {
		if value == 0 {
			delete(analysis.PctDelta, key)
		}
	}
	sort.Ints(analysis.IDs)
	sort.Ints(analysis.Missing)
	sort.Ints(analysis.Added)
	sort.Ints(analysis.Removed)
	if len(analysis.Missing) > 0 {
		analysis.Warnings = append(analysis.Warnings, fmt.Sprintf("%d node IDs are missing from this layout", len(analysis.Missing)))
	}
	if !analysis.Connected {
		analysis.Warnings = append(analysis.Warnings, "planned nodes are not connected to the root")
	}
	if analysis.PlannedCost > points {
		analysis.Warnings = append(analysis.Warnings, fmt.Sprintf("plan costs %d points but only %d are available", analysis.PlannedCost, points))
	}
	if analysis.MaxDepthGate > bestDepth {
		analysis.Warnings = append(analysis.Warnings, fmt.Sprintf("plan requires Abyss floor %d; record is %d", analysis.MaxDepthGate, bestDepth))
	}
	analysis.Valid = len(analysis.IDs) > 0 && len(analysis.Missing) == 0 && analysis.Connected && analysis.PlannedCost <= points && analysis.MaxDepthGate <= bestDepth
	return analysis
}

func abyssTreePlanCommitError(analysis abyssTreePlanAnalysis) string {
	if len(analysis.IDs) == 0 {
		return "empty build"
	}
	if len(analysis.Missing) > 0 {
		return fmt.Sprintf("node %d does not exist on this layout", analysis.Missing[0])
	}
	if !analysis.Connected {
		return "build is not connected to the root"
	}
	if analysis.MaxDepthGate > analysis.DepthRecord {
		return fmt.Sprintf("build requires Abyss Floor %d", analysis.MaxDepthGate)
	}
	if analysis.PlannedCost > analysis.Available {
		return fmt.Sprintf("build costs %d points — you have %d", analysis.PlannedCost, analysis.Available)
	}
	return ""
}

func subtractTreeStats(a, b content.Stats) content.Stats {
	return content.Stats{
		HP: a.HP - b.HP, STR: a.STR - b.STR, DEF: a.DEF - b.DEF, SPD: a.SPD - b.SPD,
		LCK: a.LCK - b.LCK, INT: a.INT - b.INT, STA: a.STA - b.STA, CRT: a.CRT - b.CRT,
		DGE: a.DGE - b.DGE, MNA: a.MNA - b.MNA, CHA: a.CHA - b.CHA, STN: a.STN - b.STN,
		SHN: a.SHN - b.SHN, HGR: a.HGR - b.HGR,
	}
}

func (s *WebServer) currentAbyssTreePlan(ctx context.Context, uid string, ids []int) (abyssTreePlanAnalysis, error) {
	active, err := s.bot.loadTreeAllocatedContext(ctx, uid)
	if err != nil {
		return abyssTreePlanAnalysis{}, err
	}
	var bestDepth int
	if err := s.bot.DB.QueryRowContext(ctx, "SELECT COALESCE(abyss_best_depth, 0) FROM users WHERE client_uid=$1", uid).Scan(&bestDepth); err != nil {
		return abyssTreePlanAnalysis{}, err
	}
	return analyzeAbyssTreePlan(content.AbyssTree(), active, ids, s.bot.treePointsTotal(uid), bestDepth,
		s.bot.abyssPrestigeMemoryNode(uid), abyssNodeOfTheDay(time.Now())), nil
}

func (s *WebServer) handleAbyssTreePlanPreview(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req struct {
		IDs []int `json:"ids"`
	}
	if readJSON(r, &req) != nil || len(req.IDs) > abyssTreePlanMaxNodes {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid or oversized plan"})
		return
	}
	analysis, err := s.currentAbyssTreePlan(r.Context(), uid, req.IDs)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "analysis": analysis})
}

func loadAbyssTreeDrafts(stored string) map[string]abyssTreePlanDraft {
	drafts := make(map[string]abyssTreePlanDraft)
	if json.Unmarshal([]byte(stored), &drafts) != nil {
		return make(map[string]abyssTreePlanDraft)
	}
	return drafts
}

func (s *WebServer) handleAbyssTreePlanDraft(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req struct {
		Action string `json:"action"`
		Slot   int    `json:"slot"`
		Name   string `json:"name"`
		IDs    []int  `json:"ids"`
	}
	if readJSON(r, &req) != nil || req.Slot < 0 || req.Slot > 5 || len(req.IDs) > abyssTreePlanMaxNodes {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	var stored string
	_ = s.bot.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssTreeDraftsKey(uid)).Scan(&stored)
	drafts := loadAbyssTreeDrafts(stored)
	if req.Action == "list" {
		writeJSON(w, map[string]any{"ok": true, "drafts": drafts})
		return
	}
	key := strconv.Itoa(req.Slot)
	if req.Action == "delete" {
		delete(drafts, key)
	} else if req.Action == "save" && req.Slot >= 1 {
		analysis, err := s.currentAbyssTreePlan(r.Context(), uid, req.IDs)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if len(analysis.Missing) > 0 || !analysis.Connected || len(analysis.IDs) == 0 {
			writeJSON(w, map[string]any{"ok": false, "error": "draft must use known nodes connected to the root"})
			return
		}
		name := strings.TrimSpace(req.Name)
		if name == "" {
			name = "Draft " + key
		}
		if len([]rune(name)) > 40 {
			name = string([]rune(name)[:40])
		}
		drafts[key] = abyssTreePlanDraft{Name: name, IDs: analysis.IDs, LayoutHash: analysis.LayoutHash, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
	} else {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown draft action"})
		return
	}
	payload, _ := json.Marshal(drafts)
	if _, err := s.bot.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1,$2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, abyssTreeDraftsKey(uid), string(payload)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "drafts": drafts})
}
