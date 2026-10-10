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

func NullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}
