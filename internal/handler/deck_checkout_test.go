package handler

import (
	"reflect"
	"testing"

	"github.com/klaital/thoth/internal/model"
)

func TestSuggestCopies(t *testing.T) {
	cp := func(id, coll int64, printing string, foil model.FoilTreatment, lang, price string, qty, out int32) model.OwnedCopy {
		return model.OwnedCopy{
			CollectionCardID: id, CollectionID: coll, ScryfallCardID: printing, FoilTreatment: foil,
			Language: lang, UnitPriceUSD: price, Quantity: qty, CheckedOutElsewhere: out,
		}
	}
	const (
		nf = model.FoilTreatmentNonfoil
		f  = model.FoilTreatmentFoil
		et = model.FoilTreatmentEtched
	)
	tests := []struct {
		name      string
		need      int32
		preferred string
		copies    []model.OwnedCopy
		want      []int32 // suggested quantity per copy, in input order
	}{
		{
			name:   "nonfoil before foil before etched",
			need:   1,
			copies: []model.OwnedCopy{cp(1, 1, "a", et, "en", "1", 1, 0), cp(2, 1, "a", f, "en", "1", 1, 0), cp(3, 1, "a", nf, "en", "9", 1, 0)},
			want:   []int32{0, 0, 1},
		},
		{
			name:   "cheapest first, unknown price last",
			need:   1,
			copies: []model.OwnedCopy{cp(1, 1, "a", nf, "en", "", 1, 0), cp(2, 1, "b", nf, "en", "3.50", 1, 0), cp(3, 1, "c", nf, "en", "0.25", 1, 0)},
			want:   []int32{0, 0, 1},
		},
		{
			name:      "preferred printing beats price and foil",
			need:      1,
			preferred: "fancy",
			copies:    []model.OwnedCopy{cp(1, 1, "cheap", nf, "en", "0.10", 1, 0), cp(2, 1, "fancy", f, "en", "40", 1, 0)},
			want:      []int32{0, 1},
		},
		{
			name:   "english before other languages",
			need:   1,
			copies: []model.OwnedCopy{cp(1, 1, "a", nf, "ja", "1", 1, 0), cp(2, 1, "a", nf, "en", "1", 1, 0)},
			want:   []int32{0, 1},
		},
		{
			name:   "splits across rows and skips copies checked out elsewhere",
			need:   3,
			copies: []model.OwnedCopy{cp(1, 1, "a", nf, "en", "1", 2, 1), cp(2, 2, "a", nf, "en", "2", 4, 0)},
			want:   []int32{1, 2},
		},
		{
			name:   "ties go to the box with the most copies",
			need:   2,
			copies: []model.OwnedCopy{cp(1, 1, "a", nf, "en", "1", 1, 0), cp(2, 2, "a", nf, "en", "1", 3, 0)},
			want:   []int32{0, 2},
		},
		{
			name:   "nothing needed",
			need:   0,
			copies: []model.OwnedCopy{cp(1, 1, "a", nf, "en", "1", 1, 0)},
			want:   []int32{0},
		},
		{
			name:   "not enough copies",
			need:   5,
			copies: []model.OwnedCopy{cp(1, 1, "a", nf, "en", "1", 2, 0)},
			want:   []int32{2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestCopies(tt.need, tt.copies, tt.preferred)
			got := make([]int32, len(tt.copies))
			for i, c := range tt.copies {
				got[i] = c.SuggestedQuantity
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("suggested = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnitPriceUSD(t *testing.T) {
	prices := `{"usd":"1.50","usd_foil":"4.00","usd_etched":null,"eur":"1.20"}`
	for foil, want := range map[model.FoilTreatment]string{
		model.FoilTreatmentNonfoil: "1.50",
		model.FoilTreatmentFoil:    "4.00",
		model.FoilTreatmentEtched:  "",
	} {
		if got := model.UnitPriceUSD(prices, foil); got != want {
			t.Errorf("%s price = %q, want %q", foil, got, want)
		}
	}
	if got := model.UnitPriceUSD("", model.FoilTreatmentNonfoil); got != "" {
		t.Errorf("empty prices = %q", got)
	}
}
