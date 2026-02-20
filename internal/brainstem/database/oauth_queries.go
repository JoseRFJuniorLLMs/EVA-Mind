// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package database

import (
	"database/sql"
	"fmt"
	"time"
)

// SaveGoogleTokens stores OAuth tokens for an idoso
func (db *DB) SaveGoogleTokens(idosoID int64, refreshToken, accessToken string, expiry time.Time) error {
	query := `
		UPDATE idosos
		SET google_refresh_token = $1,
		    google_access_token = $2,
		    google_token_expiry = $3
		WHERE id = $4
	`
	_, err := db.Conn.Exec(query, refreshToken, accessToken, expiry, idosoID)
	if err != nil {
		return fmt.Errorf("failed to save google tokens: %w", err)
	}
	return nil
}

// GetGoogleTokens retrieves OAuth tokens for an idoso
func (db *DB) GetGoogleTokens(idosoID int64) (refreshToken, accessToken string, expiry time.Time, err error) {
	query := `
		SELECT google_refresh_token, google_access_token, google_token_expiry
		FROM idosos
		WHERE id = $1
	`
	var rt, at sql.NullString
	var exp sql.NullTime

	err = db.Conn.QueryRow(query, idosoID).Scan(&rt, &at, &exp)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to get google tokens: %w", err)
	}

	if rt.Valid {
		refreshToken = rt.String
	}
	if at.Valid {
		accessToken = at.String
	}
	if exp.Valid {
		expiry = exp.Time
	}

	return refreshToken, accessToken, expiry, nil
}

// SaveGoogleEmail stores the linked Google account email for an idoso
func (db *DB) SaveGoogleEmail(idosoID int64, email string) error {
	query := `UPDATE idosos SET google_email = $1 WHERE id = $2`
	_, err := db.Conn.Exec(query, email, idosoID)
	if err != nil {
		return fmt.Errorf("failed to save google email: %w", err)
	}
	return nil
}

// GoogleStatus holds the Google connection status for an idoso
type GoogleStatus struct {
	Connected bool   `json:"connected"`
	Email     string `json:"email"`
}

// GetGoogleStatusByCPF checks if a Google account is linked for a given CPF
func (db *DB) GetGoogleStatusByCPF(cpf string) (*GoogleStatus, error) {
	query := `
		SELECT google_refresh_token, google_email
		FROM idosos
		WHERE regexp_replace(cpf, '\D', '', 'g') = regexp_replace($1, '\D', '', 'g')
		AND ativo = true
	`
	var rt, ge sql.NullString
	err := db.Conn.QueryRow(query, cpf).Scan(&rt, &ge)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("CPF not found")
		}
		return nil, fmt.Errorf("failed to get google status: %w", err)
	}

	return &GoogleStatus{
		Connected: rt.Valid && rt.String != "",
		Email:     ge.String,
	}, nil
}

// ClearGoogleTokens removes all Google OAuth data for an idoso
func (db *DB) ClearGoogleTokens(idosoID int64) error {
	query := `
		UPDATE idosos
		SET google_refresh_token = NULL,
		    google_access_token = NULL,
		    google_token_expiry = NULL,
		    google_email = NULL
		WHERE id = $1
	`
	_, err := db.Conn.Exec(query, idosoID)
	if err != nil {
		return fmt.Errorf("failed to clear google tokens: %w", err)
	}
	return nil
}
