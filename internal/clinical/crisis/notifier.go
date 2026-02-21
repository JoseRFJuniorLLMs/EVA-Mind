// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package crisis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"eva-mind/internal/motor/email"
	"eva-mind/internal/motor/sms"

	"github.com/rs/zerolog/log"
)

// Notifier handles multi-channel notifications for crisis events
type Notifier struct {
	webhookURL string
	emailSvc   *email.EmailService
	smsSvc     *sms.TwilioService
	// Fallback: direct API URLs if services not injected
	emailAPI string
	smsAPI   string
	// Emergency contact numbers/emails
	emergencyEmails []string
	emergencyPhones []string
}

// NewNotifier creates a new notifier with real email/SMS services
func NewNotifier(emailSvc *email.EmailService, smsSvc *sms.TwilioService) *Notifier {
	emergencyEmails := []string{}
	if e := os.Getenv("EMERGENCY_TEAM_EMAILS"); e != "" {
		for _, addr := range splitAndTrim(e) {
			emergencyEmails = append(emergencyEmails, addr)
		}
	}

	emergencyPhones := []string{}
	if p := os.Getenv("EMERGENCY_TEAM_PHONES"); p != "" {
		for _, phone := range splitAndTrim(p) {
			emergencyPhones = append(emergencyPhones, phone)
		}
	}

	return &Notifier{
		webhookURL:      os.Getenv("CRISIS_WEBHOOK_URL"),
		emailSvc:        emailSvc,
		smsSvc:          smsSvc,
		emailAPI:        os.Getenv("EMAIL_API_URL"),
		smsAPI:          os.Getenv("SMS_API_URL"),
		emergencyEmails: emergencyEmails,
		emergencyPhones: emergencyPhones,
	}
}

// splitAndTrim splits a comma-separated string and trims whitespace
func splitAndTrim(s string) []string {
	var result []string
	for _, part := range bytes.Split([]byte(s), []byte(",")) {
		trimmed := bytes.TrimSpace(part)
		if len(trimmed) > 0 {
			result = append(result, string(trimmed))
		}
	}
	return result
}

// NotificationPayload represents notification data
type NotificationPayload struct {
	EventID          int64     `json:"event_id"`
	PatientID        int64     `json:"patient_id"`
	CrisisType       string    `json:"crisis_type"`
	Severity         string    `json:"severity"`
	TriggerStatement string    `json:"trigger_statement"`
	Timestamp        time.Time `json:"timestamp"`
	RequiresAction   bool      `json:"requires_action"`
}

// NotifyPsychologist sends notification to psychologist
func (n *Notifier) NotifyPsychologist(ctx context.Context, event *CrisisEvent) error {
	payload := NotificationPayload{
		EventID:          event.ID,
		PatientID:        event.PatientID,
		CrisisType:       string(event.CrisisType),
		Severity:         event.Severity,
		TriggerStatement: event.TriggerStatement,
		Timestamp:        event.CreatedAt,
		RequiresAction:   event.ResponseActions["require_acknowledgment"],
	}

	var errors []string

	// 1. Send webhook (real-time)
	if n.webhookURL != "" {
		if err := n.sendWebhook(ctx, payload); err != nil {
			log.Error().Err(err).Msg("Failed to send webhook")
			errors = append(errors, "webhook: "+err.Error())
		}
	}

	// 2. Send email (backup)
	if n.emailAPI != "" || n.emailSvc != nil {
		if err := n.sendEmail(ctx, payload); err != nil {
			log.Error().Err(err).Msg("Failed to send email")
			errors = append(errors, "email: "+err.Error())
		}
	}

	// 3. Send SMS (critical only)
	if event.Severity == "CRITICAL" && (n.smsAPI != "" || n.smsSvc != nil) {
		if err := n.sendSMS(ctx, payload); err != nil {
			log.Error().Err(err).Msg("Failed to send SMS")
			errors = append(errors, "sms: "+err.Error())
		}
	}

	// Se NENHUM canal conseguiu enviar, retorna erro
	channelsAttempted := 0
	if n.webhookURL != "" {
		channelsAttempted++
	}
	if n.emailAPI != "" || n.emailSvc != nil {
		channelsAttempted++
	}
	if event.Severity == "CRITICAL" && (n.smsAPI != "" || n.smsSvc != nil) {
		channelsAttempted++
	}

	if channelsAttempted > 0 && len(errors) == channelsAttempted {
		return fmt.Errorf("CRITICO: todos os %d canais de notificacao falharam para evento %d: %v", channelsAttempted, event.ID, errors)
	}

	return nil
}

// sendWebhook sends webhook notification
func (n *Notifier) sendWebhook(ctx context.Context, payload NotificationPayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", n.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Crisis-Alert", "true")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	log.Info().
		Int64("event_id", payload.EventID).
		Str("severity", payload.Severity).
		Msg("Webhook notification sent")

	return nil
}

// sendEmail sends email notification using the real email service
func (n *Notifier) sendEmail(ctx context.Context, payload NotificationPayload) error {
	subject := fmt.Sprintf("ALERTA DE CRISE [%s] - Paciente %d", payload.Severity, payload.PatientID)
	reason := fmt.Sprintf("Crise %s detectada: %s", payload.CrisisType, payload.TriggerStatement)

	if n.emailSvc != nil {
		// Use real email service
		for _, addr := range n.emergencyEmails {
			err := n.emailSvc.SendEmergencyAlert(addr, "Equipe EVA-Mind", fmt.Sprintf("Paciente %d", payload.PatientID), reason)
			if err != nil {
				log.Error().Err(err).Str("to", addr).Msg("Falha ao enviar email de crise")
			} else {
				log.Info().Str("to", addr).Int64("event_id", payload.EventID).Msg("Email de crise enviado")
			}
		}
		return nil
	}

	// Fallback: webhook-based email API
	if n.emailAPI != "" {
		emailPayload := map[string]interface{}{
			"to":      n.emergencyEmails,
			"subject": subject,
			"payload": payload,
		}
		jsonData, err := json.Marshal(emailPayload)
		if err != nil {
			return fmt.Errorf("marshal email payload: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, "POST", n.emailAPI, bytes.NewBuffer(jsonData))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		log.Info().Int64("event_id", payload.EventID).Msg("Email de crise enviado via API")
	}

	return nil
}

// sendSMS sends SMS notification using the real Twilio service
func (n *Notifier) sendSMS(ctx context.Context, payload NotificationPayload) error {
	reason := fmt.Sprintf("CRISE %s (%s) - Paciente %d - Acao imediata necessaria",
		payload.CrisisType, payload.Severity, payload.PatientID)

	if n.smsSvc != nil {
		// Use real SMS service
		results := n.smsSvc.SendBulkEmergencyAlert(n.emergencyPhones, fmt.Sprintf("Paciente %d", payload.PatientID), reason)
		for _, r := range results {
			if r.Success {
				log.Info().Str("message_id", r.MessageID).Int64("event_id", payload.EventID).Msg("SMS de crise enviado")
			} else {
				log.Error().Str("error", r.Error).Int64("event_id", payload.EventID).Msg("Falha ao enviar SMS de crise")
			}
		}
		return nil
	}

	// Fallback: webhook-based SMS API
	if n.smsAPI != "" {
		smsPayload := map[string]interface{}{
			"phones":  n.emergencyPhones,
			"message": reason,
			"payload": payload,
		}
		jsonData, err := json.Marshal(smsPayload)
		if err != nil {
			return fmt.Errorf("marshal sms payload: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, "POST", n.smsAPI, bytes.NewBuffer(jsonData))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		log.Info().Int64("event_id", payload.EventID).Msg("SMS de crise enviado via API")
	}

	return nil
}

// NotifyEmergencyServices sends notifications to emergency services (SAMU/Police)
func (n *Notifier) NotifyEmergencyServices(ctx context.Context, event *CrisisEvent) error {
	reason := fmt.Sprintf("Crise %s detectada (severidade: %s) - Paciente %d",
		event.CrisisType, event.Severity, event.PatientID)

	var lastErr error

	// 1. SMS para telefones de emergencia configurados
	if n.smsSvc != nil && len(n.emergencyPhones) > 0 {
		results := n.smsSvc.SendBulkEmergencyAlert(n.emergencyPhones, fmt.Sprintf("Paciente %d", event.PatientID), reason)
		for _, r := range results {
			if !r.Success {
				lastErr = fmt.Errorf("SMS emergencia falhou: %s", r.Error)
				log.Error().Str("error", r.Error).Msg("Falha SMS servicos emergencia")
			}
		}
	}

	// 2. Email para equipe de emergencia
	if n.emailSvc != nil && len(n.emergencyEmails) > 0 {
		for _, addr := range n.emergencyEmails {
			err := n.emailSvc.SendEmergencyAlert(addr, "Servicos de Emergencia", fmt.Sprintf("Paciente %d", event.PatientID), reason)
			if err != nil {
				lastErr = err
				log.Error().Err(err).Msg("Falha email servicos emergencia")
			}
		}
	}

	// 3. Webhook para dashboard de emergencia
	if n.webhookURL != "" {
		payload := NotificationPayload{
			EventID:          event.ID,
			PatientID:        event.PatientID,
			CrisisType:       string(event.CrisisType),
			Severity:         event.Severity,
			TriggerStatement: event.TriggerStatement,
			Timestamp:        event.CreatedAt,
			RequiresAction:   true,
		}
		if err := n.sendWebhook(ctx, payload); err != nil {
			lastErr = err
			log.Error().Err(err).Msg("Falha webhook servicos emergencia")
		}
	}

	if lastErr != nil {
		return fmt.Errorf("algumas notificacoes de emergencia falharam: %w", lastErr)
	}

	log.Warn().Int64("patient_id", event.PatientID).Str("crisis_type", string(event.CrisisType)).
		Msg("Notificacoes de emergencia enviadas")
	return nil
}

// NotifyAuthorities sends notifications to authorities (child protective services, etc.)
func (n *Notifier) NotifyAuthorities(ctx context.Context, event *CrisisEvent) error {
	reason := fmt.Sprintf("ALERTA AUTORIDADES: %s detectado (severidade: %s) - Paciente %d - Declaracao: %q",
		event.CrisisType, event.Severity, event.PatientID, event.TriggerStatement)

	var lastErr error

	// 1. SMS para telefones de emergencia
	if n.smsSvc != nil && len(n.emergencyPhones) > 0 {
		results := n.smsSvc.SendBulkEmergencyAlert(n.emergencyPhones, fmt.Sprintf("Paciente %d", event.PatientID), reason)
		for _, r := range results {
			if !r.Success {
				lastErr = fmt.Errorf("SMS autoridades falhou: %s", r.Error)
			}
		}
	}

	// 2. Email detalhado para equipe
	if n.emailSvc != nil && len(n.emergencyEmails) > 0 {
		for _, addr := range n.emergencyEmails {
			err := n.emailSvc.SendEmergencyAlert(addr, "Autoridades", fmt.Sprintf("Paciente %d", event.PatientID), reason)
			if err != nil {
				lastErr = err
			}
		}
	}

	// 3. Webhook
	if n.webhookURL != "" {
		payload := NotificationPayload{
			EventID:          event.ID,
			PatientID:        event.PatientID,
			CrisisType:       string(event.CrisisType),
			Severity:         event.Severity,
			TriggerStatement: event.TriggerStatement,
			Timestamp:        event.CreatedAt,
			RequiresAction:   true,
		}
		if err := n.sendWebhook(ctx, payload); err != nil {
			lastErr = err
		}
	}

	if lastErr != nil {
		return fmt.Errorf("algumas notificacoes para autoridades falharam: %w", lastErr)
	}

	log.Warn().Int64("patient_id", event.PatientID).Str("crisis_type", string(event.CrisisType)).
		Msg("Notificacoes para autoridades enviadas")
	return nil
}
