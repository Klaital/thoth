package handler

import (
	"context"
	"database/sql"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/klaital/thoth/internal/db"
	thothv1 "github.com/klaital/thoth/internal/gen/thoth/v1"
	"github.com/klaital/thoth/internal/goupc"
	"github.com/klaital/thoth/internal/model"
)

type SealedItemHandler struct {
	thothv1.UnimplementedSealedItemServiceServer
	queries    *db.Queries
	upcClient  *goupc.Client
}

func NewSealedItemHandler(queries *db.Queries, upcClient *goupc.Client) *SealedItemHandler {
	return &SealedItemHandler{queries: queries, upcClient: upcClient}
}

func (h *SealedItemHandler) AddSealedItem(ctx context.Context, req *thothv1.AddSealedItemRequest) (*thothv1.AddSealedItemResponse, error) {
	if _, err := requireCollectionOwner(ctx, h.queries, uint64(req.CollectionId)); err != nil {
		return nil, err
	}
	if req.ItemType == thothv1.SealedItemType_SEALED_ITEM_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "item_type is required")
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	result, err := h.queries.CreateSealedItem(ctx, db.CreateSealedItemParams{
		CollectionID: uint64(req.CollectionId),
		ItemType:     db.SealedItemsItemType(model.SealedItemTypeFromProto(req.ItemType)),
		SetCode:      optNullString(req.SetCode),
		Name:         optNullString(req.Name),
		Quantity:     req.Quantity,
		Notes:        nullString(req.Notes),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to add sealed item")
	}
	newID, _ := result.LastInsertId()

	row, err := h.queries.GetSealedItemByID(ctx, uint64(newID))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch new sealed item")
	}
	return &thothv1.AddSealedItemResponse{Item: model.SealedItemToProto(model.SealedItemFromDB(row))}, nil
}

func (h *SealedItemHandler) ListSealedItems(ctx context.Context, req *thothv1.ListSealedItemsRequest) (*thothv1.ListSealedItemsResponse, error) {
	rows, err := h.queries.ListSealedItemsByCollection(ctx, uint64(req.CollectionId))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list sealed items")
	}
	out := make([]*thothv1.SealedItem, len(rows))
	for i, r := range rows {
		out[i] = model.SealedItemToProto(model.SealedItemFromDB(r))
	}
	return &thothv1.ListSealedItemsResponse{Items: out}, nil
}

func (h *SealedItemHandler) UpdateSealedItem(ctx context.Context, req *thothv1.UpdateSealedItemRequest) (*thothv1.UpdateSealedItemResponse, error) {
	row, err := h.queries.GetSealedItemByID(ctx, uint64(req.Id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "sealed item not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch sealed item")
	}
	if _, err := requireCollectionOwner(ctx, h.queries, row.CollectionID); err != nil {
		return nil, err
	}

	if err := h.queries.UpdateSealedItem(ctx, db.UpdateSealedItemParams{
		ItemType: db.SealedItemsItemType(model.SealedItemTypeFromProto(req.ItemType)),
		SetCode:  optNullString(req.SetCode),
		Name:     optNullString(req.Name),
		Quantity: req.Quantity,
		Notes:    nullString(req.Notes),
		ID:       uint64(req.Id),
	}); err != nil {
		return nil, status.Error(codes.Internal, "failed to update sealed item")
	}

	updated, err := h.queries.GetSealedItemByID(ctx, uint64(req.Id))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch updated sealed item")
	}
	return &thothv1.UpdateSealedItemResponse{Item: model.SealedItemToProto(model.SealedItemFromDB(updated))}, nil
}

func (h *SealedItemHandler) RemoveSealedItem(ctx context.Context, req *thothv1.RemoveSealedItemRequest) (*thothv1.RemoveSealedItemResponse, error) {
	row, err := h.queries.GetSealedItemByID(ctx, uint64(req.Id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "sealed item not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch sealed item")
	}
	if _, err := requireCollectionOwner(ctx, h.queries, row.CollectionID); err != nil {
		return nil, err
	}
	if err := h.queries.DeleteSealedItem(ctx, uint64(req.Id)); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete sealed item")
	}
	return &thothv1.RemoveSealedItemResponse{}, nil
}

func (h *SealedItemHandler) LookupUPC(ctx context.Context, req *thothv1.LookupUPCRequest) (*thothv1.LookupUPCResponse, error) {
	if req.Upc == "" {
		return nil, status.Error(codes.InvalidArgument, "upc is required")
	}
	if h.upcClient == nil {
		return nil, status.Error(codes.Unimplemented, "UPC lookup not configured")
	}

	// Check the local cache first.
	if cached, err := h.queries.GetUPCCache(ctx, req.Upc); err == nil {
		return upcCacheToProto(cached), nil
	}

	// Cache miss: call go-upc.com.
	resp, err := h.upcClient.Lookup(ctx, req.Upc)
	if err != nil {
		if errors.Is(err, goupc.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "barcode not found")
		}
		return nil, status.Error(codes.Internal, "UPC lookup failed")
	}

	parsed := goupc.ParseMTGProduct(resp.Product.Name, resp.Product.Description)

	// Try to resolve the human-readable set name to a Scryfall set code.
	// Falls back to prepending "The " because go-upc.com often drops it
	// (e.g. "Hobbit" → Scryfall "The Hobbit").
	var setCode string
	if parsed.SetName != "" {
		if code, err := h.queries.LookupSetCodeByName(ctx, parsed.SetName); err == nil {
			setCode = code
		} else if code, err := h.queries.LookupSetCodeByName(ctx, "The "+parsed.SetName); err == nil {
			setCode = code
		}
	}

	// Persist to cache regardless of whether we recognized the product type.
	cacheParams := db.UpsertUPCCacheParams{
		Upc:            req.Upc,
		RawName:        sql.NullString{String: resp.Product.Name, Valid: resp.Product.Name != ""},
		RawDescription: sql.NullString{String: resp.Product.Description, Valid: resp.Product.Description != ""},
		ParsedName:     sql.NullString{String: parsed.Name, Valid: parsed.Name != ""},
		ParsedSetName:  sql.NullString{String: parsed.SetName, Valid: parsed.SetName != ""},
		SetCode:        sql.NullString{String: setCode, Valid: setCode != ""},
	}
	if parsed.ItemType != "" {
		cacheParams.ItemType = db.NullUpcCacheItemType{
			UpcCacheItemType: db.UpcCacheItemType(parsed.ItemType),
			Valid:            true,
		}
	}
	_ = h.queries.UpsertUPCCache(ctx, cacheParams) // cache write failure is non-fatal

	return &thothv1.LookupUPCResponse{
		SetName:  parsed.SetName,
		SetCode:  setCode,
		ItemType: model.SealedItemTypeToProto(model.SealedItemType(parsed.ItemType)),
	}, nil
}

func upcCacheToProto(row db.UpcCache) *thothv1.LookupUPCResponse {
	resp := &thothv1.LookupUPCResponse{}
	if row.ParsedSetName.Valid {
		resp.SetName = row.ParsedSetName.String
	}
	if row.SetCode.Valid {
		resp.SetCode = row.SetCode.String
	}
	if row.ItemType.Valid {
		resp.ItemType = model.SealedItemTypeToProto(model.SealedItemType(row.ItemType.UpcCacheItemType))
	}
	return resp
}
