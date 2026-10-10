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

type UserHandler struct {
	thothv1.UnimplementedUserServiceServer
	queries *db.Queries
	jwt     *auth.JWTService
}

func NewUserHandler(queries *db.Queries, jwt *auth.JWTService) *UserHandler {
	return &UserHandler{queries: queries, jwt: jwt}
}

func (h *UserHandler) Register(ctx context.Context, req *thothv1.RegisterRequest) (*thothv1.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	result, err := h.queries.CreateUser(ctx, db.CreateUserParams{
		Email:        req.Email,
		PasswordHash: sql.NullString{String: hash, Valid: true},
	})
	if err != nil {
		// Duplicate email will surface as a MySQL error 1062.
		if isDuplicateKeyError(err) {
			return nil, status.Error(codes.AlreadyExists, "email already registered")
		}
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	newID, err := result.LastInsertId()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve new user id")
	}

	dbUser, err := h.queries.GetUserByID(ctx, uint64(newID))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch new user")
	}

	tokens, err := h.jwt.IssueTokenPair(int64(dbUser.ID), dbUser.Email)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to issue tokens")
	}

	return &thothv1.RegisterResponse{
		User:         model.UserToProto(model.UserFromDB(dbUser)),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (h *UserHandler) Login(ctx context.Context, req *thothv1.LoginRequest) (*thothv1.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	dbUser, err := h.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.Unauthenticated, "invalid email or password")
		}
		return nil, status.Error(codes.Internal, "failed to look up user")
	}

	if !dbUser.PasswordHash.Valid {
		// Account exists but was created via OAuth — no password set.
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}

	if err := auth.CheckPassword(dbUser.PasswordHash.String, req.Password); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}

	tokens, err := h.jwt.IssueTokenPair(int64(dbUser.ID), dbUser.Email)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to issue tokens")
	}

	return &thothv1.LoginResponse{
		User:         model.UserToProto(model.UserFromDB(dbUser)),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (h *UserHandler) RefreshToken(ctx context.Context, req *thothv1.RefreshTokenRequest) (*thothv1.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	claims, err := h.jwt.Validate(req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid or expired refresh token")
	}

	tokens, err := h.jwt.IssueTokenPair(claims.UserID, claims.Email)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to issue tokens")
	}

	return &thothv1.RefreshTokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (h *UserHandler) GetMe(ctx context.Context, _ *thothv1.GetMeRequest) (*thothv1.GetMeResponse, error) {
	userID, err := auth.UserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}

	dbUser, err := h.queries.GetUserByID(ctx, uint64(userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch user")
	}

	return &thothv1.GetMeResponse{
		User: model.UserToProto(model.UserFromDB(dbUser)),
	}, nil
}

func isDuplicateKeyError(err error) bool {
	// MySQL error 1062: Duplicate entry
	return err != nil && strings.Contains(err.Error(), "1062")
}
