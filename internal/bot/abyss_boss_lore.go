package bot

type abyssBossLoreView struct {
	Boss       string
	Title      string
	Text       string
	Unlocked   bool
	FirstSlain string
}

var abyssBossLoreCatalog = []abyssBossLoreView{
	{
		Boss:  "Gorgoroth the Firelord",
		Title: "The Horn Beneath the Mountain",
		Text:  "Gorgoroth was once the furnace-keeper of a buried kingdom. When its last bell went unanswered, he taught the mountain to breathe fire and called the ashes his court.",
	},
	{
		Boss:  "Malakor the Voidweaver",
		Title: "The Weaver's Missing Thread",
		Text:  "Malakor maps every future that fails to contain him, then pulls at its weakest seam. The strand severed in your duel still twitches between worlds.",
	},
	{
		Boss:  "Azazoth the Slumbering Eye",
		Title: "A Dream That Looked Back",
		Text:  "Azazoth does not sleep; it dreams the upper world into being. Delvers who meet its gaze sometimes remember lives that never belonged to them.",
	},
	{
		Boss:  "Abyssus, Heart of the Void",
		Title: "The Pulse at the Bottom",
		Text:  "Abyssus is less a creature than the wound around which the Abyss grew. Its defeated heartbeat continues in every corridor that shifts after midnight.",
	},
}

func abyssBossLoreViews(trophies []abyssTrophyView) []abyssBossLoreView {
	firstSlain := make(map[string]string, len(trophies))
	for _, trophy := range trophies {
		firstSlain[trophy.Boss] = trophy.Date
	}
	views := make([]abyssBossLoreView, len(abyssBossLoreCatalog))
	copy(views, abyssBossLoreCatalog)
	for i := range views {
		views[i].FirstSlain = firstSlain[views[i].Boss]
		views[i].Unlocked = views[i].FirstSlain != ""
	}
	return views
}
