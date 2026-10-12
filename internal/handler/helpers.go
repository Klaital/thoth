package handler

import (
	"context"
	"database/sql"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/klaital/thoth/internal/auth"
	"github.com/klaital/thoth/internal/db"
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
