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
		Quantity:        r.Quantity,
		Notes:           r.Notes.String,
		CardName:        r.CardName,
		SetCode:         r.SetCode,
		CollectorNumber: r.CollectorNumber,
		ImageURI:        r.ImageUri.String,
	}
}

func CardItemToProto(c CardItem) *thothv1.CardItem {
	return &thothv1.CardItem{
		Id:              c.ID,
		CollectionId:    c.CollectionID,
		ScryfallCardId:  c.ScryfallCardID,
		FoilTreatment:   foilTreatmentToProto(c.FoilTreatment),
		Quantity:        c.Quantity,
		Notes:           c.Notes,
		CardName:        c.CardName,
		SetCode:         c.SetCode,
		CollectorNumber: c.CollectorNumber,
		ImageUri:        c.ImageURI,
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

func SealedItemFromDB(r db.SealedItem) SealedItem {
	return SealedItem{
		ID:           int64(r.ID),
		CollectionID: int64(r.CollectionID),
		SetCode:      r.SetCode,
		Description:  r.Description.String,
		Quantity:     r.Quantity,
		Notes:        r.Notes.String,
	}
}

func SealedItemToProto(s SealedItem) *thothv1.SealedItem {
	return &thothv1.SealedItem{
		Id:           s.ID,
		CollectionId: s.CollectionID,
		SetCode:      s.SetCode,
		Description:  s.Description,
		Quantity:     s.Quantity,
		Notes:        s.Notes,
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
	}
}

func NullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}
