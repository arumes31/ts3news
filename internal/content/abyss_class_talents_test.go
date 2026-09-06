package content

import "testing"

func TestAbyssTalentCatalogAndBudgets(t *testing.T) {
	seen := map[string]bool{}
	for _, class := range AbyssClasses() {
		trees := []AbyssTalentTree{AbyssTalents(class.ID)}
		for _, sub := range class.Subclasses {
			trees = append(trees, AbyssTalents(sub.ID))
		}
		for _, tree := range trees {
			want := 15
			if tree.Subclass != "" {
				want = 18
			}
			if len(tree.Nodes) != want {
				t.Fatalf("%s nodes=%d", tree.ID, len(tree.Nodes))
			}
			for _, n := range tree.Nodes {
				if seen[n.ID] || n.Name == "" || n.Description == "" || n.Effect == "" || n.Art == "" {
					t.Fatal(n)
				}
				seen[n.ID] = true
			}
		}
	}
	if len(seen) != 306 {
		t.Fatal(len(seen))
	}
}
func TestAbyssTalentPointCurve(t *testing.T) {
	for i, floors := range []int64{1, 5, 15, 35, 75, 575, 1325, 2325, 3825, 5825, 8325, 11325, 14825, 18825, 23825} {
		if AbyssClassPoints(floors*1000-1) != i || AbyssClassPoints(floors*1000) != i+1 {
			t.Fatal(i, floors)
		}
	}
	if AbyssClassPoints(1<<62) != 15 || AbyssClassPoints(-1) != 0 {
		t.Fatal("cap")
	}
}
