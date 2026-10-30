package handler

import (
	"testing"

	"github.com/klaital/thoth/internal/model"
)

func TestParseMoxfieldLine(t *testing.T) {
	tests := []struct {
		line    string
		want    bulkCardLine
		wantErr bool
	}{
		{
			line: "4 Lightning Bolt (M20) 141",
			want: bulkCardLine{Quantity: 4, Name: "Lightning Bolt", SetCode: "m20", CollectorNumber: "141", Foil: model.FoilTreatmentNonfoil},
		},
		{
			line: "1 Teferi, Hero of Dominaria (DOM) 207",
			want: bulkCardLine{Quantity: 1, Name: "Teferi, Hero of Dominaria", SetCode: "dom", CollectorNumber: "207", Foil: model.FoilTreatmentNonfoil},
		},
		{
			line: "2x Sol Ring (C21) 263 *F*",
			want: bulkCardLine{Quantity: 2, Name: "Sol Ring", SetCode: "c21", CollectorNumber: "263", Foil: model.FoilTreatmentFoil},
		},
		{
			line: "1 Sol Ring (PLST) C21-263 *E*",
			want: bulkCardLine{Quantity: 1, Name: "Sol Ring", SetCode: "plst", CollectorNumber: "C21-263", Foil: model.FoilTreatmentEtched},
		},
		{
			line: "1 Delver of Secrets // Insectile Aberration (ISD) 51",
			want: bulkCardLine{Quantity: 1, Name: "Delver of Secrets // Insectile Aberration", SetCode: "isd", CollectorNumber: "51", Foil: model.FoilTreatmentNonfoil},
		},
		{
			line: "1 Tamiyo's Safekeeping (NEO) 211★",
			want: bulkCardLine{Quantity: 1, Name: "Tamiyo's Safekeeping", SetCode: "neo", CollectorNumber: "211★", Foil: model.FoilTreatmentNonfoil},
		},
		{line: "1 Sol Ring", wantErr: true},
		{line: "1 Sol Ring (C21)", wantErr: true},
		{line: "0 Lightning Bolt (M20) 141", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got, err := parseMoxfieldLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseCategoryHeader(t *testing.T) {
	tests := []struct {
		line   string
		want   model.CardCategory
		wantOK bool
	}{
		{"Sideboard", model.CardCategorySideboard, true},
		{"SIDEBOARD:", model.CardCategorySideboard, true},
		{"commander :", model.CardCategoryCommander, true},
		{"Deck", model.CardCategoryDeck, true},
		{"Maybeboard", "", false},
		{"1 Sol Ring (C21) 263", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got, ok := parseCategoryHeader(tt.line)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("got (%q, %v), want (%q, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestCardNameMatches(t *testing.T) {
	if !cardNameMatches("Delver of Secrets", "Delver of Secrets // Insectile Aberration") {
		t.Error("front face should match double-faced card")
	}
	if !cardNameMatches("lightning bolt", "Lightning Bolt") {
		t.Error("match should be case-insensitive")
	}
	if cardNameMatches("Lightning", "Lightning Bolt") {
		t.Error("partial name should not match")
	}
}
