package i18n

// NumberFormat defines locale-specific number formatting.
type NumberFormat struct {
	DecimalSep     string          // "." or ","
	GroupSep       string          // "," or "." or " " or ""
	GroupSize      int             // 3 (standard), 4 (some Asian locales)
	SuffixK        string          // "k", "Tsd" (German), "тыс" (Russian)
	SuffixM        string          // "M", "Mio" (German), "М" (Russian)
	SuffixB        string          // "B", "Mrd" (German), "Млрд" (Russian)
	GoldSuffix     string          // "g", "G" (German), "з" (Russian)
	DigitTransform func(rune) rune // transforms ASCII digits to locale-specific digits (e.g. Arabic, Thai)
}

// defaultNumberFormat returns the English (US) number format.
func defaultNumberFormat() NumberFormat {
	return NumberFormat{
		DecimalSep: ".",
		GroupSep:   ",",
		GroupSize:  3,
		SuffixK:    "k",
		SuffixM:    "M",
		SuffixB:    "B",
		GoldSuffix: "g",
	}
}

// localeNumberFormats holds the format for each locale.
var localeNumberFormats = map[LocaleID]NumberFormat{
	LocaleEnUS: {DecimalSep: ".", GroupSep: ",", GroupSize: 3, SuffixK: "k", SuffixM: "M", SuffixB: "B", GoldSuffix: "g"},
	LocaleDeDE: {DecimalSep: ",", GroupSep: ".", GroupSize: 3, SuffixK: "k", SuffixM: "Mio", SuffixB: "Mrd", GoldSuffix: "G"},
	LocaleEsES: {DecimalSep: ",", GroupSep: ".", GroupSize: 3, SuffixK: "k", SuffixM: "M", SuffixB: "M", GoldSuffix: "g"},
	LocaleFrFR: {DecimalSep: ",", GroupSep: " ", GroupSize: 3, SuffixK: "k", SuffixM: "M", SuffixB: "Mrd", GoldSuffix: "g"},
	LocaleItIT: {DecimalSep: ",", GroupSep: ".", GroupSize: 3, SuffixK: "k", SuffixM: "M", SuffixB: "Mld", GoldSuffix: "g"},
	LocalePtBR: {DecimalSep: ",", GroupSep: ".", GroupSize: 3, SuffixK: "k", SuffixM: "M", SuffixB: "B", GoldSuffix: "g"},
	LocaleJaJP: {DecimalSep: ".", GroupSep: ",", GroupSize: 3, SuffixK: "万", SuffixM: "M", SuffixB: "B", GoldSuffix: "g"},
	LocaleKoKR: {DecimalSep: ".", GroupSep: ",", GroupSize: 3, SuffixK: "천", SuffixM: "M", SuffixB: "B", GoldSuffix: "g"},
	LocaleZhCN: {DecimalSep: ".", GroupSep: ",", GroupSize: 3, SuffixK: "千", SuffixM: "M", SuffixB: "B", GoldSuffix: "g"},
	LocaleZhTW: {DecimalSep: ".", GroupSep: ",", GroupSize: 3, SuffixK: "千", SuffixM: "M", SuffixB: "B", GoldSuffix: "g"},
	LocaleRuRU: {DecimalSep: ",", GroupSep: " ", GroupSize: 3, SuffixK: "тыс", SuffixM: "М", SuffixB: "Млрд", GoldSuffix: "з"},
	LocalePlPL: {DecimalSep: ",", GroupSep: " ", GroupSize: 3, SuffixK: "tys", SuffixM: "M", SuffixB: "Mld", GoldSuffix: "g"},
	LocaleTrTR: {DecimalSep: ",", GroupSep: ".", GroupSize: 3, SuffixK: "k", SuffixM: "M", SuffixB: "M", GoldSuffix: "g"},
	LocaleNlNL: {DecimalSep: ",", GroupSep: ".", GroupSize: 3, SuffixK: "k", SuffixM: "M", SuffixB: "Mld", GoldSuffix: "g"},
	LocaleSvSE: {DecimalSep: ",", GroupSep: " ", GroupSize: 3, SuffixK: "k", SuffixM: "M", SuffixB: "Mrd", GoldSuffix: "g"},
	LocaleCsCZ: {DecimalSep: ",", GroupSep: " ", GroupSize: 3, SuffixK: "k", SuffixM: "M", SuffixB: "Mld", GoldSuffix: "g"},
	LocaleArSA: {DecimalSep: ".", GroupSep: ",", GroupSize: 3, SuffixK: "k", SuffixM: "M", SuffixB: "B", GoldSuffix: "g", DigitTransform: arabicDigits},
	LocaleThTH: {DecimalSep: ".", GroupSep: ",", GroupSize: 3, SuffixK: "k", SuffixM: "M", SuffixB: "B", GoldSuffix: "g", DigitTransform: thaiDigits},
	LocaleViVN: {DecimalSep: ",", GroupSep: ".", GroupSize: 3, SuffixK: "k", SuffixM: "M", SuffixB: "B", GoldSuffix: "g"},
	LocaleHiIN: {DecimalSep: ".", GroupSep: ",", GroupSize: 3, SuffixK: "k", SuffixM: "M", SuffixB: "B", GoldSuffix: "g"},
}

// numberFormatForLocale returns the NumberFormat for a locale, with en_US fallback.
func numberFormatForLocale(id LocaleID) NumberFormat {
	if nf, ok := localeNumberFormats[id]; ok {
		return nf
	}
	return defaultNumberFormat()
}

// arabicDigits transforms ASCII digits (0-9) to Arabic-Indic digits (٠-٩).
func arabicDigits(r rune) rune {
	if r >= '0' && r <= '9' {
		return r - '0' + '٠'
	}
	return r
}

// thaiDigits transforms ASCII digits (0-9) to Thai digits (๐-๙).
func thaiDigits(r rune) rune {
	if r >= '0' && r <= '9' {
		return r - '0' + '๐'
	}
	return r
}

// applyDigitTransform applies the DigitTransform function to all digits in s.
func applyDigitTransform(s string, transform func(rune) rune) string {
	if transform == nil {
		return s
	}
	return string([]rune(transformRunes([]rune(s), transform)))
}

func transformRunes(runes []rune, transform func(rune) rune) []rune {
	for i, r := range runes {
		runes[i] = transform(r)
	}
	return runes
}
