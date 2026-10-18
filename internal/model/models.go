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

type SealedItemType string

const (
	SealedItemTypeSecretLair            SealedItemType = "secret_lair"
	SealedItemTypePlayBooster           SealedItemType = "play_booster"
	SealedItemTypePlayBoosterBox        SealedItemType = "play_booster_box"
	SealedItemTypeCollectorBooster      SealedItemType = "collector_booster"
	SealedItemTypeCollectorBoosterBox   SealedItemType = "collector_booster_box"
	SealedItemTypeBundle                SealedItemType = "bundle"
	SealedItemTypeDraftNight            SealedItemType = "draft_night"
	SealedItemTypeDraftBooster          SealedItemType = "draft_booster"
	SealedItemTypeDraftBoosterBox       SealedItemType = "draft_booster_box"
	SealedItemTypeSetBooster            SealedItemType = "set_booster"
	SealedItemTypeSetBoosterBox         SealedItemType = "set_booster_box"
	SealedItemTypeJumpstartBooster      SealedItemType = "jumpstart_booster"
	SealedItemTypeJumpstartBoosterBox   SealedItemType = "jumpstart_booster_box"
	SealedItemTypeCommanderPrecon       SealedItemType = "commander_precon"
	SealedItemTypePrereleasePack        SealedItemType = "prerelease_pack"
)

type SealedItem struct {
	ID           int64
	CollectionID int64
	ItemType     SealedItemType
	SetCode      *string
	Name         *string
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
