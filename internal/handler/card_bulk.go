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
	"github.com/klaital/thoth/internal/telemetry"
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

// boardHeaders maps the section header lines used by deck text formats
// ("Sideboard", "Commander:", ...) to the deck board of the cards that follow.
var boardHeaders = map[string]model.DeckBoard{
	"commander": model.DeckBoardCommander,
	"deck":      model.DeckBoardMain,
	"main":      model.DeckBoardMain,
	"mainboard": model.DeckBoardMain,
	"sideboard": model.DeckBoardSideboard,
}

// parseBoardHeader reports whether line is a section header, ignoring case
// and an optional trailing colon.
func parseBoardHeader(line string) (model.DeckBoard, bool) {
	key := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), ":")))
	board, ok := boardHeaders[key]
	return board, ok
}

// cardNameMatches reports whether the name from an import line refers to the
// Scryfall card, allowing the front-face-only names used for double-faced cards.
func cardNameMatches(lineName, cardName string) bool {
	return strings.EqualFold(lineName, cardName) ||
		strings.HasPrefix(strings.ToLower(cardName), strings.ToLower(lineName)+" // ")
}

// printingRef names a printing by set code and collector number.
type printingRef struct {
	SetCode         string
	CollectorNumber string
}

// printingKey identifies a printing, ignoring case as the database does.
func printingKey(setCode, collectorNumber string) string {
	return strings.ToLower(setCode) + "#" + strings.ToLower(collectorNumber)
}

// lookupPrintings fetches the printings in refs with one query, keyed by
// printingKey. Printings that don't exist are absent from the map.
func lookupPrintings(ctx context.Context, q *db.Queries, refs []printingRef) (map[string]db.ListScryfallCardsBySetsAndNumbersRow, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	var sets, numbers []string
	seenSet, seenNumber := map[string]bool{}, map[string]bool{}
	for _, r := range refs {
		if !seenSet[r.SetCode] {
			seenSet[r.SetCode] = true
			sets = append(sets, r.SetCode)
		}
		if !seenNumber[r.CollectorNumber] {
			seenNumber[r.CollectorNumber] = true
			numbers = append(numbers, r.CollectorNumber)
		}
	}
	// The query matches any set with any number, a superset of the pairs
	// asked for; only the exact pairs are looked up in the result.
	cards, err := q.ListScryfallCardsBySetsAndNumbers(ctx, db.ListScryfallCardsBySetsAndNumbersParams{
		SetCodes:         sets,
		CollectorNumbers: numbers,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to look up cards")
	}
	byKey := make(map[string]db.ListScryfallCardsBySetsAndNumbersRow, len(cards))
	for _, c := range cards {
		key := printingKey(c.SetCode, c.CollectorNumber)
		if _, ok := byKey[key]; !ok {
			byKey[key] = c
		}
	}
	return byKey, nil
}

// pendingCardLine is one card line of a collection bulk import as it moves
// from parsing through lookup to the write. cardID stays empty if the line
// failed.
type pendingCardLine struct {
	parsed bulkCardLine
	cardID string
	res    *thothv1.BulkImportCardLineResult
}

// BulkImportCardItems adds a list of owned cards. Like a deck import, it
// parses every line, looks the printings up in one query, writes in one
// transaction, and sends the results once it commits, so the cost is a
// handful of queries plus one upsert per line.
func (h *CardItemHandler) BulkImportCardItems(req *thothv1.BulkImportCardItemsRequest, stream grpc.ServerStreamingServer[thothv1.BulkImportCardLineResult]) error {
	ctx := stream.Context()
	collectionID := uint64(req.CollectionId)
	if _, err := requireCollectionOwner(ctx, h.queries, collectionID); err != nil {
		return err
	}

	lang, err := normalizeLanguage(req.Language)
	if err != nil {
		return err
	}
	lines := strings.Split(req.Text, "\n")

	// Deck section headers are allowed (a deck export pasted into a
	// collection) but carry no meaning for owned cards, so they're skipped.
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

	pending := make([]*pendingCardLine, 0, cardLines)
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if _, ok := parseBoardHeader(line); ok {
			continue
		}
		p := &pendingCardLine{res: &thothv1.BulkImportCardLineResult{Line: line, LineNumber: int32(i + 1)}}
		parsed, err := parseMoxfieldLine(line)
		if err != nil {
			p.res.Error = err.Error()
		} else {
			p.parsed = parsed
			p.res.Name = parsed.Name
		}
		pending = append(pending, p)
	}

	if err := h.resolveCardLines(ctx, pending); err != nil {
		return err
	}
	if err := h.writeCardLines(ctx, collectionID, lang, pending); err != nil {
		return err
	}
	h.attachImportedItems(ctx, collectionID, lang, pending)

	for _, p := range pending {
		added := p.res.Error == ""
		telemetry.RecordBulkLine(ctx, "card", added)
		if added {
			telemetry.CardsAdded.Add(ctx, int64(p.parsed.Quantity), telemetry.SourceBulk)
		}
		if err := stream.Send(p.res); err != nil {
			return err
		}
	}
	return nil
}

// resolveCardLines finds the exact printing for each parsed line. It never
// guesses another printing: a missing one fails the line, and one whose name
// doesn't match is used with a warning.
func (h *CardItemHandler) resolveCardLines(ctx context.Context, pending []*pendingCardLine) error {
	var refs []printingRef
	for _, p := range pending {
		if p.res.Error == "" {
			refs = append(refs, printingRef{SetCode: p.parsed.SetCode, CollectorNumber: p.parsed.CollectorNumber})
		}
	}
	printings, err := lookupPrintings(ctx, h.queries, refs)
	if err != nil {
		return err
	}
	for _, p := range pending {
		if p.res.Error != "" {
			continue
		}
		label := fmt.Sprintf("%s #%s", strings.ToUpper(p.parsed.SetCode), p.parsed.CollectorNumber)
		card, ok := printings[printingKey(p.parsed.SetCode, p.parsed.CollectorNumber)]
		if !ok {
			p.res.Error = label + " not found"
			continue
		}
		if !cardNameMatches(p.parsed.Name, card.Name) {
			p.res.Warning = fmt.Sprintf("%s is %q, not %q", label, card.Name, p.parsed.Name)
		}
		p.cardID = card.ID
	}
	return nil
}

// writeCardLines adds every resolved line in one transaction, so the import
// lands whole or not at all. The upsert adds to an existing row's quantity,
// so repeated lines merge.
func (h *CardItemHandler) writeCardLines(ctx context.Context, collectionID uint64, lang string, pending []*pendingCardLine) error {
	hasWrites := false
	for _, p := range pending {
		if p.cardID != "" {
			hasWrites = true
			break
		}
	}
	if !hasWrites {
		return nil
	}
	err := runInTx(ctx, h.sqlDB, h.queries, func(q *db.Queries) error {
		for _, p := range pending {
			if p.cardID == "" {
				continue
			}
			if _, err := q.UpsertCardItem(ctx, db.UpsertCardItemParams{
				CollectionID:   collectionID,
				ScryfallCardID: p.cardID,
				FoilTreatment:  db.CollectionCardsFoilTreatment(p.parsed.Foil),
				Language:       lang,
				Quantity:       p.parsed.Quantity,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return status.Error(codes.Internal, "failed to import cards")
	}
	return nil
}

// attachImportedItems fills in each added line's stored row with one query
// for the imported printings. The cards are already saved, so a failure here
// only costs the echo, reported as a warning.
func (h *CardItemHandler) attachImportedItems(ctx context.Context, collectionID uint64, lang string, pending []*pendingCardLine) {
	var ids []string
	seen := map[string]bool{}
	for _, p := range pending {
		if p.cardID != "" && !seen[p.cardID] {
			seen[p.cardID] = true
			ids = append(ids, p.cardID)
		}
	}
	if len(ids) == 0 {
		return
	}
	rows, err := h.queries.ListCardItemsByCollectionAndCards(ctx, db.ListCardItemsByCollectionAndCardsParams{
		CollectionID:    collectionID,
		ScryfallCardIds: ids,
	})
	byKey := make(map[string]*thothv1.CardItem, len(rows))
	for _, row := range rows {
		key := cardItemKey(row.ScryfallCardID, model.FoilTreatment(row.FoilTreatment), row.Language)
		byKey[key] = model.CardItemToProto(model.CardItemFromDB(db.ListCardItemsByCollectionRow(row)))
	}
	for _, p := range pending {
		if p.cardID == "" {
			continue
		}
		if item, ok := byKey[cardItemKey(p.cardID, p.parsed.Foil, lang)]; ok && err == nil {
			p.res.Item = item
		} else if p.res.Warning == "" {
			p.res.Warning = "added, but failed to fetch the stored card"
		}
	}
}

// cardItemKey identifies a collection card row within one collection.
func cardItemKey(scryfallCardID string, foil model.FoilTreatment, lang string) string {
	return scryfallCardID + "\x00" + string(foil) + "\x00" + strings.ToLower(lang)
}
