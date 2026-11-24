package model

import (
	"encoding/json"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/klaital/thoth/internal/db"
	thothv1 "github.com/klaital/thoth/internal/gen/thoth/v1"
)

// OwnedCopy is one collection card row that can satisfy a deck card.
type OwnedCopy struct {
	CollectionCardID    int64
	CollectionID        int64
	CollectionName      string
	ScryfallCardID      string
	CardName            string
	SetCode             string
	CollectorNumber     string
	ImageURI            string
	FoilTreatment       FoilTreatment
	Language            string
	Quantity            int32
	CheckedOutHere      int32
	CheckedOutElsewhere int32
	// UnitPriceUSD is empty when Scryfall has no price for this foil treatment.
	UnitPriceUSD      string
	SuggestedQuantity int32
}

// Free is the number of copies still in the collection.
func (c OwnedCopy) Free() int32 {
	return max(0, c.Quantity-c.CheckedOutHere-c.CheckedOutElsewhere)
}

func OwnedCopyFromDB(r db.ListOwnedCopiesForDeckRow) OwnedCopy {
	foil := FoilTreatment(r.FoilTreatment)
	return OwnedCopy{
		CollectionCardID:    int64(r.ID),
		CollectionID:        int64(r.CollectionID),
		CollectionName:      r.CollectionName,
		ScryfallCardID:      r.ScryfallCardID,
		CardName:            r.CardName,
		SetCode:             r.SetCode,
		CollectorNumber:     r.CollectorNumber,
		ImageURI:            r.ImageUri.String,
		FoilTreatment:       foil,
		Language:            r.Language,
		Quantity:            r.Quantity,
		CheckedOutHere:      int32(r.CheckedOutHere),
		CheckedOutElsewhere: int32(r.CheckedOutTotal - r.CheckedOutHere),
		UnitPriceUSD:        UnitPriceUSD(string(r.PricesJson), foil),
	}
}

func OwnedCopyToProto(c OwnedCopy) *thothv1.OwnedCopy {
	out := &thothv1.OwnedCopy{
		CollectionCardId:    c.CollectionCardID,
		CollectionId:        c.CollectionID,
		CollectionName:      c.CollectionName,
		ScryfallCardId:      c.ScryfallCardID,
		SetCode:             c.SetCode,
		CollectorNumber:     c.CollectorNumber,
		ImageUri:            c.ImageURI,
		FoilTreatment:       foilTreatmentToProto(c.FoilTreatment),
		Language:            c.Language,
		Quantity:            c.Quantity,
		CheckedOutHere:      c.CheckedOutHere,
		CheckedOutElsewhere: c.CheckedOutElsewhere,
		Free:                c.Free(),
		SuggestedQuantity:   c.SuggestedQuantity,
	}
	if c.UnitPriceUSD != "" {
		p := c.UnitPriceUSD
		out.UnitPriceUsd = &p
	}
	return out
}

// UnitPriceUSD picks the Scryfall USD price for a foil treatment out of the
// stored price map, or "" if there is none.
func UnitPriceUSD(pricesJSON string, foil FoilTreatment) string {
	var prices map[string]*string
	if err := json.Unmarshal([]byte(pricesJSON), &prices); err != nil {
		return ""
	}
	key := "usd"
	switch foil {
	case FoilTreatmentFoil:
		key = "usd_foil"
	case FoilTreatmentEtched:
		key = "usd_etched"
	}
	if p := prices[key]; p != nil {
		return *p
	}
	return ""
}

// DeckCheckout is copies of one collection card row checked out to a deck.
type DeckCheckout struct {
	CollectionCardID int64
	CollectionID     int64
	CollectionName   string
	CardName         string
	ScryfallCardID   string
	SetCode          string
	CollectorNumber  string
	ImageURI         string
	FoilTreatment    FoilTreatment
	Language         string
	Quantity         int32
	CheckedOutAt     time.Time
}

func DeckCheckoutFromDB(r db.ListCheckoutsByDeckRow) DeckCheckout {
	return DeckCheckout{
		CollectionCardID: int64(r.CollectionCardID),
		CollectionID:     int64(r.CollectionID),
		CollectionName:   r.CollectionName,
		CardName:         r.CardName,
		ScryfallCardID:   r.ScryfallCardID,
		SetCode:          r.SetCode,
		CollectorNumber:  r.CollectorNumber,
		ImageURI:         r.ImageUri.String,
		FoilTreatment:    FoilTreatment(r.FoilTreatment),
		Language:         r.Language,
		Quantity:         r.Quantity,
		CheckedOutAt:     r.CheckedOutAt,
	}
}

func DeckCheckoutToProto(c DeckCheckout) *thothv1.DeckCheckout {
	return &thothv1.DeckCheckout{
		CollectionCardId: c.CollectionCardID,
		CollectionId:     c.CollectionID,
		CollectionName:   c.CollectionName,
		CardName:         c.CardName,
		ScryfallCardId:   c.ScryfallCardID,
		SetCode:          c.SetCode,
		CollectorNumber:  c.CollectorNumber,
		ImageUri:         c.ImageURI,
		FoilTreatment:    foilTreatmentToProto(c.FoilTreatment),
		Language:         c.Language,
		Quantity:         c.Quantity,
		CheckedOutAt:     timestamppb.New(c.CheckedOutAt),
	}
}

func CardCheckoutToProto(r db.ListCheckoutsByCollectionCardIDsRow) *thothv1.CardCheckout {
	return &thothv1.CardCheckout{
		DeckId:   int64(r.DeckID),
		DeckName: r.DeckName,
		Quantity: r.Quantity,
	}
}
