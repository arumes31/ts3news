package bot

import (
	"testing"

	"ts3news/internal/content"
)

func TestArmorySceneForEveryClassAndSubclass(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		t.Run(class.ID, func(t *testing.T) {
			scene := armorySceneForState(abyssClassState{Class: class.ID})
			if scene.Name != class.Name || scene.Asset == "" {
				t.Fatalf("class scene = %+v", scene)
			}
			for _, sub := range class.Subclasses {
				t.Run(sub.ID, func(t *testing.T) {
					subScene := armorySceneForState(abyssClassState{Class: class.ID, Selected: sub.ID})
					if subScene.Name != class.Name+" / "+sub.Name || subScene.Asset == scene.Asset {
						t.Fatalf("subclass should have its own scene: %+v", subScene)
					}
				})
			}
		})
	}
}

func TestArmorySceneFallback(t *testing.T) {
	for _, state := range []abyssClassState{{}, {Class: "unknown", Selected: "../../bad"}} {
		scene := armorySceneForState(state)
		if scene.Asset != "/static/armory_hall.webp" || scene.Name != "" {
			t.Fatalf("unexpected fallback: %+v", scene)
		}
	}
	if scene := armorySceneForState(abyssClassState{Class: "ranger", Selected: "unknown"}); scene.Name != "Ranger" {
		t.Fatalf("invalid subclass should fall back to class: %+v", scene)
	}
	if scene := armorySceneForState(abyssClassState{Selected: "voidwalker"}); scene.Name != "Reaver / Voidwalker" {
		t.Fatalf("legacy subclass should resolve its parent: %+v", scene)
	}
}
