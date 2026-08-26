package content

import "testing"

func TestAbyssTreePixelArtMetadataIsCompleteAndUnique(t *testing.T) {
	t.Parallel()

	tree := AbyssTree()
	counts := [treeSectors]int{}
	signatures := make(map[string]int, len(tree.Nodes))
	for _, node := range tree.Nodes {
		if node.ArtSheet != AbyssTreeArtSheets[node.Sector] {
			t.Errorf("node %d sheet = %q, want %q", node.ID, node.ArtSheet, AbyssTreeArtSheets[node.Sector])
		}
		if node.ArtCell != counts[node.Sector] {
			t.Errorf("node %d cell = %d, want %d", node.ID, node.ArtCell, counts[node.Sector])
		}
		if previous, duplicate := signatures[node.ArtSignature]; duplicate {
			t.Errorf("nodes %d and %d share art signature %q", previous, node.ID, node.ArtSignature)
		}
		signatures[node.ArtSignature] = node.ID
		counts[node.Sector]++
	}
	for sector, count := range counts {
		if count == 0 || count > AbyssTreeArtCapacity {
			t.Errorf("sector %d uses %d of %d atlas cells", sector, count, AbyssTreeArtCapacity)
		}
	}
	if len(signatures) != len(tree.Nodes) {
		t.Fatalf("unique signatures = %d, want %d", len(signatures), len(tree.Nodes))
	}
}
