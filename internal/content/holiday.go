package content

import (
	"time"
	"ts3news/internal/i18n"
)

// Theme describes a seasonal/holiday flavour applied to messages.
type Theme struct {
	Name    string // e.g. "Christmas"
	Emoji   string // a decorative emoji used as a prefix
	Banner  string // a themed greeting line that replaces the random greeting
	Signoff string // a themed closing line appended to the PM
}

// CurrentTheme returns the active holiday theme for the given date, or nil if it
// is an ordinary day. Date ranges are inclusive.
func CurrentTheme(now time.Time) *Theme {
	y := now.Year()
	md := func(m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, now.Location()) }
	day := time.Date(y, now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	between := func(start, end time.Time) bool {
		return !day.Before(start) && !day.After(end)
	}

	switch {
	case between(md(time.December, 1), md(time.December, 26)):
		return &Theme{
			Name:    i18n.T("content.holiday.christmas.name"),
			Emoji:   "🎄",
			Banner:  i18n.T("content.holiday.christmas.banner"),
			Signoff: i18n.T("content.holiday.christmas.signoff"),
		}
	case between(md(time.December, 27), md(time.December, 31)), between(md(time.January, 1), md(time.January, 2)):
		return &Theme{
			Name:    i18n.T("content.holiday.new_year.name"),
			Emoji:   "🎆",
			Banner:  i18n.T("content.holiday.new_year.banner"),
			Signoff: i18n.T("content.holiday.new_year.signoff"),
		}
	case between(md(time.October, 24), md(time.October, 31)):
		return &Theme{
			Name:    i18n.T("content.holiday.halloween.name"),
			Emoji:   "🎃",
			Banner:  i18n.T("content.holiday.halloween.banner"),
			Signoff: i18n.T("content.holiday.halloween.signoff"),
		}
	case day.Equal(md(time.February, 14)):
		return &Theme{
			Name:    i18n.T("content.holiday.valentines_day.name"),
			Emoji:   "❤️",
			Banner:  i18n.T("content.holiday.valentines_day.banner"),
			Signoff: i18n.T("content.holiday.valentines_day.signoff"),
		}
	case day.Equal(md(time.April, 1)):
		return &Theme{
			Name:    i18n.T("content.holiday.april_fools.name"),
			Emoji:   "🃏",
			Banner:  i18n.T("content.holiday.april_fools.banner"),
			Signoff: i18n.T("content.holiday.april_fools.signoff"),
		}
	}

	// Easter (computus): the week leading up to and including Easter Sunday.
	easter := easterSunday(y, now.Location())
	if between(easter.AddDate(0, 0, -6), easter.AddDate(0, 0, 1)) {
		return &Theme{
			Name:    i18n.T("content.holiday.easter.name"),
			Emoji:   "🐰",
			Banner:  i18n.T("content.holiday.easter.banner"),
			Signoff: i18n.T("content.holiday.easter.signoff"),
		}
	}

	return nil
}

// easterSunday returns the date of Easter Sunday for the given year using the
// anonymous Gregorian algorithm (Meeus/Jones/Butcher).
func easterSunday(year int, loc *time.Location) time.Time {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h + l - 7*m + 114) % 31) + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
}
