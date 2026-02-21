// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Service struct {
	ctx           context.Context
	accessToken   string
	phoneNumberID string
}

func NewService(ctx context.Context, accessToken, phoneNumberID string) *Service {
	return &Service{
		ctx:           ctx,
		accessToken:   accessToken,
		phoneNumberID: phoneNumberID,
	}
}

// SendMessage sends a WhatsApp message
func (s *Service) SendMessage(to, message string) error {
	url := fmt.Sprintf("https://graph.facebook.com/v18.0/%s/messages", s.phoneNumberID)

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text": map[string]string{
			"body": message,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("send failed with status: %d", resp.StatusCode)
	}

	return nil
}

// SendTemplateMessage sends a pre-approved template message
func (s *Service) SendTemplateMessage(to, templateName string, params []string) error {
	url := fmt.Sprintf("https://graph.facebook.com/v18.0/%s/messages", s.phoneNumberID)

	components := []map[string]interface{}{
		{
			"type": "body",
			"parameters": func() []map[string]string {
				var p []map[string]string
				for _, param := range params {
					p = append(p, map[string]string{"type": "text", "text": param})
				}
				return p
			}(),
		},
	}

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "template",
		"template": map[string]interface{}{
			"name":       templateName,
			"language":   map[string]string{"code": "pt_BR"},
			"components": components,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal template payload: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create template request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send template failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("send template failed with status: %d", resp.StatusCode)
	}

	return nil
}
