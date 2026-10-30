package goupc

import (
	"regexp"
	"strings"
)

// ParsedProduct holds the structured fields extracted from a UPC lookup result.
type ParsedProduct struct {
	Name     string // human-readable product name (e.g. "Wilds of Eldraine Bundle")
	SetName  string // set name only (e.g. "Wilds of Eldraine"); empty if undetected
	ItemType string // one of the SealedItemType string values, or "" if unrecognized
}

// mtgPrefix matches any known Wizards of the Coast / Magic product prefix.
// Handles:
//   - "Magic: The Gathering ..." (with or without colon)
//   - ".Wizards of the Coast ..." (go-upc.com sometimes adds a leading dot)
//   - "Wizards of the Coast ..."
//
// Followed by an optional " - " or " | " separator.
// mtgPrefix handles the many formats go-upc.com uses for WotC/MTG products:
//   "Magic: The Gathering ...", "Magic:The Gathering ...", "Magic Gathering ...",
//   ".Wizards of the Coast ...", "Wizards of the Coast ..."
// It is applied up to twice so "Wizards of the Coast Magic Gathering ..." is
// fully stripped in two passes.
var mtgPrefix = regexp.MustCompile(`(?i)^\.?(?:magic\s*:?\s*(?:the\s+)?gathering|wizards\s+of\s+the\s+coast)\s*[-|]?\s*`)

// itemTypeRules maps suffix keywords to SealedItemType strings.
// Ordered longest-first so "Collector Booster Box" beats "Collector Booster".
var itemTypeRules = []struct {
	keyword  string
	itemType string
}{
	{"Collector Booster Box", "collector_booster_box"},
	{"Collector Booster", "collector_booster"},
	{"Play Booster Box", "play_booster_box"},
	{"Play Booster", "play_booster"},
	{"Draft Booster Box", "draft_booster_box"},
	{"Draft Booster", "draft_booster"},
	{"Set Booster Box", "set_booster_box"},
	{"Set Booster", "set_booster"},
	{"Jumpstart Booster Box", "jumpstart_booster_box"},
	{"Jumpstart Booster", "jumpstart_booster"},
	{"Booster Box", "play_booster_box"},
	{"Booster", "play_booster"},
	{"Bundle", "bundle"},
	{"Commander Deck", "commander_precon"},
	{"Commander", "commander_precon"},
	{"Prerelease Pack", "prerelease_pack"},
	{"Prerelease", "prerelease_pack"},
	{"Secret Lair", "secret_lair"},
	{"Draft Night", "draft_night"},
}

// ParseMTGProduct extracts structured fields from the name and description
// returned by go-upc.com for a Magic: The Gathering sealed product.
//
// It tries the name field first. If no item type is detected and description is
// provided, it falls back to parsing the pipe-delimited description format used
// by many go-upc.com entries for MTG products.
func ParseMTGProduct(name, description string) ParsedProduct {
	isMTG := mtgPrefix.MatchString(name)
	if !isMTG {
		if description != "" {
			if p := parseFromPipeDesc(description); p.ItemType != "" {
				return p
			}
		}
		return ParsedProduct{Name: name}
	}

	// Strip up to two times: "Wizards of the Coast Magic Gathering ..." needs two passes.
	stripped := strings.TrimSpace(mtgPrefix.ReplaceAllString(name, ""))
	if mtgPrefix.MatchString(stripped) {
		stripped = strings.TrimSpace(mtgPrefix.ReplaceAllString(stripped, ""))
	}
	itemType, setName := detectItemType(stripped)

	if itemType == "" && description != "" {
		if alt := parseFromPipeDesc(description); alt.ItemType != "" {
			return alt
		}
	}

	return ParsedProduct{
		Name:     stripped,
		SetName:  strings.TrimSpace(setName),
		ItemType: itemType,
	}
}

// parseFromPipeDesc handles pipe-delimited descriptions like:
//
//	"Magic: The Gathering | Wilds of Eldraine | Bundle | 9 Play Boosters"
//
// It extracts the set name and product type from the first two non-MTG segments.
func parseFromPipeDesc(desc string) ParsedProduct {
	parts := strings.Split(desc, "|")
	segments := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || mtgPrefix.MatchString(p) {
			continue
		}
		segments = append(segments, p)
	}
	if len(segments) < 2 {
		return ParsedProduct{}
	}
	// Use only the first two segments: set name + product type keyword.
	combined := segments[0] + " " + segments[1]
	itemType, setName := detectItemType(combined)
	return ParsedProduct{
		Name:     combined,
		SetName:  strings.TrimSpace(setName),
		ItemType: itemType,
	}
}

// detectItemType scans text for a known product type keyword and returns
// (itemType, setName) where setName is the text with the keyword removed.
// If no keyword is found, itemType is empty and setName equals text.
func detectItemType(text string) (itemType, setName string) {
	lower := strings.ToLower(text)
	for _, rule := range itemTypeRules {
		kw := strings.ToLower(rule.keyword)
		idx := strings.LastIndex(lower, kw)
		if idx == -1 {
			continue
		}
		if idx > 0 && lower[idx-1] != ' ' {
			continue
		}
		setName = strings.TrimRight(text[:idx], " -|")
		return rule.itemType, setName
	}
	return "", text
}
