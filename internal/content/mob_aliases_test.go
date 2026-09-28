package content

import "testing"

func TestCanonicalMobAliasesAreDetached(t *testing.T) {
	initMobs()
	original := baseMobs
	baseMobs = append(append([]Mob(nil), baseMobs...), Mob{Name: "Alias test sentinel", Aliases: []string{"Old sentinel", "Sentinelle"}})
	defer func() { baseMobs = original }()
	var found *Mob
	for _, mob := range AbyssMobCatalog() {
		if mob.Name == "Alias test sentinel" {
			found = &mob
			break
		}
	}
	if found == nil || len(found.Aliases) != 2 || found.Aliases[0] != "Old sentinel" {
		t.Fatal("catalog lost authored aliases")
	}
	clone := found.Clone()
	clone.Aliases[0] = "clone changed"
	if found.Aliases[0] != "Old sentinel" {
		t.Fatal("clone shares alias storage")
	}
	found.Aliases[0] = "changed"
	if baseMobs[len(baseMobs)-1].Aliases[0] != "Old sentinel" {
		t.Fatal("clone changed canonical alias")
	}
	for _, mob := range AbyssMobCatalog() {
		if mob.Name == "Alias test sentinel" && mob.Aliases[0] != "Old sentinel" {
			t.Fatal("catalog alias was mutated")
		}
	}
}
