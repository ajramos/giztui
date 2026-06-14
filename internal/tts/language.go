package tts

import (
	"strings"

	"github.com/abadojack/whatlanggo"
)

// isoToWhatlang maps ISO 639-1 codes to whatlanggo languages for the languages people commonly
// have system voices for. Extend as needed.
var isoToWhatlang = map[string]whatlanggo.Lang{
	"en": whatlanggo.Eng,
	"es": whatlanggo.Spa,
	"fr": whatlanggo.Fra,
	"de": whatlanggo.Deu,
	"it": whatlanggo.Ita,
	"pt": whatlanggo.Por,
	"nl": whatlanggo.Nld,
	"ru": whatlanggo.Rus,
	"ja": whatlanggo.Jpn,
	"ko": whatlanggo.Kor,
	"zh": whatlanggo.Cmn,
	"tr": whatlanggo.Tur,
	"pl": whatlanggo.Pol,
	"sv": whatlanggo.Swe,
}

func normISO(code string) string { return strings.ToLower(strings.TrimSpace(code)) }

// DetectLanguage returns the ISO 639-1 code of text's dominant language, restricted to candidates
// (the languages you have a voice/model for). Restricting the choice makes detection far more
// accurate on short or mixed text than open-set detection. With a single candidate it returns that
// language without detecting; with none, or when it can't decide, it returns "".
func DetectLanguage(text string, candidates []string) string {
	wl := map[whatlanggo.Lang]bool{}
	var lastISO string
	for _, c := range candidates {
		iso := normISO(c)
		if l, ok := isoToWhatlang[iso]; ok {
			wl[l] = true
			lastISO = iso
		}
	}
	switch len(wl) {
	case 0:
		return ""
	case 1:
		return lastISO // only one candidate → no need to detect
	}
	info := whatlanggo.DetectWithOptions(text, whatlanggo.Options{Whitelist: wl})
	return info.Lang.Iso6391() // "" when undecidable
}
