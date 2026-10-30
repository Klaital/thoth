package handler

import (
	"context"
	"database/sql"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/klaital/thoth/internal/db"
	thothv1 "github.com/klaital/thoth/internal/gen/thoth/v1"
)

// itemKind groups collection types by the table their items live in. Card and
// deck collections share one kind so cards can move freely between them;
// books and DVDs share a table but stay separate kinds.
func itemKind(t db.CollectionsType) db.CollectionsType {
	if t == db.CollectionsTypeMtgDeck {
		return db.CollectionsTypeMtgCards
	}
	return t
}

func (h *CollectionHandler) MoveCollectionItems(ctx context.Context, req *thothv1.MoveCollectionItemsRequest) (*thothv1.MoveCollectionItemsResponse, error) {
	if req.CollectionId == req.TargetCollectionId {
		return nil, status.Error(codes.InvalidArgument, "target collection must differ from source")
	}
	src, err := requireCollectionOwner(ctx, h.queries, uint64(req.CollectionId))
	if err != nil {
		return nil, err
	}
	dst, err := requireCollectionOwner(ctx, h.queries, uint64(req.TargetCollectionId))
	if err != nil {
		return nil, err
	}
	if itemKind(src.Type) != itemKind(dst.Type) {
		return nil, status.Error(codes.InvalidArgument, "target collection holds a different kind of item")
	}
	if len(req.ItemIds) == 0 {
		return &thothv1.MoveCollectionItemsResponse{}, nil
	}
	ids := toUint64s(req.ItemIds)

	err = h.inTx(ctx, func(q *db.Queries) error {
		switch itemKind(src.Type) {
		case db.CollectionsTypeMtgCards:
			return moveCardItems(ctx, q, src.ID, dst.ID, ids)
		case db.CollectionsTypeMtgSealed:
			return q.MoveSealedItems(ctx, db.MoveSealedItemsParams{
				TargetCollectionID: dst.ID,
				SourceCollectionID: src.ID,
				Ids:                ids,
			})
		default:
			return q.MoveBookDvdItems(ctx, db.MoveBookDvdItemsParams{
				TargetCollectionID: dst.ID,
				SourceCollectionID: src.ID,
				Ids:                ids,
			})
		}
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to move items")
	}
	return &thothv1.MoveCollectionItemsResponse{}, nil
}

// moveCardItems moves cards one at a time because the target may already
// hold the same card, foil treatment and category, in which case the
// quantities are merged into the existing row.
func moveCardItems(ctx context.Context, q *db.Queries, srcID, dstID uint64, ids []uint64) error {
	rows, err := q.ListCardItemsByIDs(ctx, db.ListCardItemsByIDsParams{CollectionID: srcID, Ids: ids})
	if err != nil {
		return err
	}
	for _, r := range rows {
		existing, err := q.FindCardItem(ctx, db.FindCardItemParams{
			CollectionID:   dstID,
			ScryfallCardID: r.ScryfallCardID,
			FoilTreatment:  r.FoilTreatment,
			Category:       r.Category,
		})
		switch {
		case err == nil:
			notes := existing.Notes
			if !notes.Valid {
				notes = r.Notes
			}
			if err := q.UpdateCardItem(ctx, db.UpdateCardItemParams{
				Quantity: existing.Quantity + r.Quantity,
				Notes:    notes,
				Category: existing.Category,
				ID:       existing.ID,
			}); err != nil {
				return err
			}
			if err := q.DeleteCardItem(ctx, r.ID); err != nil {
				return err
			}
		case errors.Is(err, sql.ErrNoRows):
			if err := q.MoveCardItem(ctx, db.MoveCardItemParams{CollectionID: dstID, ID: r.ID}); err != nil {
				return err
			}
		default:
			return err
		}
	}
	return nil
}

func (h *CollectionHandler) RemoveCollectionItems(ctx context.Context, req *thothv1.RemoveCollectionItemsRequest) (*thothv1.RemoveCollectionItemsResponse, error) {
	c, err := requireCollectionOwner(ctx, h.queries, uint64(req.CollectionId))
	if err != nil {
		return nil, err
	}
	if len(req.ItemIds) == 0 {
		return &thothv1.RemoveCollectionItemsResponse{}, nil
	}
	ids := toUint64s(req.ItemIds)

	switch itemKind(c.Type) {
	case db.CollectionsTypeMtgCards:
		err = h.queries.DeleteCardItemsByIDs(ctx, db.DeleteCardItemsByIDsParams{CollectionID: c.ID, Ids: ids})
	case db.CollectionsTypeMtgSealed:
		err = h.queries.DeleteSealedItemsByIDs(ctx, db.DeleteSealedItemsByIDsParams{CollectionID: c.ID, Ids: ids})
	default:
		err = h.queries.DeleteBookDvdItemsByIDs(ctx, db.DeleteBookDvdItemsByIDsParams{CollectionID: c.ID, Ids: ids})
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to remove items")
	}
	return &thothv1.RemoveCollectionItemsResponse{}, nil
}

// inTx runs fn with queries bound to a transaction, committing if it succeeds.
func (h *CollectionHandler) inTx(ctx context.Context, fn func(q *db.Queries) error) error {
	tx, err := h.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(h.queries.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

func toUint64s(ids []int64) []uint64 {
	out := make([]uint64, len(ids))
	for i, id := range ids {
		out[i] = uint64(id)
	}
	return out
}
