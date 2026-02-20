// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package gmail

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/rs/zerolog/log"
)

// NotifyFunc sends a tool_event to the browser WebSocket session
type NotifyFunc func(idosoID int64, msgType string, payload interface{})

// TokenFunc retrieves a valid Google access token for an idoso
type TokenFunc func(idosoID int64) (string, error)

// Watcher polls Gmail for new unread emails and notifies via WebSocket
type Watcher struct {
	interval time.Duration
	getToken TokenFunc
	notify   NotifyFunc

	mu       sync.Mutex
	watching map[int64]context.CancelFunc // active watchers per idoso
	lastSeen map[int64]string             // last seen message ID per idoso
}

// NewWatcher creates a Gmail watcher with the given poll interval
func NewWatcher(interval time.Duration, getToken TokenFunc, notify NotifyFunc) *Watcher {
	return &Watcher{
		interval: interval,
		getToken: getToken,
		notify:   notify,
		watching: make(map[int64]context.CancelFunc),
		lastSeen: make(map[int64]string),
	}
}

// StartWatching begins polling Gmail for a specific idoso (called when WS session starts)
func (w *Watcher) StartWatching(idosoID int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Already watching
	if _, exists := w.watching[idosoID]; exists {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	w.watching[idosoID] = cancel

	go w.pollLoop(ctx, idosoID)
	log.Info().Int64("idoso", idosoID).Dur("interval", w.interval).Msg("[GMAIL-WATCH] Iniciado")
}

// StopWatching stops polling for a specific idoso (called when WS session ends)
func (w *Watcher) StopWatching(idosoID int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if cancel, exists := w.watching[idosoID]; exists {
		cancel()
		delete(w.watching, idosoID)
		delete(w.lastSeen, idosoID)
		log.Info().Int64("idoso", idosoID).Msg("[GMAIL-WATCH] Parado")
	}
}

func (w *Watcher) pollLoop(ctx context.Context, idosoID int64) {
	// First check immediately
	w.checkNewEmails(ctx, idosoID)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkNewEmails(ctx, idosoID)
		}
	}
}

func (w *Watcher) checkNewEmails(ctx context.Context, idosoID int64) {
	accessToken, err := w.getToken(idosoID)
	if err != nil {
		// No Google token — skip silently
		return
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	srv, err := gmail.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		log.Warn().Err(err).Int64("idoso", idosoID).Msg("[GMAIL-WATCH] Erro ao criar client")
		return
	}

	// Fetch latest unread emails (max 5)
	resp, err := srv.Users.Messages.List("me").
		Q("is:unread newer_than:3m").
		MaxResults(5).
		Do()
	if err != nil {
		log.Warn().Err(err).Int64("idoso", idosoID).Msg("[GMAIL-WATCH] Erro ao listar emails")
		return
	}

	if len(resp.Messages) == 0 {
		return
	}

	// Check if we already notified about the newest message
	newestID := resp.Messages[0].Id
	w.mu.Lock()
	lastID := w.lastSeen[idosoID]
	w.mu.Unlock()

	if newestID == lastID {
		return // Already notified
	}

	// Get details of new emails
	var newEmails []map[string]interface{}
	for _, msg := range resp.Messages {
		if msg.Id == lastID {
			break // Stop at last seen
		}

		detail, err := srv.Users.Messages.Get("me", msg.Id).
			Format("metadata").
			MetadataHeaders("From", "Subject", "Date").
			Do()
		if err != nil {
			continue
		}

		from, subject, date := "", "", ""
		for _, h := range detail.Payload.Headers {
			switch h.Name {
			case "From":
				from = h.Value
			case "Subject":
				subject = h.Value
			case "Date":
				date = h.Value
			}
		}

		newEmails = append(newEmails, map[string]interface{}{
			"id":      msg.Id,
			"from":    from,
			"subject": subject,
			"date":    date,
			"snippet": detail.Snippet,
		})
	}

	if len(newEmails) == 0 {
		return
	}

	// Update last seen
	w.mu.Lock()
	w.lastSeen[idosoID] = newestID
	w.mu.Unlock()

	// Notify via WebSocket
	w.notify(idosoID, "new_email", map[string]interface{}{
		"count":  len(newEmails),
		"emails": newEmails,
		"message": fmt.Sprintf("%d email(s) novo(s)", len(newEmails)),
	})

	log.Info().
		Int64("idoso", idosoID).
		Int("count", len(newEmails)).
		Msg("[GMAIL-WATCH] Emails novos notificados")
}
