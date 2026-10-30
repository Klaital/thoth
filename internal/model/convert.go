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
	case CollectionTypeMTGDeck:
		return thothv1.CollectionType_COLLECTION_TYPE_MTG_DECK
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
	case thothv1.CollectionType_COLLECTION_TYPE_MTG_DECK:
		return CollectionTypeMTGDeck
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
		Category:        CardCategory(r.Category),
		Quantity:        r.Quantity,
		Notes:           r.Notes.String,
		CardName:        r.CardName,
		SetCode:         r.SetCode,
		CollectorNumber: r.CollectorNumber,
		ImageURI:        r.ImageUri.String,
		PricesJSON:      string(r.PricesJson),
	}
}

func CardItemToProto(c CardItem) *thothv1.CardItem {
	return &thothv1.CardItem{
		Id:              c.ID,
		CollectionId:    c.CollectionID,
		ScryfallCardId:  c.ScryfallCardID,
		FoilTreatment:   foilTreatmentToProto(c.FoilTreatment),
		Category:        cardCategoryToProto(c.Category),
		Quantity:        c.Quantity,
		Notes:           c.Notes,
		CardName:        c.CardName,
		SetCode:         c.SetCode,
		CollectorNumber: c.CollectorNumber,
		ImageUri:        c.ImageURI,
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

func cardCategoryToProto(c CardCategory) thothv1.CardCategory {
	switch c {
	case CardCategoryCommander:
		return thothv1.CardCategory_CARD_CATEGORY_COMMANDER
	case CardCategoryDeck:
		return thothv1.CardCategory_CARD_CATEGORY_DECK
	case CardCategorySideboard:
		return thothv1.CardCategory_CARD_CATEGORY_SIDEBOARD
	default:
		return thothv1.CardCategory_CARD_CATEGORY_NONE
	}
}

func CardCategoryFromProto(c thothv1.CardCategory) CardCategory {
	switch c {
	case thothv1.CardCategory_CARD_CATEGORY_COMMANDER:
		return CardCategoryCommander
	case thothv1.CardCategory_CARD_CATEGORY_DECK:
		return CardCategoryDeck
	case thothv1.CardCategory_CARD_CATEGORY_SIDEBOARD:
		return CardCategorySideboard
	default:
		return CardCategoryNone
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
