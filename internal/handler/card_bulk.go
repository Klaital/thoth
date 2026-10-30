package handler

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

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

func (h *CardItemHandler) BulkImportCardItems(ctx context.Context, req *thothv1.BulkImportCardItemsRequest) (*thothv1.BulkImportCardItemsResponse, error) {
	if _, err := requireCollectionOwner(ctx, h.queries, uint64(req.CollectionId)); err != nil {
		return nil, err
	}

	lines := strings.Split(req.Text, "\n")

	// Text with section headers is a deck list: cards before the first header
	// are in the deck. Otherwise every card gets the requested category.
	category := model.CardCategoryFromProto(req.Category)
	for _, line := range lines {
		if _, ok := parseCategoryHeader(line); ok {
			category = model.CardCategoryDeck
			break
		}
	}

	type itemKey struct {
		cardID   string
		foil     model.FoilTreatment
		category model.CardCategory
	}
	var results []*thothv1.BulkImportCardLineResult
	resultKeys := map[*thothv1.BulkImportCardLineResult]itemKey{}

	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if c, ok := parseCategoryHeader(line); ok {
			category = c
			continue
		}
		res := &thothv1.BulkImportCardLineResult{LineNumber: int32(i + 1), Line: line}
		results = append(results, res)
		if len(results) > maxBulkImportLines {
			return nil, status.Errorf(codes.InvalidArgument, "too many lines (max %d)", maxBulkImportLines)
		}

		parsed, err := parseMoxfieldLine(line)
		if err != nil {
			res.Error = err.Error()
			continue
		}
		res.Name = parsed.Name
		card, warning, err := h.resolveCard(ctx, parsed)
		if err != nil {
			res.Error = err.Error()
			continue
		}
		res.Warning = warning

		// The upsert adds to an existing row's quantity, so repeated lines merge.
		if _, err := h.queries.UpsertCardItem(ctx, db.UpsertCardItemParams{
			CollectionID:   uint64(req.CollectionId),
			ScryfallCardID: card.ID,
			FoilTreatment:  db.CardItemsFoilTreatment(parsed.Foil),
			Category:       db.CardItemsCategory(category),
			Quantity:       parsed.Quantity,
		}); err != nil {
			res.Error = "failed to add card"
			continue
		}
		resultKeys[res] = itemKey{card.ID, parsed.Foil, category}
	}

	if len(resultKeys) > 0 {
		rows, err := h.queries.ListCardItemsByCollection(ctx, uint64(req.CollectionId))
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to fetch imported cards")
		}
		items := make(map[itemKey]*thothv1.CardItem, len(rows))
		for _, r := range rows {
			items[itemKey{r.ScryfallCardID, model.FoilTreatment(r.FoilTreatment), model.CardCategory(r.Category)}] = model.CardItemToProto(model.CardItemFromDB(r))
		}
		for res, key := range resultKeys {
			res.Item = items[key]
		}
	}

	return &thothv1.BulkImportCardItemsResponse{Results: results}, nil
}
