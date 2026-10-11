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

func NullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}
