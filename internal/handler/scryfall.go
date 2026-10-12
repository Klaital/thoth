package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/klaital/thoth/internal/db"
	thothv1 "github.com/klaital/thoth/internal/gen/thoth/v1"
	"github.com/klaital/thoth/internal/model"
)

type ScryfallHandler struct {
	thothv1.UnimplementedScryfallServiceServer
	queries *db.Queries
}

func NewScryfallHandler(queries *db.Queries) *ScryfallHandler {
	return &ScryfallHandler{queries: queries}
}

func (h *ScryfallHandler) SearchCards(ctx context.Context, req *thothv1.SearchCardsRequest) (*thothv1.SearchCardsResponse, error) {
	if req.Query == "" {
		return &thothv1.SearchCardsResponse{}, nil
	}
	rows, err := h.queries.SearchScryfallCards(ctx, req.Query)
	if err != nil {
		return nil, status.Error(codes.Internal, "search failed")
	}
	out := make([]*thothv1.ScryfallCard, len(rows))
	for i, r := range rows {
		out[i] = model.ScryfallCardToProto(r)
	}
	return &thothv1.SearchCardsResponse{Cards: out}, nil
}
