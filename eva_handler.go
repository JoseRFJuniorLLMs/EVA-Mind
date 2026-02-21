// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"encoding/json"
	gemini "eva-mind/internal/cortex/gemini"
	evaSelf "eva-mind/internal/cortex/self"
	"net/http"
	"strings"
	"sync"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// evaWSUpgrader permite conexoes do browser para /ws/eva
var evaWSUpgrader = gws.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// evaMessage formato de mensagem browser <-> EVA
type evaMessage struct {
	Type string `json:"type"`           // "text", "status"
	Data string `json:"data,omitempty"` // metadado opcional
	Text string `json:"text,omitempty"` // conteudo de texto
}

// ============================================================================
// geminiWeb — Chat de Texto via WebSocket para pagina /eva (Malaria-Angolar)
// ============================================================================
// Consumer:  geminiWeb
// Rota:      /ws/eva
// Client:    internal/cortex/gemini (v1beta, producao)
// Frontend:  Malaria-Angolar/frontend/src/pages/EvaPage.tsx
// Memoria:   Meta-cognitiva via Neo4j (internal/cortex/eva_memory)
//            Identidade pessoal via CoreMemoryEngine (Neo4j :7688)
// Protocolo:
//
//	Browser envia: {"type":"text","text":"pergunta do usuario"}
//	Server envia:  {"type":"text","text":"chunk da resposta"}  (streaming)
//	Server envia:  {"type":"status","text":"ready|turn_complete|interrupted|error"}
//
// Ver:       GEMINI_ARCHITECTURE.md para documentacao completa
func (s *SignalingServer) handleEvaChat(w http.ResponseWriter, r *http.Request) {
	conn, err := evaWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("EVA WS upgrade failed")
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Cria sessao Gemini usando cortex/gemini (producao)
	geminiClient, err := gemini.NewClient(ctx, s.cfg)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao criar cortex/gemini client para EVA")
		conn.WriteJSON(evaMessage{Type: "status", Text: "error: " + err.Error()})
		return
	}
	defer geminiClient.Close()

	sessionID := "eva-" + time.Now().Format("20060102150405")
	sessionStart := time.Now()

	// Espera primeira mensagem do cliente (deve ser tipo "config" com CPF)
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, configBytes, err := conn.ReadMessage()
	conn.SetReadDeadline(time.Time{})
	if err != nil {
		log.Error().Err(err).Msg("[EVA] Timeout esperando config do cliente")
		conn.WriteJSON(evaMessage{Type: "status", Text: "error: config timeout"})
		return
	}

	var configMsg evaMessage
	var clientContext string
	var clientCPF string
	if err := json.Unmarshal(configBytes, &configMsg); err == nil && configMsg.Type == "config" {
		clientContext = configMsg.Text
		clientCPF = configMsg.Data
	}

	// Se cliente nao enviou contexto, usa generico minimo
	if clientContext == "" {
		clientContext = "Voce e a EVA, assistente virtual inteligente. Responda em portugues de forma clara e profissional."
	}

	log.Info().Str("session", sessionID).Str("cpf", clientCPF).Bool("hasContext", configMsg.Text != "").Msg("[EVA] Config recebida do cliente")

	// Carregar memoria meta-cognitiva do Neo4j (sistema legado :7687)
	var memories []string
	if s.evaMemory != nil {
		if err := s.evaMemory.StartSession(ctx, sessionID); err != nil {
			log.Warn().Err(err).Msg("[EVA] Falha ao registrar sessao no Neo4j")
		}

		metaCognition, err := s.evaMemory.LoadMetaCognition(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("[EVA] Falha ao carregar memoria meta-cognitiva")
		} else if metaCognition != "" {
			memories = []string{metaCognition}
			log.Info().Str("session", sessionID).Msg("[EVA] Memoria meta-cognitiva injetada")
		}
	}

	// Injetar identidade pessoal da EVA (CoreMemoryEngine — Neo4j :7688)
	// GetIdentityContext() retorna: personalidade OCEAN, memorias importantes, autodescricao
	if s.coreMemory != nil {
		identityCtx, err := s.coreMemory.GetIdentityContext(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("[EVA] Falha ao carregar identidade pessoal (CoreMemory)")
		} else if identityCtx != "" {
			clientContext = identityCtx + "\n\n---\n\n" + clientContext
			log.Info().Str("session", sessionID).Msg("[EVA] Identidade pessoal injetada (CoreMemory)")
		}
	}

	// Setup com cortex/gemini — contexto vem do frontend, memoria do Neo4j
	err = geminiClient.SendSetup(
		clientContext,
		map[string]interface{}{
			"voiceName":    "Aoede",
			"languageCode": "pt-BR",
		},
		memories,
		"",
		nil,
	)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao configurar cortex/gemini para EVA")
		conn.WriteJSON(evaMessage{Type: "status", Text: "error: setup failed"})
		return
	}

	log.Info().Str("session", sessionID).Int("memories", len(memories)).Msg("EVA chat session started (cortex/gemini)")

	// Notifica browser que esta pronto
	conn.WriteJSON(evaMessage{Type: "status", Text: "ready"})

	var writeMu sync.Mutex
	var transcriptMu sync.Mutex
	errChan := make(chan error, 2)

	// Buffers para acumular conversa
	var responseAccum strings.Builder  // buffer por turno (EVA)
	var transcriptAccum strings.Builder // transcript completo da sessao
	var evaResponses []string           // respostas da EVA para CoreMemory

	// Goroutine: Gemini -> Browser (text responses)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				resp, err := geminiClient.ReadResponse()
				if err != nil {
					errChan <- err
					return
				}

				if _, ok := resp["setupComplete"]; ok {
					continue
				}

				serverContent, ok := resp["serverContent"].(map[string]interface{})
				if !ok {
					continue
				}

				if interrupted, ok := serverContent["interrupted"].(bool); ok && interrupted {
					writeMu.Lock()
					conn.WriteJSON(evaMessage{Type: "status", Text: "interrupted"})
					writeMu.Unlock()
					responseAccum.Reset()
					continue
				}

				if turnComplete, ok := serverContent["turnComplete"].(bool); ok && turnComplete {
					writeMu.Lock()
					conn.WriteJSON(evaMessage{Type: "status", Text: "turn_complete"})
					writeMu.Unlock()

					// Salvar resposta do turno no Neo4j legado + acumular transcript
					if responseAccum.Len() > 0 {
						turn := responseAccum.String()
						if s.evaMemory != nil {
							go s.evaMemory.StoreTurn(ctx, sessionID, "assistant", turn)
						}
						transcriptMu.Lock()
						transcriptAccum.WriteString("EVA: " + turn + "\n")
						evaResponses = append(evaResponses, turn)
						transcriptMu.Unlock()
					}
					responseAccum.Reset()
					continue
				}

				// Transcricao do output do modelo
				if outputTrans, ok := serverContent["outputAudioTranscription"].(map[string]interface{}); ok {
					if text, ok := outputTrans["text"].(string); ok && text != "" {
						writeMu.Lock()
						conn.WriteJSON(evaMessage{Type: "text", Text: text})
						writeMu.Unlock()
						responseAccum.WriteString(text)
					}
				}

				// Texto direto do modelo
				modelTurn, ok := serverContent["modelTurn"].(map[string]interface{})
				if !ok {
					continue
				}

				parts, ok := modelTurn["parts"].([]interface{})
				if !ok {
					continue
				}

				for _, p := range parts {
					part, ok := p.(map[string]interface{})
					if !ok {
						continue
					}
					if text, ok := part["text"].(string); ok && text != "" {
						writeMu.Lock()
						conn.WriteJSON(evaMessage{Type: "text", Text: text})
						writeMu.Unlock()
						responseAccum.WriteString(text)
					}
				}
			}
		}
	}()

	// Goroutine: Browser -> Gemini (text input)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, msgBytes, err := conn.ReadMessage()
				if err != nil {
					errChan <- err
					return
				}

				var msg evaMessage
				if err := json.Unmarshal(msgBytes, &msg); err != nil {
					continue
				}

				if msg.Type == "text" && msg.Text != "" {
					// Salvar mensagem do usuario no Neo4j legado + acumular transcript
					if s.evaMemory != nil {
						go s.evaMemory.StoreTurn(ctx, sessionID, "user", msg.Text)
					}
					transcriptMu.Lock()
					transcriptAccum.WriteString("Usuario: " + msg.Text + "\n")
					transcriptMu.Unlock()
					if err := geminiClient.SendText(msg.Text); err != nil {
						log.Error().Err(err).Str("session", sessionID).Msg("[EVA] Erro ao enviar texto para Gemini")
						writeMu.Lock()
						conn.WriteJSON(evaMessage{Type: "status", Text: "error: falha ao enviar mensagem"})
						writeMu.Unlock()
					}
				}
			}
		}
	}()

	sessionErr := <-errChan

	// --- Notificar browser do erro antes de fechar ---
	if sessionErr != nil {
		errMsg := sessionErr.Error()
		var browserErr string
		switch {
		case strings.Contains(errMsg, "websocket: close"):
			browserErr = "error: conexao com IA perdida (" + errMsg + ")"
		case strings.Contains(errMsg, "context canceled"):
			browserErr = "error: sessao cancelada"
		case strings.Contains(errMsg, "i/o timeout"):
			browserErr = "error: timeout na comunicacao"
		default:
			browserErr = "error: " + errMsg
		}
		writeMu.Lock()
		conn.WriteJSON(evaMessage{Type: "status", Text: browserErr})
		writeMu.Unlock()
		log.Error().Str("session", sessionID).Str("error_sent", browserErr).Msg("[EVA] Erro enviado ao cliente")
	}

	// Finalizar sessao no Neo4j legado
	if s.evaMemory != nil {
		s.evaMemory.EndSession(ctx, sessionID)
		go s.evaMemory.DetectPatterns(context.Background())
	}

	// Processar fim de sessao no CoreMemoryEngine (reflexao + memorias pessoais)
	if s.coreMemory != nil {
		transcript := transcriptAccum.String()
		if transcript != "" {
			duration := time.Since(sessionStart).Minutes()
			go func() {
				data := evaSelf.SessionData{
					SessionID:       sessionID,
					Transcript:      transcript,
					DurationMinutes: duration,
					EVAResponses:    evaResponses,
					Timestamp:       sessionStart,
				}
				bgCtx := context.Background()
				if err := s.coreMemory.ProcessSessionEnd(bgCtx, data); err != nil {
					log.Warn().Err(err).Str("session", sessionID).Msg("[CoreMemory] Falha ao processar fim de sessao")
				} else {
					log.Info().Str("session", sessionID).Msg("[CoreMemory] Sessao processada — memorias pessoais atualizadas")
				}
			}()
		}
	}

	log.Info().Str("session", sessionID).Err(sessionErr).Msg("EVA chat session ended")
}
