package handler

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/klaital/thoth/internal/db"
	thothv1 "github.com/klaital/thoth/internal/gen/thoth/v1"
	"github.com/klaital/thoth/internal/model"
	"github.com/klaital/thoth/internal/telemetry"
)

// Deck ownership and check-out (TH-16, TH-17).
//
// A checkout holds some copies of a collection card row for a deck. The row
// stays in its collection, so "free" copies are quantity minus all checkouts,
// and a return always goes back to the collection the card came from.

func (h *DeckHandler) GetDeckOwnership(ctx context.Context, req *thothv1.GetDeckOwnershipRequest) (*thothv1.GetDeckOwnershipResponse, error) {
	d, err := h.ownedDeck(ctx, uint64(req.DeckId))
	if err != nil {
		return nil, err
	}
	deckCards, err := h.queries.ListDeckCards(ctx, d.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list deck cards")
	}
	rows, err := h.queries.ListOwnedCopiesForDeck(ctx, db.ListOwnedCopiesForDeckParams{DeckID: d.ID, OwnerID: d.OwnerID})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list owned cards")
	}
	return buildOwnership(deckCards, rows), nil
}

// buildOwnership groups owned copies by card name, in deck order. A card on
// several boards is one entry with the quantities summed.
func buildOwnership(deckCards []db.ListDeckCardsRow, rows []db.ListOwnedCopiesForDeckRow) *thothv1.GetDeckOwnershipResponse {
	type entry struct {
		needed    int32
		preferred string
		copies    []model.OwnedCopy
	}
	var order []string
	byName := map[string]*entry{}
	get := func(name string) *entry {
		e, ok := byName[name]
		if !ok {
			e = &entry{}
			byName[name] = e
			order = append(order, name)
		}
		return e
	}
	for _, dc := range deckCards {
		e := get(dc.CardName)
		e.needed += dc.Quantity
		if e.preferred == "" && dc.PreferredScryfallCardID.Valid {
			e.preferred = dc.PreferredScryfallCardID.String
		}
	}
	// Copies checked out to the deck under a name it no longer lists come
	// last, so they still show up for return.
	for _, r := range rows {
		e := get(r.CardName)
		e.copies = append(e.copies, model.OwnedCopyFromDB(r))
	}

	resp := &thothv1.GetDeckOwnershipResponse{}
	for _, name := range order {
		e := byName[name]
		o := &thothv1.DeckCardOwnership{CardName: name, Needed: e.needed}
		for _, c := range e.copies {
			o.CheckedOutHere += c.CheckedOutHere
			o.CheckedOutElsewhere += c.CheckedOutElsewhere
			o.Free += c.Free()
		}
		o.Missing = max(0, o.Needed-o.CheckedOutHere-o.Free)
		suggestCopies(o.Needed-o.CheckedOutHere, e.copies, e.preferred)
		for _, c := range e.copies {
			o.Copies = append(o.Copies, model.OwnedCopyToProto(c))
		}
		resp.Cards = append(resp.Cards, o)
		resp.TotalMissing += o.Missing
		resp.TotalCheckedOut += o.CheckedOutHere
	}
	return resp
}

// suggestCopies sets SuggestedQuantity on copies so that up to need free
// copies are pulled. Preference order: the deck's preferred printing,
// nonfoil before foil before etched, cheapest (unknown prices last), English,
// then the collection that can supply the most copies, so fewer boxes get
// opened. copies keeps its order.
func suggestCopies(need int32, copies []model.OwnedCopy, preferredPrinting string) {
	for i := range copies {
		copies[i].SuggestedQuantity = 0
	}
	if need <= 0 {
		return
	}
	freeByCollection := map[int64]int32{}
	for _, c := range copies {
		freeByCollection[c.CollectionID] += c.Free()
	}
	idx := make([]int, len(copies))
	for i := range idx {
		idx[i] = i
	}
	slices.SortStableFunc(idx, func(a, b int) int {
		ca, cb := copies[a], copies[b]
		return cmp.Or(
			boolRank(ca.ScryfallCardID != preferredPrinting || preferredPrinting == "",
				cb.ScryfallCardID != preferredPrinting || preferredPrinting == ""),
			cmp.Compare(foilRank(ca.FoilTreatment), foilRank(cb.FoilTreatment)),
			comparePrice(ca.UnitPriceUSD, cb.UnitPriceUSD),
			boolRank(ca.Language != model.DefaultLanguage, cb.Language != model.DefaultLanguage),
			-cmp.Compare(freeByCollection[ca.CollectionID], freeByCollection[cb.CollectionID]),
			cmp.Compare(ca.CollectionID, cb.CollectionID),
			cmp.Compare(ca.CollectionCardID, cb.CollectionCardID),
		)
	})
	for _, i := range idx {
		if need == 0 {
			break
		}
		take := min(need, copies[i].Free())
		copies[i].SuggestedQuantity = take
		need -= take
	}
}

// boolRank orders false before true.
func boolRank(a, b bool) int {
	switch {
	case a == b:
		return 0
	case !a:
		return -1
	default:
		return 1
	}
}

func foilRank(f model.FoilTreatment) int {
	switch f {
	case model.FoilTreatmentNonfoil:
		return 0
	case model.FoilTreatmentFoil:
		return 1
	default:
		return 2
	}
}

// comparePrice orders by price, with unknown prices last.
func comparePrice(a, b string) int {
	pa, errA := strconv.ParseFloat(a, 64)
	pb, errB := strconv.ParseFloat(b, 64)
	switch {
	case errA != nil && errB != nil:
		return 0
	case errA != nil:
		return 1
	case errB != nil:
		return -1
	default:
		return cmp.Compare(pa, pb)
	}
}

func (h *DeckHandler) CheckOutDeckCards(ctx context.Context, req *thothv1.CheckOutDeckCardsRequest) (*thothv1.CheckOutDeckCardsResponse, error) {
	d, err := h.ownedDeck(ctx, uint64(req.DeckId))
	if err != nil {
		return nil, err
	}
	want := map[uint64]int32{}
	var ids []uint64
	for _, p := range req.Picks {
		if p.Quantity <= 0 {
			return nil, status.Error(codes.InvalidArgument, "each pick needs a quantity of at least 1")
		}
		id := uint64(p.CollectionCardId)
		if _, ok := want[id]; !ok {
			ids = append(ids, id)
		}
		want[id] += p.Quantity
	}
	if len(ids) == 0 {
		return &thothv1.CheckOutDeckCardsResponse{}, nil
	}

	err = runInTx(ctx, h.sqlDB, h.queries, func(q *db.Queries) error {
		if _, err := q.LockCollectionCards(ctx, ids); err != nil {
			return err
		}
		sources, err := q.GetCheckoutSources(ctx, ids)
		if err != nil {
			return err
		}
		if len(sources) != len(ids) {
			return status.Error(codes.NotFound, "card not found")
		}
		for _, s := range sources {
			// Only the deck owner's own cards can go into their deck.
			if s.OwnerID != d.OwnerID || s.CollectionType != db.CollectionsTypeMtgCards {
				return status.Error(codes.NotFound, "card not found")
			}
			free := int64(s.Quantity) - s.CheckedOutTotal
			if int64(want[s.ID]) > free {
				return status.Error(codes.FailedPrecondition,
					fmt.Sprintf("only %d free %s of %s", max(free, 0), plural(free, "copy", "copies"), s.CardName))
			}
		}
		for _, id := range ids {
			if err := q.UpsertCheckout(ctx, db.UpsertCheckoutParams{
				DeckID: d.ID, CollectionCardID: id, Quantity: want[id],
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, statusOr(err, "failed to check out cards")
	}
	var checkedOut int64
	for _, n := range want {
		checkedOut += int64(n)
	}
	telemetry.DeckCheckouts.Add(ctx, checkedOut)
	return &thothv1.CheckOutDeckCardsResponse{}, nil
}

func (h *DeckHandler) ListDeckCheckouts(ctx context.Context, req *thothv1.ListDeckCheckoutsRequest) (*thothv1.ListDeckCheckoutsResponse, error) {
	d, err := h.ownedDeck(ctx, uint64(req.DeckId))
	if err != nil {
		return nil, err
	}
	rows, err := h.queries.ListCheckoutsByDeck(ctx, d.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list checked-out cards")
	}
	out := make([]*thothv1.DeckCheckout, len(rows))
	for i, r := range rows {
		out[i] = model.DeckCheckoutToProto(model.DeckCheckoutFromDB(r))
	}
	return &thothv1.ListDeckCheckoutsResponse{Checkouts: out}, nil
}

func (h *DeckHandler) ReturnDeckCards(ctx context.Context, req *thothv1.ReturnDeckCardsRequest) (*thothv1.ReturnDeckCardsResponse, error) {
	d, err := h.ownedDeck(ctx, uint64(req.DeckId))
	if err != nil {
		return nil, err
	}
	var returned int64
	err = runInTx(ctx, h.sqlDB, h.queries, func(q *db.Queries) error {
		returned = 0
		for _, p := range req.Picks {
			x, err := q.GetCheckoutForUpdate(ctx, db.GetCheckoutForUpdateParams{
				DeckID: d.ID, CollectionCardID: uint64(p.CollectionCardId),
			})
			if errors.Is(err, sql.ErrNoRows) {
				return status.Error(codes.NotFound, "card is not checked out to this deck")
			}
			if err != nil {
				return err
			}
			if p.Quantity <= 0 || p.Quantity >= x.Quantity {
				returned += int64(x.Quantity)
				err = q.DeleteCheckout(ctx, x.ID)
			} else {
				returned += int64(p.Quantity)
				err = q.SetCheckoutQuantity(ctx, db.SetCheckoutQuantityParams{Quantity: x.Quantity - p.Quantity, ID: x.ID})
			}
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, statusOr(err, "failed to return cards")
	}
	telemetry.DeckReturns.Add(ctx, returned)
	return &thothv1.ReturnDeckCardsResponse{}, nil
}

// attachCheckouts fills in which decks hold copies of each card item. Call it
// only for the collection's owner: deck names may be private.
func attachCheckouts(ctx context.Context, q *db.Queries, items []*thothv1.CardItem) error {
	if len(items) == 0 {
		return nil
	}
	byID := make(map[uint64]*thothv1.CardItem, len(items))
	ids := make([]uint64, len(items))
	for i, it := range items {
		ids[i] = uint64(it.Id)
		byID[ids[i]] = it
	}
	rows, err := q.ListCheckoutsByCollectionCardIDs(ctx, ids)
	if err != nil {
		return status.Error(codes.Internal, "failed to fetch checked-out cards")
	}
	for _, r := range rows {
		it := byID[r.CollectionCardID]
		it.Checkouts = append(it.Checkouts, model.CardCheckoutToProto(r))
	}
	return nil
}

// statusOr passes gRPC status errors through and hides anything else behind
// an Internal error with msg.
func statusOr(err error, msg string) error {
	if _, ok := status.FromError(err); ok {
		return err
	}
	return status.Error(codes.Internal, msg)
}

func plural(n int64, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
