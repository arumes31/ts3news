package bot

import (
	"fmt"
	"strings"
	"time"
)

const (
	abyssChangelogPath    = "webassets/abyss_changelog.md"
	abyssChangelogMaxSize = 64 << 10
)

type abyssChangelogRelease struct {
	Date  string
	Title string
	Items []string
}

func loadAbyssChangelog() ([]abyssChangelogRelease, error) {
	content, err := webAssets.ReadFile(abyssChangelogPath)
	if err != nil {
		return nil, fmt.Errorf("read embedded changelog: %w", err)
	}
	if len(content) > abyssChangelogMaxSize {
		return nil, fmt.Errorf("embedded changelog exceeds %d bytes", abyssChangelogMaxSize)
	}
	changelog := strings.TrimSpace(string(content))
	if !strings.HasPrefix(changelog, "# The Abyss") {
		return nil, fmt.Errorf("embedded changelog has no Abyss heading")
	}

	releases := make([]abyssChangelogRelease, 0)
	for _, line := range strings.Split(changelog, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "## ") {
			heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			date, title, found := strings.Cut(heading, " — ")
			if !found || strings.TrimSpace(title) == "" {
				return nil, fmt.Errorf("changelog release %q has no title", heading)
			}
			date = strings.TrimSpace(date)
			if _, err := time.Parse(time.DateOnly, date); err != nil {
				return nil, fmt.Errorf("changelog release %q has invalid date: %w", heading, err)
			}
			releases = append(releases, abyssChangelogRelease{
				Date:  date,
				Title: strings.TrimSpace(title),
			})
			continue
		}
		if strings.HasPrefix(line, "- ") && len(releases) > 0 {
			item := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			if item != "" {
				last := len(releases) - 1
				releases[last].Items = append(releases[last].Items, item)
			}
		}
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("embedded changelog has no releases")
	}
	for _, release := range releases {
		if len(release.Items) == 0 {
			return nil, fmt.Errorf("changelog release %q has no items", release.Title)
		}
	}
	return releases, nil
}
