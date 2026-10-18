package handler

import (
	"context"
	"database/sql"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/klaital/thoth/internal/db"
	thothv1 "github.com/klaital/thoth/internal/gen/thoth/v1"
	"github.com/klaital/thoth/internal/model"
)

type SealedItemHandler struct {
	thothv1.UnimplementedSealedItemServiceServer
	queries *db.Queries
}

func NewSealedItemHandler(queries *db.Queries) *SealedItemHandler {
	return &SealedItemHandler{queries: queries}
}

func (h *SealedItemHandler) AddSealedItem(ctx context.Context, req *thothv1.AddSealedItemRequest) (*thothv1.AddSealedItemResponse, error) {
	if _, err := requireCollectionOwner(ctx, h.queries, uint64(req.CollectionId)); err != nil {
		return nil, err
	}
	if req.ItemType == thothv1.SealedItemType_SEALED_ITEM_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "item_type is required")
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	result, err := h.queries.CreateSealedItem(ctx, db.CreateSealedItemParams{
		CollectionID: uint64(req.CollectionId),
		ItemType:     db.SealedItemsItemType(model.SealedItemTypeFromProto(req.ItemType)),
		SetCode:      optNullString(req.SetCode),
		Name:         optNullString(req.Name),
		Quantity:     req.Quantity,
		Notes:        nullString(req.Notes),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to add sealed item")
	}
	newID, _ := result.LastInsertId()

	row, err := h.queries.GetSealedItemByID(ctx, uint64(newID))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch new sealed item")
	}
	return &thothv1.AddSealedItemResponse{Item: model.SealedItemToProto(model.SealedItemFromDB(row))}, nil
}

func (h *SealedItemHandler) ListSealedItems(ctx context.Context, req *thothv1.ListSealedItemsRequest) (*thothv1.ListSealedItemsResponse, error) {
	rows, err := h.queries.ListSealedItemsByCollection(ctx, uint64(req.CollectionId))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list sealed items")
	}
	out := make([]*thothv1.SealedItem, len(rows))
	for i, r := range rows {
		out[i] = model.SealedItemToProto(model.SealedItemFromDB(r))
	}
	return &thothv1.ListSealedItemsResponse{Items: out}, nil
}

func (h *SealedItemHandler) UpdateSealedItem(ctx context.Context, req *thothv1.UpdateSealedItemRequest) (*thothv1.UpdateSealedItemResponse, error) {
	row, err := h.queries.GetSealedItemByID(ctx, uint64(req.Id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "sealed item not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch sealed item")
	}
	if _, err := requireCollectionOwner(ctx, h.queries, row.CollectionID); err != nil {
		return nil, err
	}

	if err := h.queries.UpdateSealedItem(ctx, db.UpdateSealedItemParams{
		ItemType: db.SealedItemsItemType(model.SealedItemTypeFromProto(req.ItemType)),
		SetCode:  optNullString(req.SetCode),
		Name:     optNullString(req.Name),
		Quantity: req.Quantity,
		Notes:    nullString(req.Notes),
		ID:       uint64(req.Id),
	}); err != nil {
		return nil, status.Error(codes.Internal, "failed to update sealed item")
	}

	updated, err := h.queries.GetSealedItemByID(ctx, uint64(req.Id))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch updated sealed item")
	}
	return &thothv1.UpdateSealedItemResponse{Item: model.SealedItemToProto(model.SealedItemFromDB(updated))}, nil
}

func (h *SealedItemHandler) RemoveSealedItem(ctx context.Context, req *thothv1.RemoveSealedItemRequest) (*thothv1.RemoveSealedItemResponse, error) {
	row, err := h.queries.GetSealedItemByID(ctx, uint64(req.Id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "sealed item not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch sealed item")
	}
	if _, err := requireCollectionOwner(ctx, h.queries, row.CollectionID); err != nil {
		return nil, err
	}
	if err := h.queries.DeleteSealedItem(ctx, uint64(req.Id)); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete sealed item")
	}
	return &thothv1.RemoveSealedItemResponse{}, nil
}
