package handler

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/klaital/thoth/internal/auth"
	"github.com/klaital/thoth/internal/db"
	"github.com/klaital/thoth/internal/model"
)

// requireCollectionOwner fetches the collection and returns an error unless the
// authenticated caller is the owner. Returns the collection on success.
func requireCollectionOwner(ctx context.Context, q *db.Queries, collectionID uint64) (db.Collection, error) {
	c, err := q.GetCollectionByID(ctx, collectionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Collection{}, status.Error(codes.NotFound, "collection not found")
		}
		return db.Collection{}, status.Error(codes.Internal, "failed to fetch collection")
	}
	callerID, err := auth.UserIDFromContext(ctx)
	if err != nil || callerID != int64(c.OwnerID) {
		return db.Collection{}, status.Error(codes.PermissionDenied, "not the collection owner")
	}
	return c, nil
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func optNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// ensureDefaultCollection returns the owner's default card collection,
// creating it if it doesn't exist yet.
func ensureDefaultCollection(ctx context.Context, q *db.Queries, ownerID uint64) (db.Collection, error) {
	c, err := q.GetDefaultCollection(ctx, ownerID)
	if err == nil {
		return c, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return db.Collection{}, status.Error(codes.Internal, "failed to fetch default collection")
	}
	// INSERT IGNORE: a concurrent caller may win the unique key; re-read either way.
	if _, err := q.CreateDefaultCollection(ctx, ownerID); err != nil {
		return db.Collection{}, status.Error(codes.Internal, "failed to create default collection")
	}
	c, err = q.GetDefaultCollection(ctx, ownerID)
	if err != nil {
		return db.Collection{}, status.Error(codes.Internal, "failed to fetch default collection")
	}
	return c, nil
}

// languageCode matches the language codes Scryfall uses ("en", "ja", "zhs", ...).
var languageCode = regexp.MustCompile(`^[a-z]{2,3}$`)

// normalizeLanguage lowercases and validates a card language code, mapping
// empty to the default.
func normalizeLanguage(lang string) (string, error) {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return model.DefaultLanguage, nil
	}
	if !languageCode.MatchString(lang) {
		return "", status.Errorf(codes.InvalidArgument, "invalid language code %q", lang)
	}
	return lang, nil
}
