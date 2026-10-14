package model

import "time"

type User struct {
	ID           int64
	Email        string
	PasswordHash *string
	GoogleID     *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CollectionType string

const (
	CollectionTypeMTGCards  CollectionType = "mtg_cards"
	CollectionTypeMTGSealed CollectionType = "mtg_sealed"
	CollectionTypeMTGDeck   CollectionType = "mtg_deck"
	CollectionTypeBooks     CollectionType = "books"
	CollectionTypeDVDs      CollectionType = "dvds"
)

type Collection struct {
	ID          int64
	OwnerID     int64
	Name        string
	Description string
	Type        CollectionType
	IsPublic    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type FoilTreatment string

const (
	FoilTreatmentNonfoil FoilTreatment = "nonfoil"
	FoilTreatmentFoil    FoilTreatment = "foil"
	FoilTreatmentEtched  FoilTreatment = "etched"
)

type CardItem struct {
	ID              int64
	CollectionID    int64
	ScryfallCardID  string
	FoilTreatment   FoilTreatment
	Quantity        int32
	Notes           string
	CardName        string
	SetCode         string
	CollectorNumber string
	ImageURI        string
	PricesJSON      string
}

type SealedItem struct {
	ID           int64
	CollectionID int64
	SetCode      string
	Description  string
	Quantity     int32
	Notes        string
}

type BookDvdItem struct {
	ID           int64
	CollectionID int64
	UPC          string
	Title        string
	Quantity     int32
	Notes        string
}

type ScryfallCard struct {
	ID              string
	Name            string
	SetCode         string
	CollectorNumber string
	ImageURI        string
	PricesJSON      string
}
