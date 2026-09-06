package bot

import "ts3news/internal/content"

type armoryScene struct {
	Asset string
	Name  string
}

// Resolve only catalog IDs; an active subclass takes precedence over its class.
func armorySceneForState(state abyssClassState) armoryScene {
	if sub, ok := content.AbyssSubclassByID(state.Selected); ok {
		class, _ := content.AbyssClassByID(sub.ClassID)
		return armoryScene{Asset: "/static/armory_" + sub.ID + ".webp", Name: class.Name + " / " + sub.Name}
	}
	if class, ok := content.AbyssClassByID(state.Class); ok {
		asset := "/static/armory_" + class.ID + ".webp"
		if class.ID == "warrior" {
			asset = "/static/armory_hall.webp"
		}
		return armoryScene{Asset: asset, Name: class.Name}
	}
	return armoryScene{Asset: "/static/armory_hall.webp"}
}
