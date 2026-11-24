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
	IsDefault   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type FoilTreatment string

const (
	FoilTreatmentNonfoil FoilTreatment = "nonfoil"
	FoilTreatmentFoil    FoilTreatment = "foil"
	FoilTreatmentEtched  FoilTreatment = "etched"
)

// DefaultLanguage is stored when a card's language isn't given.
const DefaultLanguage = "en"

type CardItem struct {
	ID              int64
	CollectionID    int64
	ScryfallCardID  string
	FoilTreatment   FoilTreatment
	Language        string
	Quantity        int32
	Notes           string
	CardName        string
	SetCode         string
	CollectorNumber string
	ImageURI        string
	ScryfallURI     string
	PricesJSON      string
}

type DeckBoard string

const (
	DeckBoardCommander DeckBoard = "commander"
	DeckBoardMain      DeckBoard = "main"
	DeckBoardSideboard DeckBoard = "sideboard"
)

type Deck struct {
	ID        int64
	OwnerID   int64
	Name      string
	Notes     string
	IsPublic  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DeckCard struct {
	ID                       int64
	DeckID                   int64
	CardName                 string
	Board                    DeckBoard
	Quantity                 int32
	PreferredScryfallCardID  string
	PreferredSetCode         string
	PreferredCollectorNumber string
	ImageURI                 string
	ScryfallURI              string
}

type SealedItemType string

const (
	SealedItemTypeSecretLair          SealedItemType = "secret_lair"
	SealedItemTypePlayBooster         SealedItemType = "play_booster"
	SealedItemTypePlayBoosterBox      SealedItemType = "play_booster_box"
	SealedItemTypeCollectorBooster    SealedItemType = "collector_booster"
	SealedItemTypeCollectorBoosterBox SealedItemType = "collector_booster_box"
	SealedItemTypeBundle              SealedItemType = "bundle"
	SealedItemTypeDraftNight          SealedItemType = "draft_night"
	SealedItemTypeDraftBooster        SealedItemType = "draft_booster"
	SealedItemTypeDraftBoosterBox     SealedItemType = "draft_booster_box"
	SealedItemTypeSetBooster          SealedItemType = "set_booster"
	SealedItemTypeSetBoosterBox       SealedItemType = "set_booster_box"
	SealedItemTypeJumpstartBooster    SealedItemType = "jumpstart_booster"
	SealedItemTypeJumpstartBoosterBox SealedItemType = "jumpstart_booster_box"
	SealedItemTypeCommanderPrecon     SealedItemType = "commander_precon"
	SealedItemTypePrereleasePack      SealedItemType = "prerelease_pack"
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
