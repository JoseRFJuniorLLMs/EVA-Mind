// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package voice

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"eva-mind/internal/brainstem/config"
	"eva-mind/internal/brainstem/database"
	"eva-mind/internal/cortex/gemini"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

const (
	// Reduzido de 9600 para 4800 para diminuir latência (~200ms @ 24kHz)
	MIN_BUFFER_SIZE = 4800
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // Permitir conexões sem Origin (ex: apps mobile, curl)
		}
		allowedOrigins := []string{
			"https://eva-ia.org",
			"https://www.eva-ia.org",
			"https://app.eva-ia.org",
			"http://localhost:3000",
			"http://localhost:8080",
		}
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				return true
			}
		}
		return false
	},
}

// bufferPool reutiliza slices de byte para evitar pressão no GC
var bufferPool = sync.Pool{
	New: func() interface{} {
		// Aloca capacidade para o buffer máximo esperado
		return make([]byte, 0, 19200)
	},
}

// muLawToPcmTable é pré-calculada para evitar math.Pow/Log em runtime
var muLawToPcmTable [256]int16

func init() {
	// Pré-calcula tabela MuLaw
	for i := 0; i < 256; i++ {
		muLawToPcmTable[i] = decodeMuLawByte(byte(i))
	}
}

func decodeMuLawByte(muLaw byte) int16 {
	const bias = 132
	muLaw = ^muLaw
	sign := (muLaw & 0x80) >> 7
	exponent := (muLaw & 0x70) >> 4
	mantissa := muLaw & 0x0F
	sample := (int16(mantissa)<<3 + bias) << uint(exponent)
	if sign != 0 {
		return bias - sample
	}
	return sample - bias
}

type Handler struct {
	db            *database.DB
	cfg           *config.Config
	logger        zerolog.Logger
	alertService  *AlertService
	geminiHandler *gemini.Handler
}

func NewHandler(db *database.DB, cfg *config.Config, logger zerolog.Logger, alertService *AlertService, gh *gemini.Handler) *Handler {
	return &Handler{
		db:            db,
		cfg:           cfg,
		logger:        logger,
		alertService:  alertService,
		geminiHandler: gh,
	}
}

// AudioSession mantém o estado volátil da conexão de voz
type AudioSession struct {
	SessionID    string
	TurnID       uint64 // Controle de versão para evitar Race Conditions lógicas
	AudioBuffer  []byte
	BufferMu     sync.Mutex
	LastActivity time.Time
}

func (h *Handler) HandleMediaStream(w http.ResponseWriter, r *http.Request) {
	agIDStr := r.URL.Query().Get("agendamento_id")
	if agIDStr == "" {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) > 0 {
			agIDStr = parts[len(parts)-1]
		}
	}

	if agIDStr == "" || agIDStr == "stream" {
		http.Error(w, "agendamento_id obrigatório", http.StatusBadRequest)
		return
	}

	// Recupera sessão segura do Gemini (criada no Scheduler)
	geminiSession := GetSession(agIDStr)
	if geminiSession == nil {
		h.logger.Error().Str("session_id", agIDStr).Msg("Sessão Gemini não encontrada ou expirada")
		http.Error(w, "Sessão não encontrada", http.StatusNotFound)
		return
	}

	// Upgrade WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("WebSocket upgrade failed")
		return
	}
	defer conn.Close()

	// Contexto da chamada
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Estado local da sessão de áudio
	audioSess := &AudioSession{
		SessionID:    agIDStr,
		TurnID:       0,
		AudioBuffer:  bufferPool.Get().([]byte)[:0], // Resetar length, manter cap
		LastActivity: time.Now(),
	}
	defer bufferPool.Put(audioSess.AudioBuffer) // Devolver ao pool no final

	h.logger.Info().Str("ag_id", agIDStr).Msg("🎙️ Twilio Media Stream connected")

	// Canal de erro/sinalização
	errChan := make(chan error, 1)
	streamSidChan := make(chan string, 1)

	// Goroutine: Gemini -> Twilio (Output Audio)
	go func() {
		var streamSid string

		// Aguarda StreamSID do handshake inicial
		select {
		case sid := <-streamSidChan:
			streamSid = sid
		case <-ctx.Done():
			return
		case <-time.After(15 * time.Second):
			h.logger.Warn().Msg("Timeout aguardando StreamSID")
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Leitura bloqueante com timeout interno no client
				resp, err := geminiSession.Client.ReadResponse()
				if err != nil {
					h.logger.Error().Err(err).Msg("Erro leitura Gemini")
					errChan <- err
					return
				}

				// Processamento centralizado no GeminiHandler
				// Passamos o TurnID atual para garantir consistência
				audioData, turnUpdated := h.geminiHandler.ProcessResponse(ctx, geminiSession, resp, audioSess.TurnID)

				if turnUpdated {
					// Se o modelo finalizou um turno, incrementamos localmente se necessário
					// (Lógica específica de controle de fluxo pode ir aqui)
				}

				if len(audioData) > 0 {
					// Conversão Otimizada PCM -> MuLaw (Twilio)
					// TODO: Implementar conversor otimizado reverso se necessário.
					// Por enquanto, assumimos que o GeminiHandler já tratou ou usamos lib externa.
					// Aqui apenas encodamos Base64 para o WebSocket

					payload := base64.StdEncoding.EncodeToString(audioData)
					msg := map[string]interface{}{
						"event":     "media",
						"streamSid": streamSid,
						"media": map[string]string{
							"payload": payload,
						},
					}

					if err := conn.WriteJSON(msg); err != nil {
						return
					}
				}
			}
		}
	}()

	// Goroutine: Twilio -> Gemini (Input Audio)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var msg map[string]interface{}
				if err := conn.ReadJSON(&msg); err != nil {
					errChan <- err
					return
				}

				event, _ := msg["event"].(string)

				switch event {
				case "start":
					if start, ok := msg["start"].(map[string]interface{}); ok {
						if sid, ok := start["streamSid"].(string); ok {
							streamSidChan <- sid
							// Inicia a conversa
							if err := geminiSession.Client.SendText("O usuário atendeu. Diga 'Olá'."); err != nil {
								h.logger.Error().Err(err).Msg("Erro ao enviar texto inicial para Gemini")
								errChan <- fmt.Errorf("gemini SendText: %w", err)
								return
							}
						}
					}
				case "media":
					if media, ok := msg["media"].(map[string]interface{}); ok {
						if payload, ok := media["payload"].(string); ok {
							data, _ := base64.StdEncoding.DecodeString(payload)

							// Incrementa TurnID se silêncio foi quebrado (VAD simples)
							// Na prática, o Gemini faz VAD, mas aqui marcamos atividade
							atomic.StoreUint64(&audioSess.TurnID, audioSess.TurnID+1)

							// Processamento DSP Otimizado
							pcmData := processAudioChunk(data)

							audioSess.BufferMu.Lock()
							audioSess.AudioBuffer = append(audioSess.AudioBuffer, pcmData...)

							// Flush se buffer cheio
							if len(audioSess.AudioBuffer) >= MIN_BUFFER_SIZE {
								if err := geminiSession.Client.SendAudio(audioSess.AudioBuffer); err != nil {
									h.logger.Error().Err(err).Msg("Erro ao enviar audio para Gemini")
								}
								// Reset buffer mantendo capacidade
								audioSess.AudioBuffer = audioSess.AudioBuffer[:0]
							}
							audioSess.BufferMu.Unlock()
						}
					}
				case "stop":
					errChan <- nil
					return
				}
			}
		}
	}()

	<-errChan
	h.logger.Info().Msg("Chamada finalizada")
}

// processAudioChunk converte MuLaw para PCM16 usando Lookup Table (Zero allocs se possível)
func processAudioChunk(mulaw []byte) []byte {
	// 1 byte MuLaw -> 2 bytes PCM16
	// Usa buffer pool se possível, mas aqui retornamos slice para simplificar a assinatura
	// Em alta performance, passariamos o buffer de destino
	pcm := make([]byte, len(mulaw)*2)

	for i, b := range mulaw {
		val := muLawToPcmTable[b]
		// Little Endian
		pcm[i*2] = byte(val)
		pcm[i*2+1] = byte(val >> 8)
	}

	// Aqui entraria o Resample 8k -> 16k ou 24k se necessário
	// Implementação simples de duplicação de amostra (Low Quality, High Perf)
	// Para produção, usar filtro FIR polifásico
	return upsampleLinear(pcm)
}

// upsampleLinear faz interpolação linear simples 8kHz -> 16kHz
func upsampleLinear(src []byte) []byte {
	srcLen := len(src) / 2
	dst := make([]byte, srcLen*4) // 2x samples * 2 bytes/sample

	for i := 0; i < srcLen-1; i++ {
		// Sample atual
		s1 := int16(binary.LittleEndian.Uint16(src[i*2:]))
		// Próximo sample
		s2 := int16(binary.LittleEndian.Uint16(src[(i+1)*2:]))

		// Ponto original
		binary.LittleEndian.PutUint16(dst[i*4:], uint16(s1))

		// Ponto interpolado (média)
		mid := (int32(s1) + int32(s2)) / 2
		binary.LittleEndian.PutUint16(dst[i*4+2:], uint16(mid))
	}

	return dst
}
