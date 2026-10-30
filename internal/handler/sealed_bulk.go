package handler

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/klaital/thoth/internal/db"
	thothv1 "github.com/klaital/thoth/internal/gen/thoth/v1"
	"github.com/klaital/thoth/internal/model"
)

// maxBulkImportLines caps a single import so one request can't issue an
// unbounded number of inserts.
const maxBulkImportLines = 500

// bulkSealedLine is one parsed line of a bulk import.
type bulkSealedLine struct {
	ItemType model.SealedItemType
	SetName  string // set booster/box lines only
	Name     string // secret lair lines only
}

// bulkBoosterRules maps the product keyword in a bulk import line to the
// box and single-pack item types it implies.
var bulkBoosterRules = []struct {
	keyword string
	box     model.SealedItemType
	pack    model.SealedItemType
}{
	{"collector booster", model.SealedItemTypeCollectorBoosterBox, model.SealedItemTypeCollectorBooster},
	{"play booster", model.SealedItemTypePlayBoosterBox, model.SealedItemTypePlayBooster},
}

// parseBulkSealedLine interprets one line of a bulk import:
//
//	"<Set Name> Play Booster Display"      → play booster box
//	"<Set Name> Collector Booster Display" → collector booster box
//	"<Set Name> Play Booster Pack"         → single play booster
//	anything else                          → secret lair named by the whole line
func parseBulkSealedLine(line string) bulkSealedLine {
	lower := strings.ToLower(line)
	for _, rule := range bulkBoosterRules {
		idx := strings.Index(lower, rule.keyword)
		if idx == -1 {
			continue
		}
		suffix := lower[idx+len(rule.keyword):]
		itemType := rule.box
		if strings.Contains(suffix, "pack") {
			itemType = rule.pack
		}
		return bulkSealedLine{
			ItemType: itemType,
			SetName:  strings.TrimRight(strings.TrimSpace(line[:idx]), " -|:"),
		}
	}
	return bulkSealedLine{ItemType: model.SealedItemTypeSecretLair, Name: line}
}

// resolveSetCode maps a human-readable set name to a Scryfall set code,
// retrying with a "The " prefix since product listings often drop it.
func (h *SealedItemHandler) resolveSetCode(ctx context.Context, setName string) (string, bool) {
	if setName == "" {
		return "", false
	}
	if code, err := h.queries.LookupSetCodeByName(ctx, setName); err == nil {
		return code, true
	}
	if code, err := h.queries.LookupSetCodeByName(ctx, "The "+setName); err == nil {
		return code, true
	}
	return "", false
}

func (h *SealedItemHandler) BulkImportSealedItems(ctx context.Context, req *thothv1.BulkImportSealedItemsRequest) (*thothv1.BulkImportSealedItemsResponse, error) {
	if _, err := requireCollectionOwner(ctx, h.queries, uint64(req.CollectionId)); err != nil {
		return nil, err
	}

	type pending struct {
		params  db.CreateSealedItemParams
		results []*thothv1.BulkImportLineResult // every line that maps to this item
	}
	var order []string
	byLine := map[string]*pending{}
	var results []*thothv1.BulkImportLineResult

	for i, raw := range strings.Split(req.Text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		res := &thothv1.BulkImportLineResult{LineNumber: int32(i + 1), Line: line}
		results = append(results, res)
		if len(results) > maxBulkImportLines {
			return nil, status.Errorf(codes.InvalidArgument, "too many lines (max %d)", maxBulkImportLines)
		}

		// Repeated lines become one item with a higher quantity.
		key := strings.ToLower(line)
		if p, ok := byLine[key]; ok {
			p.params.Quantity++
			p.results = append(p.results, res)
			res.Warning = p.results[0].Warning
			continue
		}

		parsed := parseBulkSealedLine(line)
		params := db.CreateSealedItemParams{
			CollectionID: uint64(req.CollectionId),
			ItemType:     db.SealedItemsItemType(parsed.ItemType),
			Quantity:     1,
		}
		if parsed.ItemType == model.SealedItemTypeSecretLair {
			params.Name = nullString(parsed.Name)
		} else if code, ok := h.resolveSetCode(ctx, parsed.SetName); ok {
			params.SetCode = nullString(code)
		} else {
			// Keep the set name so the item is still identifiable.
			params.Name = nullString(parsed.SetName)
			res.Warning = fmt.Sprintf("unrecognized set %q; saved without a set code", parsed.SetName)
		}

		byLine[key] = &pending{params: params, results: []*thothv1.BulkImportLineResult{res}}
		order = append(order, key)
	}

	for _, key := range order {
		p := byLine[key]
		item, err := h.createSealedItem(ctx, p.params)
		for _, res := range p.results {
			if err != nil {
				res.Error = "failed to add item"
			} else {
				res.Item = item
			}
		}
	}

	return &thothv1.BulkImportSealedItemsResponse{Results: results}, nil
}

func (h *SealedItemHandler) createSealedItem(ctx context.Context, params db.CreateSealedItemParams) (*thothv1.SealedItem, error) {
	result, err := h.queries.CreateSealedItem(ctx, params)
	if err != nil {
		return nil, err
	}
	newID, _ := result.LastInsertId()
	row, err := h.queries.GetSealedItemByID(ctx, uint64(newID))
	if err != nil {
		return nil, err
	}
	return model.SealedItemToProto(model.SealedItemFromDB(row)), nil
}
