package handler

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/klaital/thoth/internal/auth"
	"github.com/klaital/thoth/internal/db"
	thothv1 "github.com/klaital/thoth/internal/gen/thoth/v1"
	"github.com/klaital/thoth/internal/model"
)

type DeckHandler struct {
	thothv1.UnimplementedDeckServiceServer
	queries *db.Queries
}

func NewDeckHandler(queries *db.Queries) *DeckHandler {
	return &DeckHandler{queries: queries}
}

func (h *DeckHandler) CreateDeck(ctx context.Context, req *thothv1.CreateDeckRequest) (*thothv1.CreateDeckResponse, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	result, err := h.queries.CreateDeck(ctx, db.CreateDeckParams{
		OwnerID:  uint64(userID),
		Name:     name,
		Notes:    nullString(req.Notes),
		IsPublic: req.IsPublic,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create deck")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve new deck id")
	}
	d, err := h.queries.GetDeckByID(ctx, uint64(id))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch new deck")
	}
	return &thothv1.CreateDeckResponse{Deck: model.DeckToProto(model.DeckFromDB(d))}, nil
}

func (h *DeckHandler) GetDeck(ctx context.Context, req *thothv1.GetDeckRequest) (*thothv1.GetDeckResponse, error) {
	d, err := h.readableDeck(ctx, uint64(req.Id))
	if err != nil {
		return nil, err
	}
	out := model.DeckToProto(model.DeckFromDB(d))
	if err := h.attachCommanders(ctx, []*thothv1.Deck{out}); err != nil {
		return nil, err
	}
	return &thothv1.GetDeckResponse{Deck: out}, nil
}

func (h *DeckHandler) ListDecks(ctx context.Context, req *thothv1.ListDecksRequest) (*thothv1.ListDecksResponse, error) {
	callerID, callerErr := auth.UserIDFromContext(ctx)
	var ownerID int64
	if req.OwnerId != nil {
		ownerID = *req.OwnerId
	} else {
		if callerErr != nil {
			return nil, status.Error(codes.Unauthenticated, "not authenticated")
		}
		ownerID = callerID
	}

	var rows []db.Deck
	var err error
	// Owners see all their decks; everyone else only public ones.
	if callerErr == nil && callerID == ownerID {
		rows, err = h.queries.ListDecksByOwner(ctx, uint64(ownerID))
	} else {
		rows, err = h.queries.ListPublicDecksByOwner(ctx, uint64(ownerID))
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list decks")
	}
	out := make([]*thothv1.Deck, len(rows))
	for i, r := range rows {
		out[i] = model.DeckToProto(model.DeckFromDB(r))
	}
	if err := h.attachCommanders(ctx, out); err != nil {
		return nil, err
	}
	return &thothv1.ListDecksResponse{Decks: out}, nil
}

func (h *DeckHandler) UpdateDeck(ctx context.Context, req *thothv1.UpdateDeckRequest) (*thothv1.UpdateDeckResponse, error) {
	if _, err := h.ownedDeck(ctx, uint64(req.Id)); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if err := h.queries.UpdateDeck(ctx, db.UpdateDeckParams{
		Name:     name,
		Notes:    nullString(req.Notes),
		IsPublic: req.IsPublic,
		ID:       uint64(req.Id),
	}); err != nil {
		return nil, status.Error(codes.Internal, "failed to update deck")
	}
	d, err := h.queries.GetDeckByID(ctx, uint64(req.Id))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch updated deck")
	}
	out := model.DeckToProto(model.DeckFromDB(d))
	if err := h.attachCommanders(ctx, []*thothv1.Deck{out}); err != nil {
		return nil, err
	}
	return &thothv1.UpdateDeckResponse{Deck: out}, nil
}

func (h *DeckHandler) DeleteDeck(ctx context.Context, req *thothv1.DeleteDeckRequest) (*thothv1.DeleteDeckResponse, error) {
	if _, err := h.ownedDeck(ctx, uint64(req.Id)); err != nil {
		return nil, err
	}
	if err := h.queries.DeleteDeck(ctx, uint64(req.Id)); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete deck")
	}
	return &thothv1.DeleteDeckResponse{}, nil
}

func (h *DeckHandler) ListDeckCards(ctx context.Context, req *thothv1.ListDeckCardsRequest) (*thothv1.ListDeckCardsResponse, error) {
	if _, err := h.readableDeck(ctx, uint64(req.DeckId)); err != nil {
		return nil, err
	}
	rows, err := h.queries.ListDeckCards(ctx, uint64(req.DeckId))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list deck cards")
	}
	out := make([]*thothv1.DeckCard, len(rows))
	for i, r := range rows {
		out[i] = model.DeckCardToProto(model.DeckCardFromDB(r))
	}
	return &thothv1.ListDeckCardsResponse{Cards: out}, nil
}

func (h *DeckHandler) AddDeckCard(ctx context.Context, req *thothv1.AddDeckCardRequest) (*thothv1.AddDeckCardResponse, error) {
	if _, err := h.ownedDeck(ctx, uint64(req.DeckId)); err != nil {
		return nil, err
	}
	var name string
	var printing sql.NullString
	if req.PreferredScryfallCardId != "" {
		card, err := h.queries.GetScryfallCardByID(ctx, req.PreferredScryfallCardId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, status.Error(codes.InvalidArgument, "unknown printing")
			}
			return nil, status.Error(codes.Internal, "failed to fetch printing")
		}
		name = card.Name
		printing = nullString(card.ID)
	} else {
		var err error
		if name, err = h.resolveCardName(ctx, req.CardName); err != nil {
			return nil, err
		}
	}
	card, err := h.addCard(ctx, uint64(req.DeckId), name, model.DeckBoardFromProto(req.Board), req.Quantity, printing)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to add deck card")
	}
	return &thothv1.AddDeckCardResponse{Card: card}, nil
}

func (h *DeckHandler) UpdateDeckCard(ctx context.Context, req *thothv1.UpdateDeckCardRequest) (*thothv1.UpdateDeckCardResponse, error) {
	dc, err := h.queries.GetDeckCardByID(ctx, uint64(req.Id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "deck card not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch deck card")
	}
	if _, err := h.ownedDeck(ctx, dc.DeckID); err != nil {
		return nil, err
	}

	qty := dc.Quantity
	if req.Quantity != nil {
		if *req.Quantity < 1 {
			return nil, status.Error(codes.InvalidArgument, "quantity must be at least 1; remove the card instead")
		}
		qty = *req.Quantity
	}
	board := dc.Board
	if req.Board != nil {
		board = db.DeckCardsBoard(model.DeckBoardFromProto(*req.Board))
	}
	printing := dc.PreferredScryfallCardID
	if req.PreferredScryfallCardId != nil {
		printing = nullString(*req.PreferredScryfallCardId)
		if printing.Valid {
			card, err := h.queries.GetScryfallCardByID(ctx, printing.String)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, status.Error(codes.InvalidArgument, "unknown printing")
				}
				return nil, status.Error(codes.Internal, "failed to fetch printing")
			}
			if card.Name != dc.CardName {
				return nil, status.Errorf(codes.InvalidArgument, "printing is %q, not %q", card.Name, dc.CardName)
			}
		}
	}

	resultID := dc.ID
	if board != dc.Board {
		// Unique on (deck, name, board): moving onto a board that already has
		// this card merges into that row.
		existing, err := h.queries.GetDeckCardByKey(ctx, db.GetDeckCardByKeyParams{DeckID: dc.DeckID, CardName: dc.CardName, Board: board})
		switch {
		case err == nil:
			if !existing.PreferredScryfallCardID.Valid {
				existing.PreferredScryfallCardID = printing
			}
			if err := h.queries.UpdateDeckCard(ctx, db.UpdateDeckCardParams{
				Quantity:                existing.Quantity + qty,
				Board:                   board,
				PreferredScryfallCardID: existing.PreferredScryfallCardID,
				ID:                      existing.ID,
			}); err != nil {
				return nil, status.Error(codes.Internal, "failed to update deck card")
			}
			if err := h.queries.DeleteDeckCard(ctx, dc.ID); err != nil {
				return nil, status.Error(codes.Internal, "failed to update deck card")
			}
			resultID = existing.ID
		case !errors.Is(err, sql.ErrNoRows):
			return nil, status.Error(codes.Internal, "failed to update deck card")
		}
	}
	if resultID == dc.ID {
		if err := h.queries.UpdateDeckCard(ctx, db.UpdateDeckCardParams{
			Quantity:                qty,
			Board:                   board,
			PreferredScryfallCardID: printing,
			ID:                      dc.ID,
		}); err != nil {
			return nil, status.Error(codes.Internal, "failed to update deck card")
		}
	}
	h.touch(ctx, dc.DeckID)

	row, err := h.queries.GetDeckCardDetail(ctx, resultID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch updated deck card")
	}
	return &thothv1.UpdateDeckCardResponse{Card: model.DeckCardToProto(model.DeckCardFromDB(db.ListDeckCardsRow(row)))}, nil
}

func (h *DeckHandler) RemoveDeckCard(ctx context.Context, req *thothv1.RemoveDeckCardRequest) (*thothv1.RemoveDeckCardResponse, error) {
	dc, err := h.queries.GetDeckCardByID(ctx, uint64(req.Id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "deck card not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch deck card")
	}
	if _, err := h.ownedDeck(ctx, dc.DeckID); err != nil {
		return nil, err
	}
	if err := h.queries.DeleteDeckCard(ctx, dc.ID); err != nil {
		return nil, status.Error(codes.Internal, "failed to remove deck card")
	}
	h.touch(ctx, dc.DeckID)
	return &thothv1.RemoveDeckCardResponse{}, nil
}

// addCard upserts a card onto a board (adding to an existing row's quantity)
// and returns the stored row.
func (h *DeckHandler) addCard(ctx context.Context, deckID uint64, name string, board model.DeckBoard, qty int32, printing sql.NullString) (*thothv1.DeckCard, error) {
	if qty < 1 {
		qty = 1
	}
	if _, err := h.queries.UpsertDeckCard(ctx, db.UpsertDeckCardParams{
		DeckID:                  deckID,
		CardName:                name,
		PreferredScryfallCardID: printing,
		Board:                   db.DeckCardsBoard(board),
		Quantity:                qty,
	}); err != nil {
		return nil, err
	}
	h.touch(ctx, deckID)
	dc, err := h.queries.GetDeckCardByKey(ctx, db.GetDeckCardByKeyParams{DeckID: deckID, CardName: name, Board: db.DeckCardsBoard(board)})
	if err != nil {
		return nil, err
	}
	row, err := h.queries.GetDeckCardDetail(ctx, dc.ID)
	if err != nil {
		return nil, err
	}
	return model.DeckCardToProto(model.DeckCardFromDB(db.ListDeckCardsRow(row))), nil
}

// resolveCardName returns the card name as Scryfall spells it, matching case-
// insensitively and accepting the front face of a double-faced card.
func (h *DeckHandler) resolveCardName(ctx context.Context, input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", status.Error(codes.InvalidArgument, "card_name is required")
	}
	name, err := h.queries.GetCanonicalCardName(ctx, input)
	if err == nil {
		return name, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", status.Error(codes.Internal, "failed to look up card")
	}
	name, err = h.queries.GetCardNameByFrontFace(ctx, likeEscaper.Replace(input))
	if err == nil {
		return name, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", status.Error(codes.Internal, "failed to look up card")
	}
	return "", status.Errorf(codes.InvalidArgument, "no card named %q", input)
}

// touch bumps the deck's updated_at so recently edited decks list first.
// Best effort: a failure here shouldn't fail the edit itself.
func (h *DeckHandler) touch(ctx context.Context, deckID uint64) {
	_ = h.queries.TouchDeck(ctx, deckID)
}

// readableDeck fetches a deck the caller may view: their own, or a public one.
// Other decks are reported as not found.
func (h *DeckHandler) readableDeck(ctx context.Context, id uint64) (db.Deck, error) {
	d, err := h.queries.GetDeckByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Deck{}, status.Error(codes.NotFound, "deck not found")
		}
		return db.Deck{}, status.Error(codes.Internal, "failed to fetch deck")
	}
	if d.IsPublic {
		return d, nil
	}
	callerID, err := auth.UserIDFromContext(ctx)
	if err != nil || callerID != int64(d.OwnerID) {
		return db.Deck{}, status.Error(codes.NotFound, "deck not found")
	}
	return d, nil
}

// ownedDeck fetches a deck the caller owns.
func (h *DeckHandler) ownedDeck(ctx context.Context, id uint64) (db.Deck, error) {
	d, err := h.queries.GetDeckByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Deck{}, status.Error(codes.NotFound, "deck not found")
		}
		return db.Deck{}, status.Error(codes.Internal, "failed to fetch deck")
	}
	callerID, err := auth.UserIDFromContext(ctx)
	if err != nil || callerID != int64(d.OwnerID) {
		return db.Deck{}, status.Error(codes.PermissionDenied, "not the deck owner")
	}
	return d, nil
}

// attachCommanders fills in each deck's commander cards with one query.
func (h *DeckHandler) attachCommanders(ctx context.Context, decks []*thothv1.Deck) error {
	if len(decks) == 0 {
		return nil
	}
	byID := make(map[uint64]*thothv1.Deck, len(decks))
	ids := make([]uint64, len(decks))
	for i, d := range decks {
		ids[i] = uint64(d.Id)
		byID[ids[i]] = d
	}
	rows, err := h.queries.ListDeckCommanders(ctx, ids)
	if err != nil {
		return status.Error(codes.Internal, "failed to fetch commanders")
	}
	for _, r := range rows {
		d := byID[r.DeckID]
		d.Commanders = append(d.Commanders, &thothv1.DeckCommander{CardName: r.CardName, ImageUri: r.ImageUri.String})
	}
	return nil
}
