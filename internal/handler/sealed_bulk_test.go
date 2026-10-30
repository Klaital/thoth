package handler

import (
	"testing"

	"github.com/klaital/thoth/internal/model"
)

func TestParseBulkSealedLine(t *testing.T) {
	tests := []struct {
		line string
		want bulkSealedLine
	}{
		{
			line: "Reality Fracture Play Booster Display",
			want: bulkSealedLine{ItemType: model.SealedItemTypePlayBoosterBox, SetName: "Reality Fracture"},
		},
		{
			line: "Reality Fracture Collector Booster Display",
			want: bulkSealedLine{ItemType: model.SealedItemTypeCollectorBoosterBox, SetName: "Reality Fracture"},
		},
		{
			line: "Duskmourn: House of Horror Play Booster Box",
			want: bulkSealedLine{ItemType: model.SealedItemTypePlayBoosterBox, SetName: "Duskmourn: House of Horror"},
		},
		{
			line: "Foundations Collector Booster Pack",
			want: bulkSealedLine{ItemType: model.SealedItemTypeCollectorBooster, SetName: "Foundations"},
		},
		{
			line: "Secret Lair x The Hobbit: Smaug's Spoils Foil Edition",
			want: bulkSealedLine{ItemType: model.SealedItemTypeSecretLair, Name: "Secret Lair x The Hobbit: Smaug's Spoils Foil Edition"},
		},
		{
			line: "Artist Series: Ian Miller Traditional Foil Edition",
			want: bulkSealedLine{ItemType: model.SealedItemTypeSecretLair, Name: "Artist Series: Ian Miller Traditional Foil Edition"},
		},
		{
			line: "Secret Lair x Lofi Girl: Beats to Cast To 💫 Traditional Foil Edition",
			want: bulkSealedLine{ItemType: model.SealedItemTypeSecretLair, Name: "Secret Lair x Lofi Girl: Beats to Cast To 💫 Traditional Foil Edition"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := parseBulkSealedLine(tt.line); got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
