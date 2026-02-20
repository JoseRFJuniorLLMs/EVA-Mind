// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Service struct {
	config  *oauth2.Config
	hmacKey []byte
}

// NewService creates OAuth service with Google configuration (FULL access scopes)
func NewService(clientID, clientSecret, redirectURL, hmacSecret string) *Service {
	return &Service{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				// Gmail — full access (read, send, delete, labels, drafts)
				"https://mail.google.com/",
				"https://www.googleapis.com/auth/gmail.readonly",
				// Calendar — full read/write
				"https://www.googleapis.com/auth/calendar",
				// Drive — full access
				"https://www.googleapis.com/auth/drive",
				// Sheets & Docs
				"https://www.googleapis.com/auth/spreadsheets",
				"https://www.googleapis.com/auth/documents",
				// YouTube
				"https://www.googleapis.com/auth/youtube.readonly",
				// Contacts (for email autocomplete)
				"https://www.googleapis.com/auth/contacts.readonly",
				// User identity
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
		hmacKey: []byte(hmacSecret),
	}
}

// SignState creates an HMAC-signed state parameter embedding the CPF.
// Format: base64url(cpf|timestamp.hex_signature)
func (s *Service) SignState(cpf string) string {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	payload := cpf + "|" + ts
	mac := hmac.New(sha256.New, s.hmacKey)
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	raw := payload + "." + sig
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// VerifyState verifies the HMAC signature and extracts the CPF.
// Returns error if signature is invalid or state is older than 10 minutes.
func (s *Service) VerifyState(state string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		return "", fmt.Errorf("invalid state encoding: %w", err)
	}

	parts := strings.SplitN(string(decoded), ".", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid state format")
	}

	payload := parts[0]
	sigHex := parts[1]

	// Verify HMAC
	mac := hmac.New(sha256.New, s.hmacKey)
	mac.Write([]byte(payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sigHex), []byte(expectedSig)) {
		return "", fmt.Errorf("invalid state signature")
	}

	// Extract CPF and timestamp
	payloadParts := strings.SplitN(payload, "|", 2)
	if len(payloadParts) != 2 {
		return "", fmt.Errorf("invalid state payload")
	}

	cpf := payloadParts[0]
	tsStr := payloadParts[1]
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid timestamp in state")
	}

	// Check expiration (10 minutes)
	if time.Now().Unix()-ts > 600 {
		return "", fmt.Errorf("state expired")
	}

	return cpf, nil
}

// GetAuthURL generates the Google OAuth authorization URL
func (s *Service) GetAuthURL(state string) string {
	return s.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// ExchangeCode exchanges authorization code for tokens
func (s *Service) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := s.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	return token, nil
}

// RefreshToken refreshes an expired access token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := s.config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return newToken, nil
}

// GetTokenSource creates a token source for API calls
func (s *Service) GetTokenSource(ctx context.Context, accessToken, refreshToken string, expiry time.Time) oauth2.TokenSource {
	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       expiry,
	}
	return s.config.TokenSource(ctx, token)
}

// GetUserInfo retrieves user email from Google
func (s *Service) GetUserInfo(ctx context.Context, accessToken string) (string, error) {
	client := &http.Client{}
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Email, nil
}
