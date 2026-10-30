package handler

import (
	"context"
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
)

// moxfieldLine matches Moxfield's text export format:
//
//	4 Lightning Bolt (M20) 141
//	1x Teferi, Hero of Dominaria (DOM) 207 *F*
//
// Groups: quantity, name, set code, collector number, foil marker.
var moxfieldLine = regexp.MustCompile(`^(\d+)x?\s+(.+?)\s+\(([A-Za-z0-9]+)\)\s+(\S+)(?:\s+\*([FE])\*)?$`)

// bulkCardLine is one parsed line of a card bulk import.
type bulkCardLine struct {
	Quantity        int32
	Name            string
	SetCode         string // lowercase
	CollectorNumber string
	Foil            model.FoilTreatment
}

func parseMoxfieldLine(line string) (bulkCardLine, error) {
	m := moxfieldLine.FindStringSubmatch(line)
	if m == nil {
		return bulkCardLine{}, fmt.Errorf(`expected "<qty> <name> (<set>) <number>"`)
	}
	qty, err := strconv.Atoi(m[1])
	if err != nil || qty <= 0 {
		return bulkCardLine{}, fmt.Errorf("invalid quantity %q", m[1])
	}
	parsed := bulkCardLine{
		Quantity:        int32(qty),
		Name:            m[2],
		SetCode:         strings.ToLower(m[3]),
		CollectorNumber: m[4],
		Foil:            model.FoilTreatmentNonfoil,
	}
	switch m[5] {
	case "F":
		parsed.Foil = model.FoilTreatmentFoil
	case "E":
		parsed.Foil = model.FoilTreatmentEtched
	}
	return parsed, nil
}

// categoryHeaders maps the section header lines used by deck text formats
// ("Sideboard", "Commander:", ...) to the category of the cards that follow.
var categoryHeaders = map[string]model.CardCategory{
	"commander": model.CardCategoryCommander,
	"deck":      model.CardCategoryDeck,
	"sideboard": model.CardCategorySideboard,
}

// parseCategoryHeader reports whether line is a section header, ignoring case
// and an optional trailing colon.
func parseCategoryHeader(line string) (model.CardCategory, bool) {
	key := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), ":")))
	category, ok := categoryHeaders[key]
	return category, ok
}

// cardNameMatches reports whether the name from an import line refers to the
// Scryfall card, allowing the front-face-only names used for double-faced cards.
func cardNameMatches(lineName, cardName string) bool {
	return strings.EqualFold(lineName, cardName) ||
		strings.HasPrefix(strings.ToLower(cardName), strings.ToLower(lineName)+" // ")
}

// resolveCard finds the exact Scryfall printing for a parsed line. It never
// guesses another printing, but warns when the name doesn't match.
func (h *CardItemHandler) resolveCard(ctx context.Context, parsed bulkCardLine) (db.ScryfallCard, string, error) {
	card, err := h.queries.GetScryfallCardBySetAndNumber(ctx, db.GetScryfallCardBySetAndNumberParams{
		SetCode:         parsed.SetCode,
		CollectorNumber: parsed.CollectorNumber,
	})
	if err != nil {
		return db.ScryfallCard{}, "", fmt.Errorf("%s #%s not found", strings.ToUpper(parsed.SetCode), parsed.CollectorNumber)
	}
	if !cardNameMatches(parsed.Name, card.Name) {
		return card, fmt.Sprintf("%s #%s is %q, not %q", strings.ToUpper(parsed.SetCode), parsed.CollectorNumber, card.Name, parsed.Name), nil
	}
	return card, "", nil
}

func (h *CardItemHandler) BulkImportCardItems(req *thothv1.BulkImportCardItemsRequest, stream grpc.ServerStreamingServer[thothv1.BulkImportCardLineResult]) error {
	ctx := stream.Context()
	if _, err := requireCollectionOwner(ctx, h.queries, uint64(req.CollectionId)); err != nil {
		return err
	}

	lines := strings.Split(req.Text, "\n")

	// Text with section headers is a deck list: cards before the first header
	// are in the deck. Otherwise every card gets the requested category.
	category := model.CardCategoryFromProto(req.Category)
	hasHeaders := false
	cardLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if _, ok := parseCategoryHeader(line); ok {
			hasHeaders = true
			continue
		}
		cardLines++
	}
	if hasHeaders {
		category = model.CardCategoryDeck
	}
	// Checked before importing anything so an oversized list adds nothing.
	if cardLines > maxBulkImportLines {
		return status.Errorf(codes.InvalidArgument, "too many lines (max %d)", maxBulkImportLines)
	}

	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if c, ok := parseCategoryHeader(line); ok {
			category = c
			continue
		}
		res := h.importCardLine(ctx, uint64(req.CollectionId), category, line)
		res.LineNumber = int32(i + 1)
		if err := stream.Send(res); err != nil {
			return err
		}
	}
	return nil
}

// importCardLine adds the card on one bulk import line, reporting any
// problem in the result rather than as an error.
func (h *CardItemHandler) importCardLine(ctx context.Context, collectionID uint64, category model.CardCategory, line string) *thothv1.BulkImportCardLineResult {
	res := &thothv1.BulkImportCardLineResult{Line: line}
	parsed, err := parseMoxfieldLine(line)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Name = parsed.Name
	card, warning, err := h.resolveCard(ctx, parsed)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Warning = warning

	// The upsert adds to an existing row's quantity, so repeated lines merge.
	if _, err := h.queries.UpsertCardItem(ctx, db.UpsertCardItemParams{
		CollectionID:   collectionID,
		ScryfallCardID: card.ID,
		FoilTreatment:  db.CardItemsFoilTreatment(parsed.Foil),
		Category:       db.CardItemsCategory(category),
		Quantity:       parsed.Quantity,
	}); err != nil {
		res.Error = "failed to add card"
		return res
	}
	row, err := h.queries.GetCardItemByKey(ctx, db.GetCardItemByKeyParams{
		CollectionID:   collectionID,
		ScryfallCardID: card.ID,
		FoilTreatment:  db.CardItemsFoilTreatment(parsed.Foil),
		Category:       db.CardItemsCategory(category),
	})
	if err != nil {
		// The card was added; only the echo of the stored row is missing.
		res.Warning = "added, but failed to fetch the stored card"
		return res
	}
	res.Item = model.CardItemToProto(model.CardItemFromDB(db.ListCardItemsByCollectionRow(row)))
	return res
}
