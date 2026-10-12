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

type BookDvdItemHandler struct {
	thothv1.UnimplementedBookDvdItemServiceServer
	queries *db.Queries
}

func NewBookDvdItemHandler(queries *db.Queries) *BookDvdItemHandler {
	return &BookDvdItemHandler{queries: queries}
}

func (h *BookDvdItemHandler) AddBookDvdItem(ctx context.Context, req *thothv1.AddBookDvdItemRequest) (*thothv1.AddBookDvdItemResponse, error) {
	if _, err := requireCollectionOwner(ctx, h.queries, uint64(req.CollectionId)); err != nil {
		return nil, err
	}
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	result, err := h.queries.CreateBookDvdItem(ctx, db.CreateBookDvdItemParams{
		CollectionID: uint64(req.CollectionId),
		Upc:          req.Upc,
		Title:        req.Title,
		Quantity:     req.Quantity,
		Notes:        nullString(req.Notes),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to add item")
	}
	newID, _ := result.LastInsertId()

	row, err := h.queries.GetBookDvdItemByID(ctx, uint64(newID))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch new item")
	}
	return &thothv1.AddBookDvdItemResponse{Item: model.BookDvdItemToProto(model.BookDvdItemFromDB(row))}, nil
}

func (h *BookDvdItemHandler) ListBookDvdItems(ctx context.Context, req *thothv1.ListBookDvdItemsRequest) (*thothv1.ListBookDvdItemsResponse, error) {
	rows, err := h.queries.ListBookDvdItemsByCollection(ctx, uint64(req.CollectionId))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list items")
	}
	out := make([]*thothv1.BookDvdItem, len(rows))
	for i, r := range rows {
		out[i] = model.BookDvdItemToProto(model.BookDvdItemFromDB(r))
	}
	return &thothv1.ListBookDvdItemsResponse{Items: out}, nil
}

func (h *BookDvdItemHandler) UpdateBookDvdItem(ctx context.Context, req *thothv1.UpdateBookDvdItemRequest) (*thothv1.UpdateBookDvdItemResponse, error) {
	row, err := h.queries.GetBookDvdItemByID(ctx, uint64(req.Id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "item not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch item")
	}
	if _, err := requireCollectionOwner(ctx, h.queries, row.CollectionID); err != nil {
		return nil, err
	}

	if err := h.queries.UpdateBookDvdItem(ctx, db.UpdateBookDvdItemParams{
		Title:    req.Title,
		Quantity: req.Quantity,
		Notes:    nullString(req.Notes),
		ID:       uint64(req.Id),
	}); err != nil {
		return nil, status.Error(codes.Internal, "failed to update item")
	}

	updated, err := h.queries.GetBookDvdItemByID(ctx, uint64(req.Id))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch updated item")
	}
	return &thothv1.UpdateBookDvdItemResponse{Item: model.BookDvdItemToProto(model.BookDvdItemFromDB(updated))}, nil
}

func (h *BookDvdItemHandler) RemoveBookDvdItem(ctx context.Context, req *thothv1.RemoveBookDvdItemRequest) (*thothv1.RemoveBookDvdItemResponse, error) {
	row, err := h.queries.GetBookDvdItemByID(ctx, uint64(req.Id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "item not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch item")
	}
	if _, err := requireCollectionOwner(ctx, h.queries, row.CollectionID); err != nil {
		return nil, err
	}
	if err := h.queries.DeleteBookDvdItem(ctx, uint64(req.Id)); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete item")
	}
	return &thothv1.RemoveBookDvdItemResponse{}, nil
}
