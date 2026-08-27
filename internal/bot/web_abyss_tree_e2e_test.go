//go:build e2e

package bot

import (
	"net/http"
	"sort"
	"time"

	"ts3news/internal/content"
)

func registerAbyssTreeE2EFixture(mux *http.ServeMux, server *WebServer) {
	tree := content.AbyssTree()
	edges := abyssTreeE2EEdges(tree)
	mux.HandleFunc("/abyss/tree", func(w http.ResponseWriter, _ *http.Request) {
		progression := buildAbyssTreeProgression(tree, nil, 100, 50, 0, 500, 0)
		rewards := buildAbyssProgressionPointRewards(4, 150, map[string]int64{"normal": 64})
		progression.PointSources = append(progression.PointSources, rewards.Sources...)
		progression.TotalPoints += rewards.Points
		fixture := map[string]any{
			"Title": "Abyss Skill Web", "Nav": "abyss", "U": abyssGoldenFixture(false)["U"],
			"Nodes": tree.Nodes, "Edges": edges, "Catalog": tree.CatalogSummary(),
			"TreeEffects": content.TreeEffects(), "SkillDetails": map[int]any{},
			"Progression": progression, "TreeFeaturesEnabled": true, "Portals": tree.Portals,
			"Allocated": []int{}, "Points": 1000, "Used": 0, "Avail": 1000,
			"BonusPct": map[string]string{}, "BonusPctRaw": map[string]float64{}, "Bonus": content.Stats{},
			"BestDepth": 50, "RespecTk": abyssTreeRespecTokens, "Stats": abyssStats{BestDepth: 50},
			"Spec": "", "DelverTalentDefs": content.DeepDelverTalents, "DelverTalentLevels": map[string]int{},
			"SpecTalentDefs": content.SpecTalents, "Tokens": int64(1_000), "NodeGates": abyssUpgradeMinDepth,
			"TalentMaxLevel": content.TalentMaxLevel,
			"LimitBreakID": content.NodeLimitBreak, "SanctuaryID": content.NodeSecretSanctuary,
			"Sockets": "{}", "ActiveKeystoneExpiry": "", "ActiveKeystoneCooldown": "",
			"Jewels": map[string]int{}, "Loadouts": map[string]int{}, "LoadoutNames": map[string]string{},
			"SeasonalTree": abyssSeasonalTree(time.Now()), "NodeOfDay": abyssNodeOfTheDay(time.Now()),
			"FreeRespec": false, "MasteryShards": 0, "PrestigeMemory": 0,
			"KeystoneActiveSecs": 0, "KeystoneCooldownSecs": 0,
			"Paragon": abyssParagonView{}, "BestiaryTalents": []abyssBestiaryTalent{},
		}
		if err := server.tmpl.ExecuteTemplate(w, "abysstree", fixture); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func abyssTreeE2EEdges(tree *content.AbyssTreeData) [][2]int {
	edges := make([][2]int, 0, len(tree.Nodes)*2)
	for node, neighbors := range tree.Adj {
		for _, neighbor := range neighbors {
			if node < neighbor {
				edges = append(edges, [2]int{node, neighbor})
			}
		}
	}
	sort.Slice(edges, func(left, right int) bool {
		if edges[left][0] == edges[right][0] {
			return edges[left][1] < edges[right][1]
		}
		return edges[left][0] < edges[right][0]
	})
	return edges
}
