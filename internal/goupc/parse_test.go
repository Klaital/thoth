package goupc

import (
	"testing"
)

func TestParseMTGProduct(t *testing.T) {
	tests := []struct {
		name        string
		inputName   string
		inputDesc   string
		wantName    string
		wantSetName string
		wantType    string
	}{
		{
			name:        "collector booster box",
			inputName:   "Magic: The Gathering Commander Masters Collector Booster Box",
			wantName:    "Commander Masters Collector Booster Box",
			wantSetName: "Commander Masters",
			wantType:    "collector_booster_box",
		},
		{
			name:        "play booster box",
			inputName:   "Magic The Gathering Wilds of Eldraine Play Booster Box",
			wantName:    "Wilds of Eldraine Play Booster Box",
			wantSetName: "Wilds of Eldraine",
			wantType:    "play_booster_box",
		},
		{
			name:        "bundle",
			inputName:   "Magic: The Gathering - The Lost Caverns of Ixalan Bundle",
			wantName:    "The Lost Caverns of Ixalan Bundle",
			wantSetName: "The Lost Caverns of Ixalan",
			wantType:    "bundle",
		},
		{
			name:        "prerelease pack",
			inputName:   "Magic: The Gathering Duskmourn Prerelease Pack",
			wantName:    "Duskmourn Prerelease Pack",
			wantSetName: "Duskmourn",
			wantType:    "prerelease_pack",
		},
		{
			name:        "commander precon",
			inputName:   "Magic: The Gathering March of the Machine Commander Deck",
			wantName:    "March of the Machine Commander Deck",
			wantSetName: "March of the Machine",
			wantType:    "commander_precon",
		},
		{
			name:        "secret lair magic prefix",
			inputName:   "Magic: The Gathering Secret Lair Drop: Universes Beyond: Fallout",
			wantName:    "Secret Lair Drop: Universes Beyond: Fallout",
			wantSetName: "",
			wantType:    "secret_lair",
		},
		{
			name:        "secret lair wotc prefix with leading dot",
			inputName:   ".Wizards of the Coast Secret Lair X Stardew Valley: Life In Pelican Town Foil Edition",
			wantName:    "Secret Lair X Stardew Valley: Life In Pelican Town Foil Edition",
			wantSetName: "",
			wantType:    "secret_lair",
		},
		{
			name:        "pipe-delimited description fallback",
			inputName:   "Magic: The Gathering",
			inputDesc:   "Magic: The Gathering | Wilds of Eldraine | Bundle | 9 Play Boosters",
			wantName:    "Wilds of Eldraine Bundle",
			wantSetName: "Wilds of Eldraine",
			wantType:    "bundle",
		},
		{
			name:      "non-mtg product",
			inputName: "Funko Pop! Spider-Man #1234",
			wantName:  "Funko Pop! Spider-Man #1234",
			wantType:  "",
		},
		{
			name:        "wotc magic gathering compound prefix",
			inputName:   "Wizards of the Coast Magic Gathering Hobbit Bundle",
			wantName:    "Hobbit Bundle",
			wantSetName: "Hobbit",
			wantType:    "bundle",
		},
		{
			name:        "magic colon no space prefix",
			inputName:   "Magic:The Gathering The Lord of the Rings: Tales of Middle-earth Bundle",
			wantName:    "The Lord of the Rings: Tales of Middle-earth Bundle",
			wantSetName: "The Lord of the Rings: Tales of Middle-earth",
			wantType:    "bundle",
		},
		{
			name:        "draft booster box",
			inputName:   "Magic: The Gathering Innistrad Crimson Vow Draft Booster Box",
			wantName:    "Innistrad Crimson Vow Draft Booster Box",
			wantSetName: "Innistrad Crimson Vow",
			wantType:    "draft_booster_box",
		},
		{
			name:        "set booster",
			inputName:   "Magic: The Gathering Kaldheim Set Booster",
			wantName:    "Kaldheim Set Booster",
			wantSetName: "Kaldheim",
			wantType:    "set_booster",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseMTGProduct(tc.inputName, tc.inputDesc)
			if got.Name != tc.wantName {
				t.Errorf("Name: got %q, want %q", got.Name, tc.wantName)
			}
			if got.SetName != tc.wantSetName {
				t.Errorf("SetName: got %q, want %q", got.SetName, tc.wantSetName)
			}
			if got.ItemType != tc.wantType {
				t.Errorf("ItemType: got %q, want %q", got.ItemType, tc.wantType)
			}
		})
	}
}
