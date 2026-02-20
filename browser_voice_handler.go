// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	gemini "eva-mind/internal/cortex/gemini"
	"eva-mind/internal/cortex/lacan"
	"eva-mind/internal/cortex/personality"
	"eva-mind/internal/cortex/voice/speaker"
	"eva-mind/internal/swarm"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// ============================================================================
// geminiApp — Voz e Video via WebSocket para App Mobile (EVA-Mobile)
// ============================================================================
// Consumer:  geminiApp
// Rota:      /ws/browser
// Client:    internal/gemini (v1beta, producao)
// Frontend:  App mobile EVA-Mobile / Malaria frontend
// Protocolo: WebSocket — audio PCM (16kHz in, 24kHz out) + video JPEG + texto
// Memoria:   Meta-cognitiva via Neo4j (carrega no inicio, salva transcricoes)
// CRITICO:   Protocolo WebSocket NAO pode mudar — app mobile depende
// Ver:       GEMINI_ARCHITECTURE.md para documentacao completa
//
// RECONEXAO: Quando o Gemini faz timeout (~10 min), o handler reconecta
// automaticamente sem fechar o WebSocket do browser. O browser recebe
// {"type":"status","text":"reconnecting"} e depois {"type":"status","text":"ready"}.

// browserWSUpgrader permite conexoes de browsers (CORS flexivel)
var browserWSUpgrader = gws.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Browser clients de qualquer origem
	},
}

// browserMessage formato de mensagem browser <-> server
type browserMessage struct {
	Type string `json:"type"`           // "audio", "text", "config", "status"
	Data string `json:"data,omitempty"` // base64 PCM para audio / "user" para transcricao
	Text string `json:"text,omitempty"` // texto para subtitles/chat
}

// browserSignalKind classifica o tipo de sinal do loop de reconexao
type browserSignalKind int

const (
	bsigFatal     browserSignalKind = iota // browser desconectou ou erro irrecuperavel
	bsigReconnect                          // Gemini fez timeout — reconectar sem fechar o browser WS
)

// browserSignal carrega o tipo de sinal, a geracao do client e o erro
type browserSignal struct {
	kind browserSignalKind
	gen  int64 // geracao do client Gemini que gerou o sinal (0 = writer goroutine)
	err  error
}

// isGeminiTimeout detecta erros de timeout/cancelamento da API Gemini Live.
// Esses erros sao esperados apos ~10 minutos de sessao e sao recuperaveis.
func isGeminiTimeout(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "CANCELLED") ||
		strings.Contains(msg, "Thread was cancelled") ||
		strings.Contains(msg, "websocket: close 1011") ||
		strings.Contains(msg, "context deadline exceeded")
}

// handleBrowserVoice lida com WebSocket de voz vindo do browser.
// Reconecta automaticamente ao Gemini quando a sessao expira (~10 min),
// sem interromper o WebSocket do browser (max 5 reconexoes por sessao).
func (s *SignalingServer) handleBrowserVoice(w http.ResponseWriter, r *http.Request) {
	conn, err := browserWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Browser WS upgrade failed")
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	sessionID := "browser-" + time.Now().Format("20060102150405")

	// --- Config inicial do cliente ---
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, configBytes, err := conn.ReadMessage()
	conn.SetReadDeadline(time.Time{})
	if err != nil {
		log.Error().Err(err).Msg("[BROWSER] Timeout esperando config do cliente")
		conn.WriteJSON(browserMessage{Type: "status", Text: "error: config timeout"})
		return
	}

	var configMsg browserMessage
	var clientContext string
	var clientCPF string
	if err := json.Unmarshal(configBytes, &configMsg); err == nil && configMsg.Type == "config" {
		clientContext = configMsg.Text
		clientCPF = configMsg.Data
	}

	if clientContext == "" {
		clientContext = "Voce e a EVA, assistente virtual inteligente. Responda em portugues de forma clara e profissional."
	}

	log.Info().Str("session", sessionID).Str("cpf", clientCPF).Bool("hasContext", configMsg.Text != "").Msg("[BROWSER] Config recebida do cliente")

	// --- Enriquecimento de contexto ---
	var idosoID int64

	if clientCPF != "" && lacan.IsCreatorCPF(clientCPF) && s.db != nil {
		log.Info().Str("session", sessionID).Msg("[BROWSER] === MODO CRIADOR ATIVADO ===")

		creatorSvc := personality.NewCreatorProfileService(s.db.Conn)
		profile, err := creatorSvc.LoadCreatorProfile(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("[BROWSER] Falha ao carregar perfil do criador")
		} else {
			clientContext = creatorSvc.GenerateSystemPrompt(profile)
		}

		debugMode := lacan.NewDebugMode(s.db.Conn)
		clientContext += "\n" + debugMode.BuildDebugPromptSection(ctx)

	} else if clientCPF != "" && s.db != nil {
		idoso, err := s.db.GetIdosoByCPF(clientCPF)
		if err != nil {
			log.Warn().Err(err).Str("cpf", clientCPF).Msg("[BROWSER] Pessoa nao encontrada")
		} else {
			idosoID = idoso.ID
			fullIdoso, err := s.db.GetIdoso(idoso.ID)
			if err == nil && fullIdoso != nil {
				clientContext += fmt.Sprintf("\n\nVoce esta conversando com %s (CPF: %s, nascido em %s). Use o nome dele/dela na conversa.",
					fullIdoso.Nome, clientCPF, fullIdoso.DataNascimento.Format("02/01/2006"))
				log.Info().Str("session", sessionID).Str("nome", fullIdoso.Nome).Int64("id", fullIdoso.ID).Msg("[BROWSER] Pessoa carregada")
			}

			rows, err := s.db.Conn.Query(`
				SELECT tipo, dados_tarefa, status, data_hora_agendada
				FROM agendamentos
				WHERE idoso_id = $1 AND status IN ('agendado','ativo','pendente','nao_atendido','aguardando_retry')
				ORDER BY data_hora_agendada ASC LIMIT 20`, idoso.ID)
			if err == nil {
				defer rows.Close()
				var medsInfo strings.Builder
				count := 0
				for rows.Next() {
					var tipo, dados, status string
					var dataHora time.Time
					if err := rows.Scan(&tipo, &dados, &status, &dataHora); err == nil {
						if count == 0 {
							medsInfo.WriteString("\n\n[MEDICAMENTOS E AGENDAMENTOS]")
						}
						medsInfo.WriteString(fmt.Sprintf("\n- %s: %s (Status: %s, Hora: %s)",
							tipo, dados, status, dataHora.Format("02/01 15:04")))
						count++
					}
				}
				if count > 0 {
					clientContext += medsInfo.String()
					log.Info().Str("session", sessionID).Int("count", count).Msg("[BROWSER] Agendamentos carregados")
				}
			}
		}
	}

	// --- Personalidade, memorias episodicas e sabedoria ---
	if idosoID > 0 {
		// #1 Personalidade: nivel de relacionamento, emocao, topicos
		var dominantEmotion string
		if s.personalityService != nil {
			state, err := s.personalityService.GetState(ctx, idosoID)
			if err == nil && state != nil {
				dominantEmotion = state.DominantEmotion
				clientContext += fmt.Sprintf("\n\n[RELACIONAMENTO] Nivel: %d/10, Conversas anteriores: %d, Emocao dominante: %s",
					state.RelationshipLevel, state.ConversationCount, state.DominantEmotion)
				if len(state.FavoriteTopics) > 0 {
					clientContext += fmt.Sprintf(", Topicos favoritos: %s", strings.Join(state.FavoriteTopics, ", "))
				}
				log.Info().Str("session", sessionID).Int("level", state.RelationshipLevel).Str("emotion", state.DominantEmotion).Msg("[BROWSER] Personalidade carregada")
			}
		}

		// #2 Memorias episodicas recentes
		if s.memoryStore != nil {
			recentMems, err := s.memoryStore.GetRecent(ctx, idosoID, 5)
			if err == nil && len(recentMems) > 0 {
				var memBuf strings.Builder
				memBuf.WriteString("\n\n[MEMORIAS RECENTES]")
				for _, m := range recentMems {
					content := m.Content
					if len(content) > 150 {
						content = content[:150] + "..."
					}
					memBuf.WriteString(fmt.Sprintf("\n- [%s] %s: %s",
						m.Timestamp.Format("02/01 15:04"), m.Speaker, content))
				}
				clientContext += memBuf.String()
				log.Info().Str("session", sessionID).Int("count", len(recentMems)).Msg("[BROWSER] Memorias episodicas carregadas")
			}
		}

		// #3 Sabedoria terapeutica (busca semantica por emocao)
		if s.wisdomService != nil && dominantEmotion != "" && dominantEmotion != "neutro" {
			wisdomCtx := s.wisdomService.GetWisdomContext(ctx, dominantEmotion, nil)
			if wisdomCtx != "" {
				clientContext += "\n\n[SABEDORIA TERAPEUTICA]\n" + wisdomCtx
				log.Info().Str("session", sessionID).Str("emotion", dominantEmotion).Msg("[BROWSER] Sabedoria injetada")
			}
		}

		// --- FASE 2+3+4: Inteligencia avancada (EVA livre, sem controladores) ---

		// #4 FDPN: padrao de demanda lacaniano (a quem a pessoa dirige suas demandas)
		if s.fdpnEngine != nil {
			demandCtx := s.fdpnEngine.BuildGraphContext(ctx, idosoID)
			if demandCtx != "" {
				clientContext += "\n\n[PADRAO DE DEMANDA]\n" + demandCtx
				log.Info().Str("session", sessionID).Msg("[BROWSER] FDPN context injetado")
			}
		}

		// #5 Habitos: resumo diario (agua, medicamento, exercicio, sono)
		if s.habitTracker != nil {
			summary, err := s.habitTracker.GetDailySummary(ctx, idosoID)
			if err == nil && summary != nil {
				var habitBuf strings.Builder
				habitBuf.WriteString("\n\n[HABITOS DO DIA]")
				if habits, ok := summary["habits"].([]interface{}); ok {
					for _, h := range habits {
						if hm, ok := h.(map[string]interface{}); ok {
							habitBuf.WriteString(fmt.Sprintf("\n- %v: %v", hm["name"], hm["status"]))
						}
					}
				}
				if streak, ok := summary["best_streak"]; ok {
					habitBuf.WriteString(fmt.Sprintf("\n- Melhor sequencia: %v dias", streak))
				}
				clientContext += habitBuf.String()
				log.Info().Str("session", sessionID).Msg("[BROWSER] Habitos injetados")
			}
		}

		// #6 Spaced Repetition: memorias pendentes de revisao
		if s.spacedRepetition != nil {
			reviews, err := s.spacedRepetition.GetPendingReviews(ctx, idosoID, 5)
			if err == nil && len(reviews) > 0 {
				var revBuf strings.Builder
				revBuf.WriteString("\n\n[MEMORIAS PARA REVISAR - reforce naturalmente na conversa]")
				for _, r := range reviews {
					revBuf.WriteString(fmt.Sprintf("\n- [%s] %s (gatilho: %s)",
						r.Category, r.Content, r.Trigger))
				}
				clientContext += revBuf.String()
				log.Info().Str("session", sessionID).Int("count", len(reviews)).Msg("[BROWSER] Revisoes pendentes injetadas")
			}
		}

		// #7 Superhuman Memory: espelho psicologico profundo
		if s.superhumanMemory != nil {
			mirrors, err := s.superhumanMemory.GenerateComprehensiveMirror(ctx, idosoID)
			if err == nil && len(mirrors) > 0 {
				var mirrorBuf strings.Builder
				mirrorBuf.WriteString("\n\n[PERFIL PSICOLOGICO - use como intuicao, NAO mencione diretamente]")
				for _, m := range mirrors {
					if m.Question != "" {
						mirrorBuf.WriteString(fmt.Sprintf("\n- [%s] %s", m.Type, m.Question))
					} else if len(m.DataPoints) > 0 {
						mirrorBuf.WriteString(fmt.Sprintf("\n- [%s] %s", m.Type, m.DataPoints[0]))
					}
				}
				clientContext += mirrorBuf.String()
				log.Info().Str("session", sessionID).Int("insights", len(mirrors)).Msg("[BROWSER] Espelho psicologico injetado")
			}
		}

		// #8 Conhecimento aprendido autonomamente (Scholar Agent)
		if s.autonomousLearner != nil {
			searchQuery := dominantEmotion
			if searchQuery == "" || searchQuery == "neutro" {
				searchQuery = "bem-estar saude mental"
			}
			learningCtx := s.autonomousLearner.GetLearningContext(ctx, searchQuery)
			if learningCtx != "" {
				clientContext += "\n\n[CONHECIMENTO APRENDIDO]\n" + learningCtx
				log.Info().Str("session", sessionID).Msg("[BROWSER] Conhecimento aprendido injetado")
			}
		}
	}

	// --- Memoria meta-cognitiva (Neo4j) ---
	var memories []string
	if s.evaMemory != nil {
		if err := s.evaMemory.StartSession(ctx, sessionID); err != nil {
			log.Warn().Err(err).Msg("[BROWSER] Falha ao registrar sessao no Neo4j")
		}
		metaCognition, err := s.evaMemory.LoadMetaCognition(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("[BROWSER] Falha ao carregar memoria meta-cognitiva")
		} else if metaCognition != "" {
			memories = []string{metaCognition}
			log.Info().Str("session", sessionID).Msg("[BROWSER] Memoria meta-cognitiva injetada")
		}
	}

	// --- Identidade pessoal da EVA (CoreMemoryEngine — Neo4j) ---
	// GetIdentityContext() retorna: personalidade OCEAN, memorias importantes, autodescricao, capacidades
	if s.coreMemory != nil {
		identityCtx, err := s.coreMemory.GetIdentityContext(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("[BROWSER] Falha ao carregar identidade pessoal (CoreMemory)")
		} else if identityCtx != "" {
			clientContext = identityCtx + "\n\n---\n\n" + clientContext
			log.Info().Str("session", sessionID).Msg("[BROWSER] Identidade pessoal injetada (CoreMemory)")
		}
	}

	// --- setupGemini: cria e configura um novo client Gemini ---
	// Captura clientContext e memories do escopo externo — sao imutaveis apos esta linha.
	setupGemini := func() (*gemini.Client, error) {
		client, err := gemini.NewClient(ctx, s.cfg)
		if err != nil {
			return nil, err
		}
		if err := client.SendSetup(
			clientContext,
			map[string]interface{}{"voiceName": "Aoede", "languageCode": "pt-BR"},
			memories, "", nil,
		); err != nil {
			client.Close()
			return nil, err
		}
		return client, nil
	}

	initialClient, err := setupGemini()
	if err != nil {
		log.Error().Err(err).Msg("Erro ao configurar Gemini para browser")
		conn.WriteJSON(browserMessage{Type: "status", Text: "error: setup failed"})
		return
	}

	log.Info().Str("session", sessionID).Int("memories", len(memories)).Msg("Browser voice session started (cortex/gemini)")

	conn.WriteJSON(browserMessage{Type: "status", Text: "ready"})

	// --- Estado compartilhado entre goroutines ---
	var writeMu sync.Mutex      // protege escritas no conn do browser
	var geminiMu sync.RWMutex   // protege geminiRef
	geminiRef := initialClient  // client Gemini ativo
	var currentGen int64 = 1    // geracao atual (incrementada a cada reconexao)
	var reconnecting atomic.Bool // true enquanto reconexao em progresso
	var responseAccum strings.Builder

	// sigChan recebe sinais das goroutines para o loop principal
	// Buffer 4: captura sinais de goroutines mortas sem bloquear
	sigChan := make(chan browserSignal, 4)

	const maxReconnects = 5

	// --- startReader: lanca goroutine que le do client Gemini e encaminha ao browser ---
	// gen identifica a geracao deste client.
	// Sinais de geracoes antigas (goroutines mortas pelo Close do client anterior)
	// sao filtrados no loop principal usando a comparacao de gen.
	startReader := func(client *gemini.Client, gen int64) {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				resp, err := client.ReadResponse()
				if err != nil {
					if isGeminiTimeout(err) {
						sigChan <- browserSignal{kind: bsigReconnect, gen: gen, err: err}
					} else {
						sigChan <- browserSignal{kind: bsigFatal, gen: gen, err: err}
					}
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
					conn.WriteJSON(browserMessage{Type: "status", Text: "interrupted"})
					writeMu.Unlock()
					responseAccum.Reset()
					continue
				}

				if turnComplete, ok := serverContent["turnComplete"].(bool); ok && turnComplete {
					writeMu.Lock()
					conn.WriteJSON(browserMessage{Type: "status", Text: "turn_complete"})
					writeMu.Unlock()
					if s.evaMemory != nil && responseAccum.Len() > 0 {
						go s.evaMemory.StoreTurn(ctx, sessionID, "assistant", responseAccum.String())
					}
					responseAccum.Reset()
					continue
				}

				if inputTrans, ok := serverContent["inputAudioTranscription"].(map[string]interface{}); ok {
					if text, ok := inputTrans["text"].(string); ok && text != "" {
						writeMu.Lock()
						conn.WriteJSON(browserMessage{Type: "text", Text: text, Data: "user"})
						writeMu.Unlock()
						if s.evaMemory != nil {
							go s.evaMemory.StoreTurn(ctx, sessionID, "user", text)
						}

						// --- Tool Detection & Execution ---
						if s.toolsClient != nil && s.toolsHandler != nil && idosoID > 0 {
							go func(transcript string, patientID int64) {
								toolCalls, err := s.toolsClient.AnalyzeTranscription(ctx, transcript, "user")
								if err != nil {
									log.Warn().Err(err).Msg("[TOOLS] Falha na analise de transcricao")
									return
								}
								for _, tc := range toolCalls {
									if tc.Name == "" || tc.Name == "none" {
										continue
									}
									log.Info().Str("tool", tc.Name).Interface("args", tc.Args).Msg("[TOOLS] Tool detectada na fala")

									result, err := s.toolsHandler.ExecuteTool(tc.Name, tc.Args, patientID)
									if err != nil && strings.Contains(err.Error(), "ferramenta desconhecida") {
										// Fallthrough: tool nao existe no handlers.go, tentar swarm orchestrator
										if s.swarmOrchestrator != nil {
											swarmCall := swarm.ToolCall{
												Name:   tc.Name,
												Args:   tc.Args,
												UserID: patientID,
											}
											swarmResult, swarmErr := s.swarmOrchestrator.Route(ctx, swarmCall)
											if swarmErr == nil && swarmResult != nil {
												result = map[string]interface{}{
													"success": swarmResult.Success,
													"message": swarmResult.Message,
												}
												err = nil
												log.Info().Str("tool", tc.Name).Msg("[SWARM] Tool executada via swarm orchestrator")
											}
										}
									}
									if err != nil {
										log.Error().Err(err).Str("tool", tc.Name).Msg("[TOOLS] Erro ao executar tool")
										continue
									}

									// Envia ao browser: ui_action para control_ui, tool_event para o resto
									if isUI, _ := result["ui_action"].(bool); isUI {
										uiPayload := map[string]interface{}{
											"type":    "ui_action",
											"action":  result["action"],
											"target":  result["target"],
											"mode":    result["mode"],
											"url":     result["url"],
											"message": result["message_ui"],
										}
										writeMu.Lock()
										conn.WriteJSON(uiPayload)
										writeMu.Unlock()
										log.Info().Str("action", fmt.Sprint(result["action"])).Msg("[UI] ui_action enviado ao browser")
									} else {
										toolEventPayload := map[string]interface{}{
											"type":      "tool_event",
											"tool":      tc.Name,
											"tool_data": result,
											"status":    "success",
										}
										writeMu.Lock()
										conn.WriteJSON(toolEventPayload)
										writeMu.Unlock()
										log.Info().Str("tool", tc.Name).Msg("[TOOLS] tool_event enviado ao browser")
									}

									// Envia resultado como texto para o Gemini processar
									toolMsg := fmt.Sprintf("[TOOL_RESULT:%s] %v", tc.Name, result["message"])
									geminiMu.RLock()
									c := geminiRef
									geminiMu.RUnlock()
									if c != nil {
										c.SendText(toolMsg)
									}
									log.Info().Str("tool", tc.Name).Msg("[TOOLS] Resultado enviado ao Gemini")
								}
							}(text, idosoID)
						}
					}
				}

				if outputTrans, ok := serverContent["outputAudioTranscription"].(map[string]interface{}); ok {
					if text, ok := outputTrans["text"].(string); ok && text != "" {
						writeMu.Lock()
						conn.WriteJSON(browserMessage{Type: "text", Text: text})
						writeMu.Unlock()
						responseAccum.WriteString(text)
					}
				}

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
						conn.WriteJSON(browserMessage{Type: "text", Text: text})
						writeMu.Unlock()
					}
					if inlineData, ok := part["inlineData"].(map[string]interface{}); ok {
						if audioB64, ok := inlineData["data"].(string); ok {
							writeMu.Lock()
							conn.WriteJSON(browserMessage{Type: "audio", Data: audioB64})
							writeMu.Unlock()
						}
					}
				}
			}
		}()
	}

	startReader(initialClient, 1)

	// --- Speaker Recognition: register callback ---
	if s.speakerSvc != nil {
		s.speakerSvc.SetCallback(sessionID, func(sid string, msg speaker.SpeakerMessage) {
			writeMu.Lock()
			conn.WriteJSON(msg)
			writeMu.Unlock()
		})
		defer s.speakerSvc.RemoveSession(sessionID)
	}

	// --- Goroutine: Browser -> Gemini ---
	// Usa gen=0 no sinal para que o loop principal sempre o processe (e nunca o filtre).
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				sigChan <- browserSignal{kind: bsigFatal, gen: 0, err: err}
				return
			}

			// Dropa mensagens enquanto reconexao ao Gemini esta em progresso
			if reconnecting.Load() {
				continue
			}

			var msg browserMessage
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				continue
			}

			geminiMu.RLock()
			client := geminiRef
			geminiMu.RUnlock()

			if client == nil {
				continue
			}

			switch msg.Type {
			case "audio":
				pcmData, err := base64.StdEncoding.DecodeString(msg.Data)
				if err == nil {
					client.SendAudio(pcmData)
					if s.speakerSvc != nil {
						go s.speakerSvc.ProcessAudioChunk(sessionID, clientCPF, pcmData)
					}
				}
			case "video":
				jpegData, err := base64.StdEncoding.DecodeString(msg.Data)
				if err == nil {
					client.SendImage(jpegData)
				}
			case "text":
				if msg.Text != "" {
					if s.evaMemory != nil {
						go s.evaMemory.StoreTurn(ctx, sessionID, "user", msg.Text)
					}
					client.SendText(msg.Text)
				}
			case "config":
				log.Info().Str("session", sessionID).Msg("Browser sent config update")
			}
		}
	}()

	// --- Loop principal: processa sinais e reconecta quando necessario ---
	reconnectCount := 0
	var finalErr error

	for {
		sig := <-sigChan

		// Filtra sinais de geracoes antigas (goroutines mortas pelo Close do client anterior).
		// gen=0 e reservado para o writer goroutine e nunca filtrado.
		if sig.gen != 0 && sig.gen != atomic.LoadInt64(&currentGen) {
			continue
		}

		if sig.kind == bsigFatal {
			finalErr = sig.err
			break
		}

		// bsigReconnect: Gemini expirou — reconectar sem fechar o WebSocket do browser
		reconnectCount++
		if reconnectCount > maxReconnects {
			log.Error().
				Str("session", sessionID).
				Int("attempts", reconnectCount).
				Msg("[BROWSER] Limite de reconexoes atingido — encerrando sessao")
			writeMu.Lock()
			conn.WriteJSON(browserMessage{Type: "status", Text: "error: max reconnects exceeded"})
			writeMu.Unlock()
			break
		}

		log.Warn().
			Str("session", sessionID).
			Int("attempt", reconnectCount).
			Err(sig.err).
			Msg("[BROWSER] Gemini timeout — reconectando...")

		reconnecting.Store(true)
		writeMu.Lock()
		conn.WriteJSON(browserMessage{Type: "status", Text: "reconnecting"})
		writeMu.Unlock()

		// Fecha client antigo (faz a goroutine reader antiga retornar)
		geminiMu.Lock()
		old := geminiRef
		geminiRef = nil
		geminiMu.Unlock()
		if old != nil {
			old.Close()
		}

		// Backoff antes de reconectar (evita hammering na API)
		time.Sleep(1500 * time.Millisecond)

		newClient, err := setupGemini()
		if err != nil {
			log.Error().Err(err).Str("session", sessionID).Msg("[BROWSER] Falha ao reconectar ao Gemini")
			writeMu.Lock()
			conn.WriteJSON(browserMessage{Type: "status", Text: "error: reconnect failed"})
			writeMu.Unlock()
			break
		}

		// Incrementa geracao ANTES de atualizar geminiRef para que o loop principal
		// ignore eventuais sinais tardios da goroutine antiga
		newGen := atomic.AddInt64(&currentGen, 1)
		geminiMu.Lock()
		geminiRef = newClient
		geminiMu.Unlock()

		reconnecting.Store(false)
		startReader(newClient, newGen)

		writeMu.Lock()
		conn.WriteJSON(browserMessage{Type: "status", Text: "ready"})
		writeMu.Unlock()

		log.Info().
			Str("session", sessionID).
			Int("attempt", reconnectCount).
			Int64("gen", newGen).
			Msg("[BROWSER] Reconexao ao Gemini bem-sucedida")
	}

	// --- Finalizar sessao no Neo4j ---
	if s.evaMemory != nil {
		s.evaMemory.EndSession(ctx, sessionID)
		go s.evaMemory.DetectPatterns(context.Background())
	}

	log.Info().
		Str("session", sessionID).
		Int("reconnects", reconnectCount).
		Err(finalErr).
		Msg("Browser voice session ended")
}
