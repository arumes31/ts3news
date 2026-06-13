package i18n

// PluralCategory represents a CLDR plural category.
type PluralCategory string

const (
	PluralZero  PluralCategory = "zero"
	PluralOne   PluralCategory = "one"
	PluralTwo   PluralCategory = "two"
	PluralFew   PluralCategory = "few"
	PluralMany  PluralCategory = "many"
	PluralOther PluralCategory = "other"
)

// PluralRule maps a count to a CLDR plural category.
type PluralRule func(n int) PluralCategory

// pluralRules holds the plural rule for each supported locale.
var pluralRules = map[LocaleID]PluralRule{
	LocaleEnUS: ruleOneOther,
	LocaleDeDE: ruleOneOther,
	LocaleEsES: ruleOneOther,
	LocaleItIT: ruleOneOther,
	LocaleNlNL: ruleOneOther,
	LocaleSvSE: ruleOneOther,
	LocaleCsCZ: ruleCzech,
	LocaleViVN: ruleOther,
	LocaleHiIN: ruleOneOther,
	LocaleFrFR: ruleFrenchOneOther,
	LocalePtBR: ruleFrenchOneOther,
	LocalePlPL: rulePolish,
	LocaleRuRU: ruleRussian,
	LocaleTrTR: ruleOneOther,
	LocaleJaJP: ruleOther,
	LocaleKoKR: ruleOther,
	LocaleZhCN: ruleOther,
	LocaleZhTW: ruleOther,
	LocaleThTH: ruleOther,
	LocaleArSA: ruleArabic,
}

// pluralRuleForLocale returns the plural rule for the given locale.
func pluralRuleForLocale(id LocaleID) PluralRule {
	if rule, ok := pluralRules[id]; ok {
		return rule
	}
	return ruleOneOther
}

// Form returns the plural category for the given count.
func (r PluralRule) Form(n int) PluralCategory {
	if r == nil {
		return ruleOneOther(n)
	}
	return r(n)
}

// ruleOneOther: one=1, other=everything else.
func ruleOneOther(n int) PluralCategory {
	if n == 1 {
		return PluralOne
	}
	return PluralOther
}

// ruleFrenchOneOther: one=0 or 1, other=everything else.
func ruleFrenchOneOther(n int) PluralCategory {
	if n == 0 || n == 1 {
		return PluralOne
	}
	return PluralOther
}

// rulePolish: one=1, few=2-4 (except 12-14), many=0,5-21,25-31,..., other=rest.
func rulePolish(n int) PluralCategory {
	if n == 1 {
		return PluralOne
	}
	mod10 := n % 10
	mod100 := n % 100
	if mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14) {
		return PluralFew
	}
	if (mod10 == 0 || mod10 == 1) || (mod10 >= 5 && mod10 <= 9) ||
		(mod100 >= 12 && mod100 <= 14) {
		return PluralMany
	}
	return PluralOther
}

// ruleRussian: one=1,21,31,..., few=2-4,22-24,32-34,..., many=0,5-20,25-30,...
func ruleRussian(n int) PluralCategory {
	mod10 := n % 10
	mod100 := n % 100
	if mod10 == 1 && mod100 != 11 {
		return PluralOne
	}
	if mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14) {
		return PluralFew
	}
	if mod10 == 0 || (mod10 >= 5 && mod10 <= 9) ||
		(mod100 >= 11 && mod100 <= 14) {
		return PluralMany
	}
	return PluralOther
}

// ruleCzech: zero=0, one=1, few=2-4, other=rest.
func ruleCzech(n int) PluralCategory {
	if n == 0 {
		return PluralZero
	}
	if n == 1 {
		return PluralOne
	}
	if n >= 2 && n <= 4 {
		return PluralFew
	}
	return PluralOther
}

// ruleArabic: full CLDR Arabic plural rules.
//   zero  → n = 0
//   one   → n = 1
//   two   → n = 2
//   few   → n % 100 ∈ 3..10
//   many  → n % 100 ∈ 11..99
//   other → everything else (n % 100 = 0,1,2 handled above; n ≥ 100 with mod 0)
func ruleArabic(n int) PluralCategory {
	if n == 0 {
		return PluralZero
	}
	if n == 1 {
		return PluralOne
	}
	if n == 2 {
		return PluralTwo
	}
	mod100 := n % 100
	if mod100 >= 3 && mod100 <= 10 {
		return PluralFew
	}
	if mod100 >= 11 && mod100 <= 99 {
		return PluralMany
	}
	// n % 100 is 0 or 1 or 2 for n ≥ 100 (e.g. 100, 101, 102, 200…)
	// These fall into "other" per CLDR
	return PluralOther
}

// ruleOther: always "other" (Asian languages with no plural distinction).
func ruleOther(n int) PluralCategory {
	return PluralOther
}
