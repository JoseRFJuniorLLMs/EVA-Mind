// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import (
	"encoding/json"
	"eva-mind/internal/brainstem/database"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service         *Service
	db              *database.DB
	frontendBaseURL string
}

func NewHandler(service *Service, db *database.DB, frontendBaseURL string) *Handler {
	return &Handler{
		service:         service,
		db:              db,
		frontendBaseURL: frontendBaseURL,
	}
}

// HandleAuthorize redirects user to Google OAuth consent screen.
// The CPF is embedded in the HMAC-signed state parameter.
// GET /api/v1/oauth/authorize?cpf=12345678900
func (h *Handler) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	cpf := r.URL.Query().Get("cpf")
	if cpf == "" || len(cpf) < 11 {
		http.Error(w, `{"error":"CPF invalido"}`, http.StatusBadRequest)
		return
	}

	// Verify CPF exists in DB
	_, err := h.db.GetIdosoByCPF(cpf)
	if err != nil {
		log.Warn().Str("cpf", cpf[:3]+"***").Err(err).Msg("[OAUTH] CPF nao encontrado")
		http.Error(w, `{"error":"CPF nao encontrado"}`, http.StatusNotFound)
		return
	}

	// Generate HMAC-signed state with CPF
	state := h.service.SignState(cpf)

	// Set state cookie for double-check on callback
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	authURL := h.service.GetAuthURL(state)
	log.Info().Str("cpf", cpf[:3]+"***").Msg("[OAUTH] Redirecionando para Google consent")
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// HandleCallback processes OAuth callback from Google.
// Extracts CPF from HMAC-signed state, exchanges code for tokens,
// saves to DB, and redirects to frontend.
// GET /api/v1/oauth/callback?code=...&state=...
func (h *Handler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	frontendURL := h.frontendBaseURL + "/eva"

	stateParam := r.URL.Query().Get("state")

	// Verify state cookie matches (CSRF protection)
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != stateParam {
		log.Warn().Msg("[OAUTH] State cookie mismatch — possivel CSRF")
		http.Redirect(w, r, frontendURL+"?google=error&reason=csrf", http.StatusTemporaryRedirect)
		return
	}

	// Verify HMAC and extract CPF from state
	cpf, err := h.service.VerifyState(stateParam)
	if err != nil {
		log.Warn().Err(err).Msg("[OAUTH] State verification failed")
		http.Redirect(w, r, frontendURL+"?google=error&reason=state_invalid", http.StatusTemporaryRedirect)
		return
	}

	// Check if Google returned an error (user denied consent)
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		log.Info().Str("error", errParam).Msg("[OAUTH] User denied consent")
		http.Redirect(w, r, frontendURL+"?google=error&reason="+errParam, http.StatusTemporaryRedirect)
		return
	}

	// Exchange code for tokens
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, frontendURL+"?google=error&reason=no_code", http.StatusTemporaryRedirect)
		return
	}

	token, err := h.service.ExchangeCode(r.Context(), code)
	if err != nil {
		log.Error().Err(err).Msg("[OAUTH] Failed to exchange code")
		http.Redirect(w, r, frontendURL+"?google=error&reason=exchange_failed", http.StatusTemporaryRedirect)
		return
	}

	// Get user email from Google
	email, err := h.service.GetUserInfo(r.Context(), token.AccessToken)
	if err != nil {
		log.Warn().Err(err).Msg("[OAUTH] Failed to get user email (non-fatal)")
	}

	// Lookup idoso by CPF
	idoso, err := h.db.GetIdosoByCPF(cpf)
	if err != nil {
		log.Error().Err(err).Str("cpf", cpf[:3]+"***").Msg("[OAUTH] CPF not found on callback")
		http.Redirect(w, r, frontendURL+"?google=error&reason=cpf_not_found", http.StatusTemporaryRedirect)
		return
	}

	// Save tokens to DB
	err = h.db.SaveGoogleTokens(idoso.ID, token.RefreshToken, token.AccessToken, token.Expiry)
	if err != nil {
		log.Error().Err(err).Int64("idoso", idoso.ID).Msg("[OAUTH] Failed to save tokens")
		http.Redirect(w, r, frontendURL+"?google=error&reason=save_failed", http.StatusTemporaryRedirect)
		return
	}

	// Save Google email
	if email != "" {
		if saveErr := h.db.SaveGoogleEmail(idoso.ID, email); saveErr != nil {
			log.Warn().Err(saveErr).Msg("[OAUTH] Failed to save google email (non-fatal)")
		}
	}

	// Clear the state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})

	log.Info().
		Int64("idoso", idoso.ID).
		Str("email", email).
		Msg("✅ [OAUTH] Google account linked successfully")

	// Redirect to frontend with success
	http.Redirect(w, r, frontendURL+"?google=success", http.StatusTemporaryRedirect)
}

// HandleTokenExchange receives auth code from mobile and saves tokens directly.
// POST /api/v1/oauth/token-exchange  {code, idoso_id}
func (h *Handler) HandleTokenExchange(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code    string `json:"code"`
		IdosoID int64  `json:"idoso_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request"}`, http.StatusBadRequest)
		return
	}

	token, err := h.service.ExchangeCode(r.Context(), req.Code)
	if err != nil {
		log.Error().Err(err).Msg("[OAUTH] Failed to exchange code (mobile)")
		http.Error(w, `{"error":"Failed to exchange code"}`, http.StatusInternalServerError)
		return
	}

	err = h.db.SaveGoogleTokens(req.IdosoID, token.RefreshToken, token.AccessToken, token.Expiry)
	if err != nil {
		log.Error().Err(err).Msg("[OAUTH] Failed to save tokens (mobile)")
		http.Error(w, `{"error":"Failed to save tokens"}`, http.StatusInternalServerError)
		return
	}

	// Save email
	email, _ := h.service.GetUserInfo(r.Context(), token.AccessToken)
	if email != "" {
		h.db.SaveGoogleEmail(req.IdosoID, email)
	}

	log.Info().Int64("idoso", req.IdosoID).Msg("✅ [OAUTH] Tokens saved (mobile)")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Google account linked successfully",
	})
}

// HandleGoogleStatus returns the Google connection status for a given CPF.
// GET /api/v1/idosos/by-cpf/{cpf}/google-status
func (h *Handler) HandleGoogleStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cpf := vars["cpf"]

	if cpf == "" {
		http.Error(w, `{"error":"CPF is required"}`, http.StatusBadRequest)
		return
	}

	status, err := h.db.GetGoogleStatusByCPF(cpf)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"connected": false,
			"email":     "",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// HandleGoogleDisconnect removes Google tokens for a given CPF.
// POST /api/v1/idosos/by-cpf/{cpf}/google-disconnect
func (h *Handler) HandleGoogleDisconnect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cpf := vars["cpf"]

	if cpf == "" {
		http.Error(w, `{"error":"CPF is required"}`, http.StatusBadRequest)
		return
	}

	idoso, err := h.db.GetIdosoByCPF(cpf)
	if err != nil {
		http.Error(w, `{"error":"CPF not found"}`, http.StatusNotFound)
		return
	}

	err = h.db.ClearGoogleTokens(idoso.ID)
	if err != nil {
		log.Error().Err(err).Int64("idoso", idoso.ID).Msg("[OAUTH] Failed to clear tokens")
		http.Error(w, `{"error":"Failed to disconnect"}`, http.StatusInternalServerError)
		return
	}

	log.Info().Int64("idoso", idoso.ID).Msg("[OAUTH] Google account disconnected")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Google account disconnected",
	})
}
