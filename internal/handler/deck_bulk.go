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

func (h *DeckHandler) BulkImportDeckCards(req *thothv1.BulkImportDeckCardsRequest, stream grpc.ServerStreamingServer[thothv1.BulkImportDeckCardLineResult]) error {
	ctx := stream.Context()
	if _, err := h.ownedDeck(ctx, uint64(req.DeckId)); err != nil {
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
		res := h.importDeckLine(ctx, uint64(req.DeckId), board, line)
		res.LineNumber = int32(i + 1)
		telemetry.RecordBulkLine(ctx, "deck", res.Error == "")
		if err := stream.Send(res); err != nil {
			return err
		}
	}
	return nil
}

// importDeckLine adds the card on one line, reporting problems in the result.
// A printing that can't be found doesn't fail the line: the card is added by
// name with a warning.
func (h *DeckHandler) importDeckLine(ctx context.Context, deckID uint64, board model.DeckBoard, line string) *thothv1.BulkImportDeckCardLineResult {
	res := &thothv1.BulkImportDeckCardLineResult{Line: line}
	parsed, err := parseDeckLine(line)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Name = parsed.Name

	var name string
	var printing sql.NullString
	if parsed.SetCode != "" {
		card, err := h.queries.GetScryfallCardBySetAndNumber(ctx, db.GetScryfallCardBySetAndNumberParams{
			SetCode:         parsed.SetCode,
			CollectorNumber: parsed.CollectorNumber,
		})
		switch {
		case err != nil:
			res.Warning = fmt.Sprintf("%s #%s not found; added by name", strings.ToUpper(parsed.SetCode), parsed.CollectorNumber)
		case !cardNameMatches(parsed.Name, card.Name):
			res.Warning = fmt.Sprintf("%s #%s is %q, not %q; added by name", strings.ToUpper(parsed.SetCode), parsed.CollectorNumber, card.Name, parsed.Name)
		default:
			name = card.Name
			printing = nullString(card.ID)
		}
	}
	if name == "" {
		if name, err = h.resolveCardName(ctx, parsed.Name); err != nil {
			res.Error = status.Convert(err).Message()
			return res
		}
	}

	card, err := h.addCard(ctx, deckID, name, board, parsed.Quantity, printing)
	if err != nil {
		res.Error = "failed to add card"
		return res
	}
	res.Card = card
	return res
}
