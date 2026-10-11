package handler

import (
	"context"
	"database/sql"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/klaital/thoth/internal/auth"
	"github.com/klaital/thoth/internal/db"
	thothv1 "github.com/klaital/thoth/internal/gen/thoth/v1"
	"github.com/klaital/thoth/internal/model"
)

type CollectionHandler struct {
	thothv1.UnimplementedCollectionServiceServer
	queries *db.Queries
}

func NewCollectionHandler(queries *db.Queries) *CollectionHandler {
	return &CollectionHandler{queries: queries}
}

func (h *CollectionHandler) CreateCollection(ctx context.Context, req *thothv1.CreateCollectionRequest) (*thothv1.CreateCollectionResponse, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.Type == thothv1.CollectionType_COLLECTION_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "type is required")
	}

	result, err := h.queries.CreateCollection(ctx, db.CreateCollectionParams{
		OwnerID:     uint64(userID),
		Name:        req.Name,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		Type:        db.CollectionsType(model.CollectionTypeFromProto(req.Type)),
		IsPublic:    req.IsPublic,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create collection")
	}

	newID, err := result.LastInsertId()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve new collection id")
	}

	c, err := h.queries.GetCollectionByID(ctx, uint64(newID))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch new collection")
	}

	return &thothv1.CreateCollectionResponse{
		Collection: model.CollectionToProto(model.CollectionFromDB(c)),
	}, nil
}

func (h *CollectionHandler) GetCollection(ctx context.Context, req *thothv1.GetCollectionRequest) (*thothv1.GetCollectionResponse, error) {
	c, err := h.queries.GetCollectionByID(ctx, uint64(req.Id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "collection not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch collection")
	}

	if err := h.authorizeRead(ctx, c); err != nil {
		return nil, err
	}

	return &thothv1.GetCollectionResponse{
		Collection: model.CollectionToProto(model.CollectionFromDB(c)),
	}, nil
}

func (h *CollectionHandler) ListCollections(ctx context.Context, req *thothv1.ListCollectionsRequest) (*thothv1.ListCollectionsResponse, error) {
	callerID, callerErr := auth.UserIDFromContext(ctx)

	// Determine whose collections to list.
	var targetOwnerID int64
	if req.OwnerId != nil {
		targetOwnerID = *req.OwnerId
	} else {
		if callerErr != nil {
			return nil, status.Error(codes.Unauthenticated, "not authenticated")
		}
		targetOwnerID = callerID
	}

	var rows []db.Collection
	var dbErr error

	// Show all collections to the owner; only public ones to everyone else.
	if callerErr == nil && callerID == targetOwnerID {
		rows, dbErr = h.queries.ListCollectionsByOwner(ctx, uint64(targetOwnerID))
	} else {
		rows, dbErr = h.queries.ListPublicCollectionsByOwner(ctx, uint64(targetOwnerID))
	}
	if dbErr != nil {
		return nil, status.Error(codes.Internal, "failed to list collections")
	}

	out := make([]*thothv1.Collection, len(rows))
	for i, row := range rows {
		out[i] = model.CollectionToProto(model.CollectionFromDB(row))
	}
	return &thothv1.ListCollectionsResponse{Collections: out}, nil
}

func (h *CollectionHandler) UpdateCollection(ctx context.Context, req *thothv1.UpdateCollectionRequest) (*thothv1.UpdateCollectionResponse, error) {
	c, err := h.queries.GetCollectionByID(ctx, uint64(req.Id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "collection not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch collection")
	}
	if err := h.authorizeWrite(ctx, c); err != nil {
		return nil, err
	}

	if err := h.queries.UpdateCollection(ctx, db.UpdateCollectionParams{
		Name:        req.Name,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		IsPublic:    req.IsPublic,
		ID:          uint64(req.Id),
	}); err != nil {
		return nil, status.Error(codes.Internal, "failed to update collection")
	}

	updated, err := h.queries.GetCollectionByID(ctx, uint64(req.Id))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch updated collection")
	}
	return &thothv1.UpdateCollectionResponse{
		Collection: model.CollectionToProto(model.CollectionFromDB(updated)),
	}, nil
}

func (h *CollectionHandler) DeleteCollection(ctx context.Context, req *thothv1.DeleteCollectionRequest) (*thothv1.DeleteCollectionResponse, error) {
	c, err := h.queries.GetCollectionByID(ctx, uint64(req.Id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "collection not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch collection")
	}
	if err := h.authorizeWrite(ctx, c); err != nil {
		return nil, err
	}

	if err := h.queries.DeleteCollection(ctx, uint64(req.Id)); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete collection")
	}
	return &thothv1.DeleteCollectionResponse{}, nil
}

// authorizeRead returns an error if the caller may not view the collection.
func (h *CollectionHandler) authorizeRead(ctx context.Context, c db.Collection) error {
	if c.IsPublic {
		return nil
	}
	callerID, err := auth.UserIDFromContext(ctx)
	if err != nil || callerID != int64(c.OwnerID) {
		return status.Error(codes.NotFound, "collection not found")
	}
	return nil
}

// authorizeWrite returns an error if the caller is not the owner.
func (h *CollectionHandler) authorizeWrite(ctx context.Context, c db.Collection) error {
	callerID, err := auth.UserIDFromContext(ctx)
	if err != nil || callerID != int64(c.OwnerID) {
		return status.Error(codes.PermissionDenied, "not the owner")
	}
	return nil
}
