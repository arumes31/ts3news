package content

import (
	"fmt"
	"hash/fnv"
)

const (
	// AbyssTreeArtCellSize is deliberately tiny: the web renders most passive
	// nodes at 8-20 CSS pixels, so a crisp 16-bit silhouette reads better than a
	// large downscaled illustration.
	AbyssTreeArtCellSize = 16
	AbyssTreeArtColumns  = 32
	AbyssTreeArtRows     = 32
	AbyssTreeArtCapacity = AbyssTreeArtColumns * AbyssTreeArtRows
)

// AbyssTreeArtSheets is the stable sector-to-atlas contract used by the
// generator, embedded asset routes, and browser renderer.
var AbyssTreeArtSheets = [treeSectors]string{
	"tree_war",
	"tree_vitality",
	"tree_shadow",
	"tree_arcane",
	"tree_fortune",
	"tree_void",
}

func assignAbyssTreeArt(nodes []TreeNode) {
	counts := [treeSectors]int{}
	for index := range nodes {
		node := &nodes[index]
		if node.Sector < 0 || node.Sector >= len(AbyssTreeArtSheets) {
			continue
		}
		node.ArtSheet = AbyssTreeArtSheets[node.Sector]
		node.ArtCell = counts[node.Sector]
		node.ArtSignature = abyssTreeArtSignature(*node)
		counts[node.Sector]++
	}
}

func abyssTreeArtSignature(node TreeNode) string {
	hash := fnv.New64a()
	_, _ = fmt.Fprintf(hash, "%d:%d:%d:%s:%s", node.ID, node.Sector, node.Ring, node.Type, node.Name)
	return fmt.Sprintf("tree-%016x", hash.Sum64())
}
