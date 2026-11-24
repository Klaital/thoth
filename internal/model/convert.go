package model

import (
	"database/sql"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/klaital/thoth/internal/db"
	thothv1 "github.com/klaital/thoth/internal/gen/thoth/v1"
)

func UserFromDB(u db.User) User {
	var passwordHash *string
	if u.PasswordHash.Valid {
		passwordHash = &u.PasswordHash.String
	}
	var googleID *string
	if u.GoogleID.Valid {
		googleID = &u.GoogleID.String
	}
	return User{
		ID:           int64(u.ID),
		Email:        u.Email,
		PasswordHash: passwordHash,
		GoogleID:     googleID,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func UserToProto(u User) *thothv1.User {
	return &thothv1.User{
		Id:        u.ID,
		Email:     u.Email,
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
}

func CollectionFromDB(c db.Collection) Collection {
	return Collection{
		ID:          int64(c.ID),
		OwnerID:     int64(c.OwnerID),
		Name:        c.Name,
		Description: c.Description.String,
		Type:        CollectionType(c.Type),
		IsPublic:    c.IsPublic,
		IsDefault:   c.IsDefault,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func CollectionToProto(c Collection) *thothv1.Collection {
	return &thothv1.Collection{
		Id:          c.ID,
		OwnerId:     c.OwnerID,
		Name:        c.Name,
		Description: c.Description,
		Type:        collectionTypeToProto(c.Type),
		IsPublic:    c.IsPublic,
		IsDefault:   c.IsDefault,
		CreatedAt:   timestamppb.New(c.CreatedAt),
		UpdatedAt:   timestamppb.New(c.UpdatedAt),
	}
}

func collectionTypeToProto(t CollectionType) thothv1.CollectionType {
	switch t {
	case CollectionTypeMTGCards:
		return thothv1.CollectionType_COLLECTION_TYPE_MTG_CARDS
	case CollectionTypeMTGSealed:
		return thothv1.CollectionType_COLLECTION_TYPE_MTG_SEALED
	case CollectionTypeBooks:
		return thothv1.CollectionType_COLLECTION_TYPE_BOOKS
	case CollectionTypeDVDs:
		return thothv1.CollectionType_COLLECTION_TYPE_DVDS
	default:
		return thothv1.CollectionType_COLLECTION_TYPE_UNSPECIFIED
	}
}

func CollectionTypeFromProto(t thothv1.CollectionType) CollectionType {
	switch t {
	case thothv1.CollectionType_COLLECTION_TYPE_MTG_CARDS:
		return CollectionTypeMTGCards
	case thothv1.CollectionType_COLLECTION_TYPE_MTG_SEALED:
		return CollectionTypeMTGSealed
	case thothv1.CollectionType_COLLECTION_TYPE_BOOKS:
		return CollectionTypeBooks
	case thothv1.CollectionType_COLLECTION_TYPE_DVDS:
		return CollectionTypeDVDs
	default:
		return CollectionTypeMTGCards
	}
}

func CardItemFromDB(r db.ListCardItemsByCollectionRow) CardItem {
	return CardItem{
		ID:              int64(r.ID),
		CollectionID:    int64(r.CollectionID),
		ScryfallCardID:  r.ScryfallCardID,
		FoilTreatment:   FoilTreatment(r.FoilTreatment),
		Language:        r.Language,
		Quantity:        r.Quantity,
		Notes:           r.Notes.String,
		CardName:        r.CardName,
		SetCode:         r.SetCode,
		CollectorNumber: r.CollectorNumber,
		ImageURI:        r.ImageUri.String,
		ScryfallURI:     r.ScryfallUri.String,
		PricesJSON:      string(r.PricesJson),
	}
}

func CardItemToProto(c CardItem) *thothv1.CardItem {
	return &thothv1.CardItem{
		Id:              c.ID,
		CollectionId:    c.CollectionID,
		ScryfallCardId:  c.ScryfallCardID,
		FoilTreatment:   foilTreatmentToProto(c.FoilTreatment),
		Language:        c.Language,
		Quantity:        c.Quantity,
		Notes:           c.Notes,
		CardName:        c.CardName,
		SetCode:         c.SetCode,
		CollectorNumber: c.CollectorNumber,
		ImageUri:        c.ImageURI,
		ScryfallUri:     c.ScryfallURI,
		PricesJson:      c.PricesJSON,
	}
}

func foilTreatmentToProto(f FoilTreatment) thothv1.FoilTreatment {
	switch f {
	case FoilTreatmentFoil:
		return thothv1.FoilTreatment_FOIL_TREATMENT_FOIL
	case FoilTreatmentEtched:
		return thothv1.FoilTreatment_FOIL_TREATMENT_ETCHED
	default:
		return thothv1.FoilTreatment_FOIL_TREATMENT_NONFOIL
	}
}

func FoilTreatmentFromProto(f thothv1.FoilTreatment) FoilTreatment {
	switch f {
	case thothv1.FoilTreatment_FOIL_TREATMENT_FOIL:
		return FoilTreatmentFoil
	case thothv1.FoilTreatment_FOIL_TREATMENT_ETCHED:
		return FoilTreatmentEtched
	default:
		return FoilTreatmentNonfoil
	}
}

func DeckFromDB(d db.Deck) Deck {
	return Deck{
		ID:        int64(d.ID),
		OwnerID:   int64(d.OwnerID),
		Name:      d.Name,
		Notes:     d.Notes.String,
		IsPublic:  d.IsPublic,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

func DeckToProto(d Deck) *thothv1.Deck {
	return &thothv1.Deck{
		Id:        d.ID,
		OwnerId:   d.OwnerID,
		Name:      d.Name,
		Notes:     d.Notes,
		IsPublic:  d.IsPublic,
		CreatedAt: timestamppb.New(d.CreatedAt),
		UpdatedAt: timestamppb.New(d.UpdatedAt),
	}
}

func DeckCardFromDB(r db.ListDeckCardsRow) DeckCard {
	return DeckCard{
		ID:                       int64(r.ID),
		DeckID:                   int64(r.DeckID),
		CardName:                 r.CardName,
		Board:                    DeckBoard(r.Board),
		Quantity:                 r.Quantity,
		PreferredScryfallCardID:  r.PreferredScryfallCardID.String,
		PreferredSetCode:         r.PreferredSetCode.String,
		PreferredCollectorNumber: r.PreferredCollectorNumber.String,
		ImageURI:                 r.ImageUri.String,
		ScryfallURI:              r.ScryfallUri.String,
	}
}

func DeckCardToProto(c DeckCard) *thothv1.DeckCard {
	return &thothv1.DeckCard{
		Id:                       c.ID,
		DeckId:                   c.DeckID,
		CardName:                 c.CardName,
		Board:                    DeckBoardToProto(c.Board),
		Quantity:                 c.Quantity,
		PreferredScryfallCardId:  c.PreferredScryfallCardID,
		PreferredSetCode:         c.PreferredSetCode,
		PreferredCollectorNumber: c.PreferredCollectorNumber,
		ImageUri:                 c.ImageURI,
		ScryfallUri:              c.ScryfallURI,
	}
}

func DeckBoardToProto(b DeckBoard) thothv1.DeckBoard {
	switch b {
	case DeckBoardCommander:
		return thothv1.DeckBoard_DECK_BOARD_COMMANDER
	case DeckBoardSideboard:
		return thothv1.DeckBoard_DECK_BOARD_SIDEBOARD
	default:
		return thothv1.DeckBoard_DECK_BOARD_MAIN
	}
}

// DeckBoardFromProto maps UNSPECIFIED to MAIN.
func DeckBoardFromProto(b thothv1.DeckBoard) DeckBoard {
	switch b {
	case thothv1.DeckBoard_DECK_BOARD_COMMANDER:
		return DeckBoardCommander
	case thothv1.DeckBoard_DECK_BOARD_SIDEBOARD:
		return DeckBoardSideboard
	default:
		return DeckBoardMain
	}
}

func SealedItemFromDB(r db.SealedItem) SealedItem {
	var setCode *string
	if r.SetCode.Valid {
		setCode = &r.SetCode.String
	}
	var name *string
	if r.Name.Valid {
		name = &r.Name.String
	}
	return SealedItem{
		ID:           int64(r.ID),
		CollectionID: int64(r.CollectionID),
		ItemType:     SealedItemType(r.ItemType),
		SetCode:      setCode,
		Name:         name,
		Quantity:     r.Quantity,
		Notes:        r.Notes.String,
	}
}

func SealedItemToProto(s SealedItem) *thothv1.SealedItem {
	return &thothv1.SealedItem{
		Id:           s.ID,
		CollectionId: s.CollectionID,
		ItemType:     sealedItemTypeToProto(s.ItemType),
		SetCode:      s.SetCode,
		Name:         s.Name,
		Quantity:     s.Quantity,
		Notes:        s.Notes,
	}
}

func sealedItemTypeToProto(t SealedItemType) thothv1.SealedItemType {
	switch t {
	case SealedItemTypeSecretLair:
		return thothv1.SealedItemType_SEALED_ITEM_TYPE_SECRET_LAIR
	case SealedItemTypePlayBooster:
		return thothv1.SealedItemType_SEALED_ITEM_TYPE_PLAY_BOOSTER
	case SealedItemTypePlayBoosterBox:
		return thothv1.SealedItemType_SEALED_ITEM_TYPE_PLAY_BOOSTER_BOX
	case SealedItemTypeCollectorBooster:
		return thothv1.SealedItemType_SEALED_ITEM_TYPE_COLLECTOR_BOOSTER
	case SealedItemTypeCollectorBoosterBox:
		return thothv1.SealedItemType_SEALED_ITEM_TYPE_COLLECTOR_BOOSTER_BOX
	case SealedItemTypeBundle:
		return thothv1.SealedItemType_SEALED_ITEM_TYPE_BUNDLE
	case SealedItemTypeDraftNight:
		return thothv1.SealedItemType_SEALED_ITEM_TYPE_DRAFT_NIGHT
	case SealedItemTypeDraftBooster:
		return thothv1.SealedItemType_SEALED_ITEM_TYPE_DRAFT_BOOSTER
	case SealedItemTypeDraftBoosterBox:
		return thothv1.SealedItemType_SEALED_ITEM_TYPE_DRAFT_BOOSTER_BOX
	case SealedItemTypeSetBooster:
		return thothv1.SealedItemType_SEALED_ITEM_TYPE_SET_BOOSTER
	case SealedItemTypeSetBoosterBox:
		return thothv1.SealedItemType_SEALED_ITEM_TYPE_SET_BOOSTER_BOX
	case SealedItemTypeJumpstartBooster:
		return thothv1.SealedItemType_SEALED_ITEM_TYPE_JUMPSTART_BOOSTER
	case SealedItemTypeJumpstartBoosterBox:
		return thothv1.SealedItemType_SEALED_ITEM_TYPE_JUMPSTART_BOOSTER_BOX
	case SealedItemTypeCommanderPrecon:
		return thothv1.SealedItemType_SEALED_ITEM_TYPE_COMMANDER_PRECON
	case SealedItemTypePrereleasePack:
		return thothv1.SealedItemType_SEALED_ITEM_TYPE_PRERELEASE_PACK
	default:
		return thothv1.SealedItemType_SEALED_ITEM_TYPE_UNSPECIFIED
	}
}

func SealedItemTypeToProto(t SealedItemType) thothv1.SealedItemType {
	return sealedItemTypeToProto(t)
}

func SealedItemTypeFromProto(t thothv1.SealedItemType) SealedItemType {
	switch t {
	case thothv1.SealedItemType_SEALED_ITEM_TYPE_SECRET_LAIR:
		return SealedItemTypeSecretLair
	case thothv1.SealedItemType_SEALED_ITEM_TYPE_PLAY_BOOSTER:
		return SealedItemTypePlayBooster
	case thothv1.SealedItemType_SEALED_ITEM_TYPE_PLAY_BOOSTER_BOX:
		return SealedItemTypePlayBoosterBox
	case thothv1.SealedItemType_SEALED_ITEM_TYPE_COLLECTOR_BOOSTER:
		return SealedItemTypeCollectorBooster
	case thothv1.SealedItemType_SEALED_ITEM_TYPE_COLLECTOR_BOOSTER_BOX:
		return SealedItemTypeCollectorBoosterBox
	case thothv1.SealedItemType_SEALED_ITEM_TYPE_BUNDLE:
		return SealedItemTypeBundle
	case thothv1.SealedItemType_SEALED_ITEM_TYPE_DRAFT_NIGHT:
		return SealedItemTypeDraftNight
	case thothv1.SealedItemType_SEALED_ITEM_TYPE_DRAFT_BOOSTER:
		return SealedItemTypeDraftBooster
	case thothv1.SealedItemType_SEALED_ITEM_TYPE_DRAFT_BOOSTER_BOX:
		return SealedItemTypeDraftBoosterBox
	case thothv1.SealedItemType_SEALED_ITEM_TYPE_SET_BOOSTER:
		return SealedItemTypeSetBooster
	case thothv1.SealedItemType_SEALED_ITEM_TYPE_SET_BOOSTER_BOX:
		return SealedItemTypeSetBoosterBox
	case thothv1.SealedItemType_SEALED_ITEM_TYPE_JUMPSTART_BOOSTER:
		return SealedItemTypeJumpstartBooster
	case thothv1.SealedItemType_SEALED_ITEM_TYPE_JUMPSTART_BOOSTER_BOX:
		return SealedItemTypeJumpstartBoosterBox
	case thothv1.SealedItemType_SEALED_ITEM_TYPE_COMMANDER_PRECON:
		return SealedItemTypeCommanderPrecon
	case thothv1.SealedItemType_SEALED_ITEM_TYPE_PRERELEASE_PACK:
		return SealedItemTypePrereleasePack
	default:
		return SealedItemTypePlayBoosterBox
	}
}

func BookDvdItemFromDB(r db.BookDvdItem) BookDvdItem {
	return BookDvdItem{
		ID:           int64(r.ID),
		CollectionID: int64(r.CollectionID),
		UPC:          r.Upc,
		Title:        r.Title,
		Quantity:     r.Quantity,
		Notes:        r.Notes.String,
	}
}

func BookDvdItemToProto(b BookDvdItem) *thothv1.BookDvdItem {
	return &thothv1.BookDvdItem{
		Id:           b.ID,
		CollectionId: b.CollectionID,
		Upc:          b.UPC,
		Title:        b.Title,
		Quantity:     b.Quantity,
		Notes:        b.Notes,
	}
}

func ScryfallCardToProto(s db.ScryfallCard) *thothv1.ScryfallCard {
	return &thothv1.ScryfallCard{
		Id:              s.ID,
		Name:            s.Name,
		SetCode:         s.SetCode,
		CollectorNumber: s.CollectorNumber,
		ImageUri:        s.ImageUri.String,
		PricesJson:      string(s.PricesJson),
		Foil:            s.Foil,
		Nonfoil:         s.Nonfoil,
	}
}

func NullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}
