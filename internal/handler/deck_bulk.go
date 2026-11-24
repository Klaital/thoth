package handler

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/klaital/thoth/internal/db"
	thothv1 "github.com/klaital/thoth/internal/gen/thoth/v1"
	"github.com/klaital/thoth/internal/model"
	"github.com/klaital/thoth/internal/telemetry"
)

// deckNameLine matches a deck list line that names a card without a printing:
//
//	4 Lightning Bolt
//	1x Sol Ring
//	City of Brass
//
// Groups: optional quantity, name. A trailing foil marker is tolerated and
// ignored, since deck cards don't track finish.
var deckNameLine = regexp.MustCompile(`^(?:(\d+)x?\s+)?(.+?)(?:\s+\*[FE]\*)?$`)

// deckLine is one parsed line of a deck bulk import. SetCode and
// CollectorNumber are set only when the line names a printing.
type deckLine struct {
	Quantity        int32
	Name            string
	SetCode         string
	CollectorNumber string
}

// parseDeckLine accepts Moxfield lines ("4 Lightning Bolt (M20) 141 *F*") and
// plain name lines ("4 Lightning Bolt", "Lightning Bolt").
func parseDeckLine(line string) (deckLine, error) {
	line = strings.TrimSpace(line)
	if p, err := parseMoxfieldLine(line); err == nil {
		return deckLine{Quantity: p.Quantity, Name: p.Name, SetCode: p.SetCode, CollectorNumber: p.CollectorNumber}, nil
	}
	m := deckNameLine.FindStringSubmatch(line)
	if m == nil || strings.TrimSpace(m[2]) == "" {
		return deckLine{}, fmt.Errorf(`expected "<qty> <name>" or "<qty> <name> (<set>) <number>"`)
	}
	qty := 1
	if m[1] != "" {
		n, err := strconv.Atoi(m[1])
		if err != nil || n <= 0 {
			return deckLine{}, fmt.Errorf("invalid quantity %q", m[1])
		}
		qty = n
	}
	return deckLine{Quantity: int32(qty), Name: strings.TrimSpace(m[2])}, nil
}

// pendingDeckLine is one card line of a deck bulk import as it moves from
// parsing through lookup to the write. name stays empty if the line failed.
type pendingDeckLine struct {
	board    model.DeckBoard
	parsed   deckLine
	name     string
	printing sql.NullString
	res      *thothv1.BulkImportDeckCardLineResult
}

// BulkImportDeckCards adds a deck list. The deck is imported as a whole: lines
// are parsed first, card lookups are batched, every write goes in one
// transaction, and the results are sent once it commits. Each step is a
// network round trip to the database, so the cost stays a handful of queries
// plus one upsert per line, rather than several queries per line.
func (h *DeckHandler) BulkImportDeckCards(req *thothv1.BulkImportDeckCardsRequest, stream grpc.ServerStreamingServer[thothv1.BulkImportDeckCardLineResult]) error {
	ctx := stream.Context()
	deckID := uint64(req.DeckId)
	if _, err := h.ownedDeck(ctx, deckID); err != nil {
		return err
	}

	lines := strings.Split(req.Text, "\n")
	cardLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if _, ok := parseBoardHeader(line); ok {
			continue
		}
		cardLines++
	}
	// Checked before importing anything so an oversized list adds nothing.
	if cardLines > maxBulkImportLines {
		return status.Errorf(codes.InvalidArgument, "too many lines (max %d)", maxBulkImportLines)
	}

	pending := make([]*pendingDeckLine, 0, cardLines)
	board := model.DeckBoardMain
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if b, ok := parseBoardHeader(line); ok {
			board = b
			continue
		}
		p := &pendingDeckLine{board: board, res: &thothv1.BulkImportDeckCardLineResult{Line: line, LineNumber: int32(i + 1)}}
		parsed, err := parseDeckLine(line)
		if err != nil {
			p.res.Error = err.Error()
		} else {
			p.parsed = parsed
			p.res.Name = parsed.Name
		}
		pending = append(pending, p)
	}

	if err := h.resolveDeckLines(ctx, pending); err != nil {
		return err
	}
	if err := h.writeDeckLines(ctx, deckID, pending); err != nil {
		return err
	}
	h.attachImportedCards(ctx, deckID, pending)

	for _, p := range pending {
		telemetry.RecordBulkLine(ctx, "deck", p.res.Error == "")
		if err := stream.Send(p.res); err != nil {
			return err
		}
	}
	return nil
}

// resolveDeckLines sets each parsed line's card name as Scryfall spells it,
// and its printing when the line names one that exists. A printing that can't
// be found doesn't fail the line: the card is added by name with a warning.
// Lines whose card can't be found get an error.
func (h *DeckHandler) resolveDeckLines(ctx context.Context, pending []*pendingDeckLine) error {
	var refs []printingRef
	for _, p := range pending {
		if p.res.Error == "" && p.parsed.SetCode != "" {
			refs = append(refs, printingRef{SetCode: p.parsed.SetCode, CollectorNumber: p.parsed.CollectorNumber})
		}
	}
	if len(refs) > 0 {
		byPrinting, err := lookupPrintings(ctx, h.queries, refs)
		if err != nil {
			return err
		}
		for _, p := range pending {
			if p.res.Error != "" || p.parsed.SetCode == "" {
				continue
			}
			label := fmt.Sprintf("%s #%s", strings.ToUpper(p.parsed.SetCode), p.parsed.CollectorNumber)
			card, ok := byPrinting[printingKey(p.parsed.SetCode, p.parsed.CollectorNumber)]
			switch {
			case !ok:
				p.res.Warning = label + " not found; added by name"
			case !cardNameMatches(p.parsed.Name, card.Name):
				p.res.Warning = fmt.Sprintf("%s is %q, not %q; added by name", label, card.Name, p.parsed.Name)
			default:
				p.name = card.Name
				p.printing = nullString(card.ID)
			}
		}
	}

	var names []string
	seenName := map[string]bool{}
	for _, p := range pending {
		if p.res.Error != "" || p.name != "" {
			continue
		}
		if key := strings.ToLower(p.parsed.Name); !seenName[key] {
			seenName[key] = true
			names = append(names, p.parsed.Name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	found, err := h.queries.GetCanonicalCardNames(ctx, names)
	if err != nil {
		return status.Error(codes.Internal, "failed to look up cards")
	}
	canonical := make(map[string]string, len(found))
	for _, n := range found {
		canonical[strings.ToLower(n)] = n
	}
	for _, p := range pending {
		if p.res.Error != "" || p.name != "" {
			continue
		}
		if n, ok := canonical[strings.ToLower(p.parsed.Name)]; ok {
			p.name = n
			continue
		}
		// Rare lines take the one-at-a-time lookup: front-face names of
		// double-faced cards, and spellings the database collation matches
		// but a case fold doesn't (e.g. a missing accent).
		name, err := h.resolveCardName(ctx, p.parsed.Name)
		if err != nil {
			p.res.Error = status.Convert(err).Message()
			continue
		}
		p.name = name
	}
	return nil
}

// writeDeckLines adds every resolved line in one transaction, so the import
// lands whole or not at all. Upserts add to an existing row's quantity, so
// repeated lines merge.
func (h *DeckHandler) writeDeckLines(ctx context.Context, deckID uint64, pending []*pendingDeckLine) error {
	hasWrites := false
	for _, p := range pending {
		if p.name != "" {
			hasWrites = true
			break
		}
	}
	if !hasWrites {
		return nil
	}
	err := runInTx(ctx, h.sqlDB, h.queries, func(q *db.Queries) error {
		for _, p := range pending {
			if p.name == "" {
				continue
			}
			qty := p.parsed.Quantity
			if qty < 1 {
				qty = 1
			}
			if _, err := q.UpsertDeckCard(ctx, db.UpsertDeckCardParams{
				DeckID:                  deckID,
				CardName:                p.name,
				PreferredScryfallCardID: p.printing,
				Board:                   db.DeckCardsBoard(p.board),
				Quantity:                qty,
			}); err != nil {
				return err
			}
		}
		return q.TouchDeck(ctx, deckID)
	})
	if err != nil {
		return status.Error(codes.Internal, "failed to import cards")
	}
	return nil
}

// attachImportedCards fills in each added line's stored deck card, fetching
// the whole deck once rather than each card. The cards are already saved, so
// a failure here only costs the echo, reported as a warning.
func (h *DeckHandler) attachImportedCards(ctx context.Context, deckID uint64, pending []*pendingDeckLine) {
	rows, err := h.queries.ListDeckCards(ctx, deckID)
	byKey := make(map[string]*thothv1.DeckCard, len(rows))
	for _, row := range rows {
		byKey[deckCardKey(row.CardName, model.DeckBoard(row.Board))] = model.DeckCardToProto(model.DeckCardFromDB(row))
	}
	for _, p := range pending {
		if p.name == "" {
			continue
		}
		if card, ok := byKey[deckCardKey(p.name, p.board)]; ok && err == nil {
			p.res.Card = card
		} else if p.res.Warning == "" {
			p.res.Warning = "added, but failed to fetch the stored card"
		}
	}
}

// deckCardKey identifies a deck card row, ignoring case as the database does.
func deckCardKey(name string, board model.DeckBoard) string {
	return strings.ToLower(name) + "\x00" + string(board)
}
