package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/klaital/thoth/internal/auth"
	"github.com/klaital/thoth/internal/db"
	"github.com/klaital/thoth/internal/model"
)

const stateCookie = "oauth_state"

type OAuthHandlers struct {
	queries     *db.Queries
	jwt         *auth.JWTService
	google      *auth.GoogleOAuth
	frontendURL string
}

func NewOAuthHandlers(queries *db.Queries, jwt *auth.JWTService, google *auth.GoogleOAuth, frontendURL string) *OAuthHandlers {
	return &OAuthHandlers{queries: queries, jwt: jwt, google: google, frontendURL: frontendURL}
}

func (h *OAuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	state, err := randomState()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, h.google.AuthURL(state), http.StatusTemporaryRedirect)
}

func (h *OAuthHandlers) Callback(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(stateCookie)
	if err != nil || cookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: stateCookie, Path: "/", MaxAge: -1})

	info, err := h.google.ExchangeCode(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("google oauth exchange: %v", err)
		http.Error(w, "oauth failed", http.StatusBadGateway)
		return
	}

	user, err := h.findOrCreateGoogleUser(r.Context(), info)
	if err != nil {
		log.Printf("find/create google user: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tokens, err := h.jwt.IssueTokenPair(user.ID, user.Email)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	dest := fmt.Sprintf("%s/auth/callback?access_token=%s&refresh_token=%s",
		h.frontendURL, tokens.AccessToken, tokens.RefreshToken)
	http.Redirect(w, r, dest, http.StatusTemporaryRedirect)
}

func (h *OAuthHandlers) findOrCreateGoogleUser(ctx context.Context, info *auth.GoogleUserInfo) (model.User, error) {
	// 1. Try to find by google_id (returning user on subsequent logins).
	dbUser, err := h.queries.GetUserByGoogleID(ctx, sql.NullString{String: info.ID, Valid: true})
	if err == nil {
		return model.UserFromDB(dbUser), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return model.User{}, fmt.Errorf("looking up by google_id: %w", err)
	}

	// 2. Upsert by email: inserts a new user or links google_id onto an existing email account.
	_, err = h.queries.UpsertGoogleUser(ctx, db.UpsertGoogleUserParams{
		Email:    info.Email,
		GoogleID: sql.NullString{String: info.ID, Valid: true},
	})
	if err != nil {
		return model.User{}, fmt.Errorf("upserting google user: %w", err)
	}

	dbUser, err = h.queries.GetUserByEmail(ctx, info.Email)
	if err != nil {
		return model.User{}, fmt.Errorf("fetching upserted user: %w", err)
	}
	return model.UserFromDB(dbUser), nil
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
