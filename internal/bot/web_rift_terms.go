package bot

import "ts3news/internal/content"

// riftClassNames keeps current and historical class labels on the Abyss catalog.
func riftClassNames() map[string]string {
	names := map[string]string{}
	for _, class := range content.AbyssClasses() {
		names[class.ID] = class.Name
		for _, subclass := range class.Subclasses {
			names[subclass.ID] = subclass.Name
		}
	}
	return names
}
