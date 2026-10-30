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

type CardItemHandler struct {
	thothv1.UnimplementedCardItemServiceServer
	queries *db.Queries
}

func NewCardItemHandler(queries *db.Queries) *CardItemHandler {
	return &CardItemHandler{queries: queries}
}

func (h *CardItemHandler) AddCardItem(ctx context.Context, req *thothv1.AddCardItemRequest) (*thothv1.AddCardItemResponse, error) {
	if _, err := requireCollectionOwner(ctx, h.queries, uint64(req.CollectionId)); err != nil {
		return nil, err
	}
	if req.ScryfallCardId == "" {
		return nil, status.Error(codes.InvalidArgument, "scryfall_card_id is required")
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}
	foil := model.FoilTreatmentFromProto(req.FoilTreatment)
	category := model.CardCategoryFromProto(req.Category)

	_, err := h.queries.UpsertCardItem(ctx, db.UpsertCardItemParams{
		CollectionID:   uint64(req.CollectionId),
		ScryfallCardID: req.ScryfallCardId,
		FoilTreatment:  db.CardItemsFoilTreatment(foil),
		Category:       db.CardItemsCategory(category),
		Quantity:       req.Quantity,
		Notes:          nullString(req.Notes),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to add card item")
	}

	// Fetch the canonical row back to return.
	rows, err := h.queries.ListCardItemsByCollection(ctx, uint64(req.CollectionId))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch card item")
	}
	for _, r := range rows {
		if r.ScryfallCardID == req.ScryfallCardId && model.FoilTreatment(r.FoilTreatment) == foil && model.CardCategory(r.Category) == category {
			return &thothv1.AddCardItemResponse{Item: model.CardItemToProto(model.CardItemFromDB(r))}, nil
		}
	}
	return nil, status.Error(codes.Internal, "failed to locate upserted card item")
}

func (h *CardItemHandler) ListCardItems(ctx context.Context, req *thothv1.ListCardItemsRequest) (*thothv1.ListCardItemsResponse, error) {
	rows, err := h.queries.ListCardItemsByCollection(ctx, uint64(req.CollectionId))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list card items")
	}
	out := make([]*thothv1.CardItem, len(rows))
	for i, r := range rows {
		out[i] = model.CardItemToProto(model.CardItemFromDB(r))
	}
	return &thothv1.ListCardItemsResponse{Items: out}, nil
}

func (h *CardItemHandler) UpdateCardItem(ctx context.Context, req *thothv1.UpdateCardItemRequest) (*thothv1.UpdateCardItemResponse, error) {
	row, err := h.queries.GetCardItemByID(ctx, uint64(req.Id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "card item not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch card item")
	}
	if _, err := requireCollectionOwner(ctx, h.queries, row.CollectionID); err != nil {
		return nil, err
	}

	category := row.Category
	if req.Category != nil {
		category = db.CardItemsCategory(model.CardCategoryFromProto(*req.Category))
	}
	resultID := uint64(req.Id)

	if category != row.Category {
		// The unique key includes category, so moving into a category that
		// already holds this card and foil merges into that row.
		existing, err := h.queries.FindCardItem(ctx, db.FindCardItemParams{
			CollectionID:   row.CollectionID,
			ScryfallCardID: row.ScryfallCardID,
			FoilTreatment:  row.FoilTreatment,
			Category:       category,
		})
		switch {
		case err == nil:
			if err := h.queries.UpdateCardItem(ctx, db.UpdateCardItemParams{
				Quantity: existing.Quantity + req.Quantity,
				Notes:    existing.Notes,
				Category: category,
				ID:       existing.ID,
			}); err != nil {
				return nil, status.Error(codes.Internal, "failed to update card item")
			}
			if err := h.queries.DeleteCardItem(ctx, uint64(req.Id)); err != nil {
				return nil, status.Error(codes.Internal, "failed to update card item")
			}
			resultID = existing.ID
		case !errors.Is(err, sql.ErrNoRows):
			return nil, status.Error(codes.Internal, "failed to update card item")
		}
	}

	if resultID == uint64(req.Id) {
		if err := h.queries.UpdateCardItem(ctx, db.UpdateCardItemParams{
			Quantity: req.Quantity,
			Notes:    nullString(req.Notes),
			Category: category,
			ID:       uint64(req.Id),
		}); err != nil {
			return nil, status.Error(codes.Internal, "failed to update card item")
		}
	}

	rows, err := h.queries.ListCardItemsByCollection(ctx, row.CollectionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch updated item")
	}
	for _, r := range rows {
		if r.ID == resultID {
			return &thothv1.UpdateCardItemResponse{Item: model.CardItemToProto(model.CardItemFromDB(r))}, nil
		}
	}
	return nil, status.Error(codes.Internal, "failed to locate updated card item")
}

func (h *CardItemHandler) RemoveCardItem(ctx context.Context, req *thothv1.RemoveCardItemRequest) (*thothv1.RemoveCardItemResponse, error) {
	row, err := h.queries.GetCardItemByID(ctx, uint64(req.Id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "card item not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch card item")
	}
	if _, err := requireCollectionOwner(ctx, h.queries, row.CollectionID); err != nil {
		return nil, err
	}
	if err := h.queries.DeleteCardItem(ctx, uint64(req.Id)); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete card item")
	}
	return &thothv1.RemoveCardItemResponse{}, nil
}
