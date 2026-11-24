package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/klaital/thoth/internal/auth"
	"github.com/klaital/thoth/internal/db"
	thothv1 "github.com/klaital/thoth/internal/gen/thoth/v1"
	"github.com/klaital/thoth/internal/model"
	"github.com/klaital/thoth/internal/telemetry"
)

type CardItemHandler struct {
	thothv1.UnimplementedCardItemServiceServer
	sqlDB   *sql.DB
	queries *db.Queries
}

func NewCardItemHandler(sqlDB *sql.DB, queries *db.Queries) *CardItemHandler {
	return &CardItemHandler{sqlDB: sqlDB, queries: queries}
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
	foil := db.CollectionCardsFoilTreatment(model.FoilTreatmentFromProto(req.FoilTreatment))
	lang, err := normalizeLanguage(req.Language)
	if err != nil {
		return nil, err
	}

	if _, err := h.queries.UpsertCardItem(ctx, db.UpsertCardItemParams{
		CollectionID:   uint64(req.CollectionId),
		ScryfallCardID: req.ScryfallCardId,
		FoilTreatment:  foil,
		Language:       lang,
		Quantity:       req.Quantity,
		Notes:          nullString(req.Notes),
	}); err != nil {
		return nil, status.Error(codes.Internal, "failed to add card item")
	}
	telemetry.CardsAdded.Add(ctx, int64(req.Quantity), telemetry.SourceSingle)

	// Fetch the canonical row back to return.
	row, err := h.queries.GetCardItemByKey(ctx, db.GetCardItemByKeyParams{
		CollectionID:   uint64(req.CollectionId),
		ScryfallCardID: req.ScryfallCardId,
		FoilTreatment:  foil,
		Language:       lang,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch card item")
	}
	return &thothv1.AddCardItemResponse{Item: model.CardItemToProto(model.CardItemFromDB(db.ListCardItemsByCollectionRow(row)))}, nil
}

func (h *CardItemHandler) ListCardItems(ctx context.Context, req *thothv1.ListCardItemsRequest) (*thothv1.ListCardItemsResponse, error) {
	c, err := h.queries.GetCollectionByID(ctx, uint64(req.CollectionId))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "collection not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch collection")
	}
	if err := authorizeCollectionRead(ctx, c); err != nil {
		return nil, err
	}
	rows, err := h.queries.ListCardItemsByCollection(ctx, uint64(req.CollectionId))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list card items")
	}
	out := make([]*thothv1.CardItem, len(rows))
	for i, r := range rows {
		out[i] = model.CardItemToProto(model.CardItemFromDB(r))
	}
	if callerID, err := auth.UserIDFromContext(ctx); err == nil && callerID == int64(c.OwnerID) {
		if err := attachCheckouts(ctx, h.queries, out); err != nil {
			return nil, err
		}
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

	lang := row.Language
	if req.Language != nil {
		if lang, err = normalizeLanguage(*req.Language); err != nil {
			return nil, err
		}
	}
	resultID := uint64(req.Id)

	err = runInTx(ctx, h.sqlDB, h.queries, func(q *db.Queries) error {
		// Checked-out copies are still owned, so the quantity can't drop
		// below them. They'd have to be returned first.
		out, err := q.SumCheckoutsForCard(ctx, uint64(req.Id))
		if err != nil {
			return err
		}
		if int64(req.Quantity) < out {
			return status.Error(codes.FailedPrecondition,
				fmt.Sprintf("%d %s checked out to decks; return them before lowering the quantity", out, plural(out, "copy is", "copies are")))
		}

		if lang != row.Language {
			// The unique key includes language, so changing to a language the
			// collection already holds for this printing and foil merges the rows.
			existing, err := q.FindCardItem(ctx, db.FindCardItemParams{
				CollectionID:   row.CollectionID,
				ScryfallCardID: row.ScryfallCardID,
				FoilTreatment:  row.FoilTreatment,
				Language:       lang,
			})
			switch {
			case err == nil:
				if err := q.UpdateCardItem(ctx, db.UpdateCardItemParams{
					Quantity: existing.Quantity + req.Quantity,
					Notes:    existing.Notes,
					Language: lang,
					ID:       existing.ID,
				}); err != nil {
					return err
				}
				if err := q.MergeCheckoutsInto(ctx, db.MergeCheckoutsIntoParams{ToID: existing.ID, FromID: uint64(req.Id)}); err != nil {
					return err
				}
				if err := q.DeleteCardItem(ctx, uint64(req.Id)); err != nil {
					return err
				}
				resultID = existing.ID
				return nil
			case !errors.Is(err, sql.ErrNoRows):
				return err
			}
		}

		return q.UpdateCardItem(ctx, db.UpdateCardItemParams{
			Quantity: req.Quantity,
			Notes:    nullString(req.Notes),
			Language: lang,
			ID:       uint64(req.Id),
		})
	})
	if err != nil {
		return nil, statusOr(err, "failed to update card item")
	}

	rows, err := h.queries.ListCardItemsByCollection(ctx, row.CollectionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch updated item")
	}
	for _, r := range rows {
		if r.ID == resultID {
			item := model.CardItemToProto(model.CardItemFromDB(r))
			if err := attachCheckouts(ctx, h.queries, []*thothv1.CardItem{item}); err != nil {
				return nil, err
			}
			return &thothv1.UpdateCardItemResponse{Item: item}, nil
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
