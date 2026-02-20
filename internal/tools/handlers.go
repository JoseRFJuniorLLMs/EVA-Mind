// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package tools

import (
	"context"
	"eva-mind/internal/brainstem/database"
	"eva-mind/internal/swarm"
	"eva-mind/internal/brainstem/infrastructure/graph"
	"eva-mind/internal/brainstem/infrastructure/vector"
	"eva-mind/internal/brainstem/oauth"
	"eva-mind/internal/brainstem/push"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"eva-mind/internal/cortex/alert"
	"eva-mind/internal/hippocampus/habits"
	"eva-mind/internal/hippocampus/spaced"
	"eva-mind/internal/motor/actions"
	"eva-mind/internal/motor/email"
	"eva-mind/internal/cortex/llm"
	"eva-mind/internal/cortex/skills"
	"eva-mind/internal/motor/browser"
	"eva-mind/internal/motor/cron"
	"eva-mind/internal/motor/filesystem"
	"eva-mind/internal/motor/messaging"
	"eva-mind/internal/motor/sandbox"
	"eva-mind/internal/motor/selfcode"
	"eva-mind/internal/motor/smarthome"
	"eva-mind/internal/motor/telegram"
	"eva-mind/internal/motor/webhooks"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// WebSearchFunc tipo de função para pesquisa web (evita import cycle com cortex/learning)
type WebSearchFunc func(ctx context.Context, topic string) (interface{}, error)

// EmbedFunc tipo de função para gerar embeddings (evita import cycle com knowledge)
type EmbedFunc func(ctx context.Context, text string) ([]float32, error)

// SwarmRouter interface para rotear tool calls ao swarm orchestrator
// Desacopla ToolsHandler de swarm.Orchestrator para evitar dependencia circular
type SwarmRouter interface {
	Route(ctx context.Context, call swarm.ToolCall) (*swarm.ToolResult, error)
}

type ToolsHandler struct {
	db                *database.DB
	pushService       *push.FirebaseService
	emailService      *email.EmailService
	escalationService *alert.EscalationService        // ✅ Escalation Service
	spacedService     *spaced.SpacedRepetitionService // ✅ Spaced Repetition
	habitTracker      *habits.HabitTracker            // ✅ Habit Tracking
	oauthService      *oauth.Service                  // ✅ Google OAuth (token refresh)
	autonomousLearner WebSearchFunc                   // ✅ Web Research
	whatsappToken     string                          // ✅ WhatsApp Meta API token
	whatsappPhoneID   string                          // ✅ WhatsApp Phone Number ID
	telegramService   *telegram.Service               // ✅ Telegram Bot
	filesystemService *filesystem.Service             // ✅ Filesystem Access
	selfcodeService   *selfcode.Service               // ✅ Self-Coding
	mapsAPIKey        string                          // ✅ Google Maps API Key
	sandboxService    *sandbox.Service                // ✅ Code Execution Sandbox
	browserService    *browser.Service                // ✅ Browser Automation
	cronService       *cron.Service                   // ✅ Scheduled Tasks
	llmService        *llm.Service                    // ✅ Multi-LLM (Claude, GPT, DeepSeek)
	slackService      *messaging.SlackService         // ✅ Slack
	discordService    *messaging.DiscordService       // ✅ Discord
	teamsService      *messaging.TeamsService         // ✅ Microsoft Teams
	signalService     *messaging.SignalService        // ✅ Signal
	smartHomeService  *smarthome.Service              // ✅ Smart Home (Home Assistant)
	webhookService    *webhooks.Service               // ✅ Webhooks
	skillsService     *skills.Service                 // ✅ Runtime Skills
	neo4jClient       *graph.Neo4jClient               // ✅ Neo4j geral (porta 7687 — grafo de conhecimento)
	neo4jCoreDriver   neo4j.DriverWithContext         // ✅ Neo4j Core (porta 7688 — memória pessoal EVA)
	qdrantClient      *vector.QdrantClient            // ✅ Qdrant (busca vetorial)
	embedFunc         EmbedFunc                       // ✅ Embedding func (text → vector para Qdrant)
	debugMode         bool                            // ✅ Novas tools só habilitadas em debug
	swarmRouter       SwarmRouter                     // ✅ Bridge para swarm orchestrator (tools sem case no switch)
	NotifyFunc        func(idosoID int64, msgType string, payload interface{})
	BrowserNotifiers  sync.Map // map[int64]func(msgType string, payload interface{}) — browser voice WS sessions
}

func NewToolsHandler(db *database.DB, pushService *push.FirebaseService, emailService *email.EmailService) *ToolsHandler {
	return &ToolsHandler{
		db:           db,
		pushService:  pushService,
		emailService: emailService,
	}
}

// SetEscalationService configura o serviço de escalation
func (h *ToolsHandler) SetEscalationService(svc *alert.EscalationService) {
	h.escalationService = svc
}

// SetSpacedService configura o serviço de spaced repetition
func (h *ToolsHandler) SetSpacedService(svc *spaced.SpacedRepetitionService) {
	h.spacedService = svc
}

// SetHabitTracker configura o serviço de habit tracking
func (h *ToolsHandler) SetHabitTracker(tracker *habits.HabitTracker) {
	h.habitTracker = tracker
}

// SetOAuthService configura o serviço de OAuth para refresh de tokens Google
func (h *ToolsHandler) SetOAuthService(svc *oauth.Service) {
	h.oauthService = svc
}

// SetAutonomousLearner configura o learner para web search
func (h *ToolsHandler) SetAutonomousLearner(learner WebSearchFunc) {
	h.autonomousLearner = learner
}

// SetWhatsAppConfig configura credenciais WhatsApp Meta API
func (h *ToolsHandler) SetWhatsAppConfig(accessToken, phoneNumberID string) {
	h.whatsappToken = accessToken
	h.whatsappPhoneID = phoneNumberID
}

// SetTelegramService configura o serviço Telegram
func (h *ToolsHandler) SetTelegramService(svc *telegram.Service) {
	h.telegramService = svc
}

// SetFilesystemService configura o serviço de filesystem
func (h *ToolsHandler) SetFilesystemService(svc *filesystem.Service) {
	h.filesystemService = svc
}

// SetSelfCodeService configura o serviço de auto-programação
func (h *ToolsHandler) SetSelfCodeService(svc *selfcode.Service) {
	h.selfcodeService = svc
}

// SetMapsAPIKey configura a API key do Google Maps
func (h *ToolsHandler) SetMapsAPIKey(key string) {
	h.mapsAPIKey = key
}

// SetSandboxService configura o serviço de execução de código
func (h *ToolsHandler) SetSandboxService(svc *sandbox.Service) {
	h.sandboxService = svc
}

// SetBrowserService configura o serviço de browser automation
func (h *ToolsHandler) SetBrowserService(svc *browser.Service) {
	h.browserService = svc
}

// SetCronService configura o serviço de tarefas agendadas
func (h *ToolsHandler) SetCronService(svc *cron.Service) {
	h.cronService = svc
}

// SetLLMService configura o serviço multi-LLM
func (h *ToolsHandler) SetLLMService(svc *llm.Service) {
	h.llmService = svc
}

// SetSlackService configura o serviço Slack
func (h *ToolsHandler) SetSlackService(svc *messaging.SlackService) {
	h.slackService = svc
}

// SetDiscordService configura o serviço Discord
func (h *ToolsHandler) SetDiscordService(svc *messaging.DiscordService) {
	h.discordService = svc
}

// SetTeamsService configura o serviço Microsoft Teams
func (h *ToolsHandler) SetTeamsService(svc *messaging.TeamsService) {
	h.teamsService = svc
}

// SetSignalService configura o serviço Signal
func (h *ToolsHandler) SetSignalService(svc *messaging.SignalService) {
	h.signalService = svc
}

// SetSmartHomeService configura o serviço Smart Home
func (h *ToolsHandler) SetSmartHomeService(svc *smarthome.Service) {
	h.smartHomeService = svc
}

// SetWebhookService configura o serviço de webhooks
func (h *ToolsHandler) SetWebhookService(svc *webhooks.Service) {
	h.webhookService = svc
}

// SetSkillsService configura o serviço de skills dinâmicas
func (h *ToolsHandler) SetSkillsService(svc *skills.Service) {
	h.skillsService = svc
}

// SetNeo4jClient configura o client Neo4j geral (porta 7687 — grafo de conhecimento)
func (h *ToolsHandler) SetNeo4jClient(client *graph.Neo4jClient) {
	h.neo4jClient = client
}

// SetQdrantClient configura o client Qdrant para busca vetorial
func (h *ToolsHandler) SetQdrantClient(client *vector.QdrantClient) {
	h.qdrantClient = client
}

// SetEmbedFunc configura a funcao de embeddings (text → vector)
func (h *ToolsHandler) SetEmbedFunc(fn EmbedFunc) {
	h.embedFunc = fn
}

// SetNeo4jCoreDriver configura o driver Neo4j Core (porta 7688 — memória pessoal da EVA)
func (h *ToolsHandler) SetNeo4jCoreDriver(driver neo4j.DriverWithContext) {
	h.neo4jCoreDriver = driver
}

// SetDebugMode habilita/desabilita novas ferramentas (debug only)
func (h *ToolsHandler) SetDebugMode(enabled bool) {
	h.debugMode = enabled
}

func (h *ToolsHandler) SetSwarmRouter(router SwarmRouter) {
	h.swarmRouter = router
}

// productionTools — ferramentas liberadas em producao (acesso a informacao em tempo real)
// Estas tools NAO sao bloqueadas pelo gate debugOnly, mesmo em ENVIRONMENT=production.
var productionTools = map[string]bool{
	// Acesso a informacao em tempo real (pesquisa web, navegacao, google search)
	"web_search": true, "browse_webpage": true, "show_webpage": true,
	"google_search_retrieval": true,
	// Google Services essenciais (Calendar, Drive, Maps — necessarios para idosos)
	"send_email": true, "manage_calendar_event": true, "save_to_drive": true,
	"find_nearby_places": true, "search_videos": true, "play_music": true, "play_video": true,
	// Messaging (escalacao de emergencia depende destes)
	"send_whatsapp": true, "send_telegram": true,
	// Multi-LLM (segunda opiniao medica)
	"ask_llm": true,
	// Tarefas agendadas (lembretes de medicacao)
	"create_scheduled_task": true, "list_scheduled_tasks": true, "cancel_scheduled_task": true,
	// MCP Bridge — memoria e identidade (leitura segura)
	"mcp_remember": true, "mcp_recall": true, "mcp_get_identity": true, "mcp_learn_topic": true,
}

// debugOnlyTools — tools que so funcionam em modo debug (acoes destrutivas/perigosas)
var debugOnlyTools = map[string]bool{
	// Filesystem (leitura/escrita arbitraria)
	"read_file": true, "write_file": true, "list_files": true, "search_files": true,
	// Self-Coding (EVA edita seu proprio codigo)
	"edit_my_code": true, "create_branch": true, "commit_code": true,
	"run_tests": true, "get_code_diff": true,
	// Database queries diretas (risco de dados sensiveis)
	"query_postgresql": true, "query_neo4j": true, "query_qdrant": true, "query_nietzsche": true,
	// Code Execution Sandbox
	"execute_code": true,
	// Browser Automation (form filling, extraction — risco de abuso)
	"browser_navigate": true, "browser_fill_form": true, "browser_extract": true,
	// Messaging corporativo (Slack/Discord/Teams/Signal — evitar spam acidental)
	"send_slack": true, "send_discord": true, "send_teams": true, "send_signal": true,
	// Smart Home (controle fisico)
	"smart_home_control": true, "smart_home_status": true,
	// Webhooks (chamadas externas arbitrarias)
	"create_webhook": true, "list_webhooks": true, "trigger_webhook": true,
	// Skills dinamicas (execucao arbitraria)
	"create_skill": true, "list_skills": true, "execute_skill": true, "delete_skill": true,
	// MCP Bridge — escrita/edicao de codigo (perigoso)
	"mcp_teach_eva": true, "mcp_query_neo4j_core": true, "mcp_read_source": true, "mcp_edit_source": true,
}

// getGoogleAccessToken obtém um access token válido para Google APIs
func (h *ToolsHandler) getGoogleAccessToken(idosoID int64) (string, error) {
	refreshToken, accessToken, expiry, err := h.db.GetGoogleTokens(idosoID)
	if err != nil {
		return "", fmt.Errorf("Google não conectado: %v", err)
	}
	if refreshToken == "" && accessToken == "" {
		return "", fmt.Errorf("conta Google não vinculada — peça ao cuidador para conectar")
	}

	// Token ainda válido
	if accessToken != "" && time.Now().Before(expiry) {
		return accessToken, nil
	}

	// Token expirado — refresh
	if h.oauthService == nil {
		return "", fmt.Errorf("serviço OAuth não configurado")
	}
	if refreshToken == "" {
		return "", fmt.Errorf("refresh token ausente — reconectar conta Google")
	}

	newToken, err := h.oauthService.RefreshToken(context.Background(), refreshToken)
	if err != nil {
		return "", fmt.Errorf("falha ao renovar token: %v", err)
	}

	// Salvar novos tokens
	if err := h.db.SaveGoogleTokens(idosoID, refreshToken, newToken.AccessToken, newToken.Expiry); err != nil {
		log.Printf("⚠️ [OAUTH] Erro ao salvar tokens renovados: %v", err)
	}

	return newToken.AccessToken, nil
}

// ExecuteTool dispatches the tool call to the appropriate handler
func (h *ToolsHandler) ExecuteTool(name string, args map[string]interface{}, idosoID int64) (map[string]interface{}, error) {
	log.Printf("🛠️ [TOOLS] Executando tool: %s para Idoso %d", name, idosoID)

	// 🔓 TODAS as tools liberadas para TODOS os modos (producao + debug)
	// Gate removido: todas as 150+ tools ativas independente de ENVIRONMENT
	log.Printf("🔓 [TOOLS] Tool '%s' liberada (all-access mode)", name)

	// 🚦 Rate limiting
	if err := checkRateLimit(name, idosoID); err != nil {
		log.Printf("🚦 [TOOLS] Rate limit: %s para Idoso %d — %v", name, idosoID, err)
		return map[string]interface{}{
			"status":  "rate_limited",
			"message": err.Error(),
		}, nil
	}

	switch name {
	case "alert_family":
		reason, _ := args["reason"].(string)
		severity, _ := args["severity"].(string)
		if severity == "" {
			severity = "alta"
		}
		err := actions.AlertFamilyWithSeverity(h.db.Conn, h.pushService, h.emailService, idosoID, reason, severity)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}

		// ✅ NOVO: Trigger Escalation Service para alertas críticos
		if h.escalationService != nil && (severity == "critica" || severity == "alta") {
			go func(eid int64, msg, sev string) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()

				priority := alert.PriorityHigh
				if sev == "critica" {
					priority = alert.PriorityCritical
				}

				// Buscar contatos do idoso
				contacts, err := h.escalationService.GetContactsForElder(eid)
				if err != nil || len(contacts) == 0 {
					log.Printf("⚠️ [ESCALATION] Sem contatos para idoso %d: %v", eid, err)
					return
				}

				// Buscar nome do idoso
				var elderName string
				h.db.Conn.QueryRow("SELECT nome FROM idosos WHERE id = $1", eid).Scan(&elderName)
				if elderName == "" {
					elderName = fmt.Sprintf("Paciente %d", eid)
				}

				result := h.escalationService.SendEmergencyAlert(ctx, elderName, msg, priority, contacts)
				if result.Acknowledged {
					log.Printf("✅ [ESCALATION] Alerta reconhecido: %s", msg)
				} else {
					log.Printf("⚠️ [ESCALATION] Alerta não reconhecido após escalação: %s", msg)
				}
			}(idosoID, reason, severity)
		}

		return map[string]interface{}{"status": "sucesso", "alerta": reason}, nil

	case "confirm_medication":
		medicationName, _ := args["medication_name"].(string)
		err := actions.ConfirmMedication(h.db.Conn, h.pushService, idosoID, medicationName)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}
		return map[string]interface{}{"status": "sucesso", "medicamento": medicationName}, nil

	case "pending_schedule":
		// Armazena agendamento pendente e retorna instrução para EVA pedir confirmação
		timestamp, _ := args["timestamp"].(string)
		tipo, _ := args["type"].(string)
		description, _ := args["description"].(string)

		// 🛡️ SAFETY CHECK: Verificar interações medicamentosas ANTES de agendar
		if tipo == "medicamento" || tipo == "remedio" || tipo == "medication" {
			interacoes, err := actions.CheckMedicationInteractions(h.db.Conn, idosoID, description)
			if err != nil {
				log.Printf("⚠️ [SAFETY] Erro ao verificar interações: %v", err)
				// Continua mesmo com erro - melhor agendar do que bloquear por falha técnica
			} else if len(interacoes) > 0 {
				// 🚨 BLOQUEAR AGENDAMENTO - Interação perigosa detectada
				warning := actions.FormatInteractionWarning(interacoes)
				log.Printf("⛔ [SAFETY] AGENDAMENTO BLOQUEADO: %s", warning)

				// Notificar cuidador/família sobre tentativa bloqueada
				alertMsg := fmt.Sprintf("EVA bloqueou agendamento de %s para idoso %d devido a interação medicamentosa: %s",
					description, idosoID, interacoes[0].NivelPerigo)
				go actions.AlertFamilyWithSeverity(h.db.Conn, h.pushService, h.emailService, idosoID, alertMsg, "alta")

				return map[string]interface{}{
					"status":       "bloqueado",
					"blocked":      true,
					"reason":       "interacao_medicamentosa",
					"nivel_perigo": interacoes[0].NivelPerigo,
					"warning":      warning,
					"message":      "BLOQUEADO: Diga ao usuário que não pode agendar este medicamento e explique o motivo",
				}, nil
			}
		}

		confirmMsg := actions.StorePendingSchedule(idosoID, timestamp, tipo, description)
		return map[string]interface{}{
			"status":              "aguardando_confirmacao",
			"pending":             true,
			"description":         description,
			"confirmation_prompt": confirmMsg,
			"message":             "Pergunte ao usuário se ele confirma o agendamento antes de prosseguir",
		}, nil

	case "confirm_schedule":
		// Confirma ou cancela agendamento pendente
		confirmed, _ := args["confirmed"].(bool)
		success, desc, err := actions.ConfirmPendingSchedule(h.db.Conn, idosoID, confirmed)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}
		if !success && desc == "" {
			return map[string]interface{}{
				"status":  "nenhum_pendente",
				"message": "Não há agendamento pendente para confirmar",
			}, nil
		}
		if confirmed && success {
			return map[string]interface{}{
				"status":      "agendado",
				"description": desc,
				"message":     "Agendamento confirmado e salvo com sucesso",
			}, nil
		}
		return map[string]interface{}{
			"status":      "cancelado",
			"description": desc,
			"message":     "Agendamento cancelado pelo usuário",
		}, nil

	case "schedule_appointment":
		// Agendamento direto (sem confirmação) - mantido para compatibilidade
		timestamp, _ := args["timestamp"].(string)
		tipo, _ := args["type"].(string)
		description, _ := args["description"].(string)
		err := actions.ScheduleAppointment(h.db.Conn, idosoID, timestamp, tipo, description)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}
		return map[string]interface{}{"status": "sucesso", "agendamento": description}, nil

	// ============================================================================
	// ⏰ ALARMES E DESPERTADOR
	// ============================================================================

	case "set_alarm":
		timeStr, _ := args["time"].(string)
		label, _ := args["label"].(string)
		if label == "" {
			label = "Alarme"
		}

		// Parse repeat_days
		var repeatDays []string
		if rd, ok := args["repeat_days"].([]interface{}); ok {
			for _, d := range rd {
				if ds, ok := d.(string); ok {
					repeatDays = append(repeatDays, ds)
				}
			}
		}

		alarmID, err := h.createAlarm(idosoID, timeStr, label, repeatDays)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}

		// Notificar app para configurar alarme local
		if h.NotifyFunc != nil {
			h.NotifyFunc(idosoID, "alarm_set", map[string]interface{}{
				"alarm_id":    alarmID,
				"time":        timeStr,
				"label":       label,
				"repeat_days": repeatDays,
			})
		}

		repeatMsg := "apenas uma vez"
		if len(repeatDays) == 7 {
			repeatMsg = "todos os dias"
		} else if len(repeatDays) > 0 {
			repeatMsg = fmt.Sprintf("nos dias: %v", repeatDays)
		}

		return map[string]interface{}{
			"status":      "sucesso",
			"alarm_id":    alarmID,
			"time":        timeStr,
			"label":       label,
			"repeat_days": repeatDays,
			"message":     fmt.Sprintf("Alarme configurado para %s (%s)", timeStr, repeatMsg),
		}, nil

	case "cancel_alarm":
		alarmID, _ := args["alarm_id"].(string)
		if alarmID == "" {
			alarmID = "all"
		}

		count, err := h.cancelAlarm(idosoID, alarmID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}

		// Notificar app para cancelar alarme local
		if h.NotifyFunc != nil {
			h.NotifyFunc(idosoID, "alarm_cancelled", map[string]interface{}{
				"alarm_id": alarmID,
				"count":    count,
			})
		}

		if alarmID == "all" {
			return map[string]interface{}{
				"status":  "sucesso",
				"count":   count,
				"message": fmt.Sprintf("%d alarme(s) cancelado(s)", count),
			}, nil
		}
		return map[string]interface{}{
			"status":  "sucesso",
			"message": "Alarme cancelado",
		}, nil

	case "list_alarms":
		alarms, err := h.listAlarms(idosoID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}

		if len(alarms) == 0 {
			return map[string]interface{}{
				"status":  "sucesso",
				"alarms":  []interface{}{},
				"message": "Você não tem alarmes configurados",
			}, nil
		}

		return map[string]interface{}{
			"status":  "sucesso",
			"alarms":  alarms,
			"count":   len(alarms),
			"message": fmt.Sprintf("Você tem %d alarme(s) configurado(s)", len(alarms)),
		}, nil

	case "call_family_webrtc", "call_doctor_webrtc", "call_caregiver_webrtc", "call_central_webrtc":
		// Buscar CPF do contato baseado no tipo de chamada
		targetCPF, targetName, err := h.getCallTargetCPF(idosoID, name)
		if err != nil {
			log.Printf("⚠️ [CALL] Erro ao buscar contato: %v", err)
			return map[string]interface{}{"error": fmt.Sprintf("Não encontrei contato para %s", name)}, nil
		}

		if h.NotifyFunc != nil {
			h.NotifyFunc(idosoID, "initiate_call", map[string]interface{}{
				"target":      name,
				"target_cpf":  targetCPF,
				"target_name": targetName,
			})
			return map[string]interface{}{
				"status":      "iniciando chamada",
				"alvo":        name,
				"target_name": targetName,
			}, nil
		}
		return map[string]interface{}{"error": "serviço de sinalização não disponível"}, nil

	case "google_search_retrieval":
		query, _ := args["query"].(string)
		if query == "" {
			return map[string]interface{}{"error": "Informe o que deseja pesquisar"}, nil
		}

		// Usar AutonomousLearner para busca REAL via Gemini + Google Search grounding
		if h.autonomousLearner != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			result, err := h.autonomousLearner(ctx, query)
			if err != nil {
				log.Printf("⚠️ [GOOGLE_SEARCH] Erro na busca real: %v", err)
				return map[string]interface{}{
					"status": "erro",
					"query":  query,
					"error":  fmt.Sprintf("Pesquisa falhou: %v", err),
				}, nil
			}

			log.Printf("✅ [GOOGLE_SEARCH] Busca real concluida para: %s", query)
			return map[string]interface{}{
				"status":  "sucesso",
				"query":   query,
				"result":  result,
				"source":  "gemini_google_search_grounding",
				"message": fmt.Sprintf("Resultados reais da pesquisa sobre '%s'", query),
			}, nil
		}

		log.Printf("⚠️ [GOOGLE_SEARCH] AutonomousLearner nao configurado — retornando fallback")
		return map[string]interface{}{
			"status":  "indisponivel",
			"query":   query,
			"message": "Serviço de pesquisa web não está configurado neste momento",
		}, nil

	case "get_vitals":
		// Extrair argumentos
		vitalsType, _ := args["vitals_type"].(string)
		limitFloat, _ := args["limit"].(float64) // JSON numbers are float64
		limit := int(limitFloat)
		if limit == 0 {
			limit = 3
		}
		return h.handleGetVitals(idosoID, vitalsType, limit)

	case "get_agendamentos":
		limitFloat, _ := args["limit"].(float64)
		limit := int(limitFloat)
		if limit == 0 {
			limit = 5
		}
		return h.handleGetAgendamentos(idosoID, limit)

	case "scan_medication_visual":
		reason, _ := args["reason"].(string)
		timeOfDay, _ := args["time_of_day"].(string)
		return h.handleScanMedicationVisual(idosoID, reason, timeOfDay)

	case "analyze_voice_prosody":
		analysisType, _ := args["analysis_type"].(string)
		audioSegmentFloat, _ := args["audio_segment_seconds"].(float64)
		audioSegment := int(audioSegmentFloat)
		if audioSegment == 0 {
			audioSegment = 30
		}
		return h.handleAnalyzeVoiceProsody(idosoID, analysisType, audioSegment)

	case "apply_phq9":
		startAssessment, _ := args["start_assessment"].(bool)
		return h.handleApplyPHQ9(idosoID, startAssessment)

	case "apply_gad7":
		startAssessment, _ := args["start_assessment"].(bool)
		return h.handleApplyGAD7(idosoID, startAssessment)

	case "apply_cssrs":
		triggerPhrase, _ := args["trigger_phrase"].(string)
		startAssessment, _ := args["start_assessment"].(bool)
		return h.handleApplyCSSRS(idosoID, triggerPhrase, startAssessment)

	case "submit_phq9_response":
		sessionID, _ := args["session_id"].(string)
		questionNumber, _ := args["question_number"].(float64)
		responseValue, _ := args["response_value"].(float64)
		responseText, _ := args["response_text"].(string)
		return h.handleSubmitPHQ9Response(idosoID, sessionID, int(questionNumber), int(responseValue), responseText)

	case "submit_gad7_response":
		sessionID, _ := args["session_id"].(string)
		questionNumber, _ := args["question_number"].(float64)
		responseValue, _ := args["response_value"].(float64)
		responseText, _ := args["response_text"].(string)
		return h.handleSubmitGAD7Response(idosoID, sessionID, int(questionNumber), int(responseValue), responseText)

	case "submit_cssrs_response":
		sessionID, _ := args["session_id"].(string)
		questionNumber, _ := args["question_number"].(float64)
		responseValue, _ := args["response_value"].(float64)
		responseText, _ := args["response_text"].(string)
		return h.handleSubmitCSSRSResponse(idosoID, sessionID, int(questionNumber), int(responseValue), responseText)

	// ========================================
	// ENTERTAINMENT TOOLS (30 ferramentas)
	// ========================================

	// --- Música e Áudio (6) ---
	case "play_nostalgic_music":
		return h.handlePlayNostalgicMusic(idosoID, args)

	case "play_radio_station":
		return h.handlePlayRadioStation(idosoID, args)

	case "nature_sounds":
		return h.handleNatureSounds(idosoID, args)

	case "audiobook_reader":
		return h.handleAudiobookReader(idosoID, args)

	case "podcast_player":
		return h.handlePodcastPlayer(idosoID, args)

	case "religious_content":
		return h.handleReligiousContent(idosoID, args)

	// --- Jogos Cognitivos (6) ---
	case "play_trivia_game":
		return h.handlePlayTriviaGame(idosoID, args)

	case "memory_game":
		return h.handleMemoryGame(idosoID, args)

	case "word_association":
		return h.handleWordAssociation(idosoID, args)

	case "brain_training":
		return h.handleBrainTraining(idosoID, args)

	case "complete_the_lyrics":
		return h.handleCompleteTheLyrics(idosoID, args)

	case "riddles_and_jokes":
		return h.handleRiddlesAndJokes(idosoID, args)

	// --- Histórias e Narrativas (5) ---
	case "story_generator":
		return h.handleStoryGenerator(idosoID, args)

	case "reminiscence_therapy":
		return h.handleReminiscenceTherapy(idosoID, args)

	case "biography_writer":
		return h.handleBiographyWriter(idosoID, args)

	case "read_newspaper":
		return h.handleReadNewspaper(idosoID, args)

	case "daily_horoscope":
		return h.handleDailyHoroscope(idosoID, args)

	// --- Bem-estar e Saúde (6) ---
	case "guided_meditation":
		return h.handleGuidedMeditation(idosoID, args)

	case "breathing_exercises":
		return h.handleBreathingExercises(idosoID, args)

	case "wim_hof_breathing":
		return h.handleWimHofBreathing(idosoID, args)

	case "pomodoro_timer":
		return h.handlePomodoroTimer(idosoID, args)

	case "chair_exercises":
		return h.handleChairExercises(idosoID, args)

	case "sleep_stories":
		return h.handleSleepStories(idosoID, args)

	case "gratitude_journal":
		return h.handleGratitudeJournal(idosoID, args)

	case "motivational_quotes":
		return h.handleMotivationalQuotes(idosoID, args)

	// --- Social e Família (4) ---
	case "voice_capsule":
		return h.handleVoiceCapsule(idosoID, args)

	case "birthday_reminder":
		return h.handleBirthdayReminder(idosoID, args)

	case "family_tree_explorer":
		return h.handleFamilyTreeExplorer(idosoID, args)

	case "photo_slideshow":
		return h.handlePhotoSlideshow(idosoID, args)

	// --- Utilidades Diárias (3) ---
	case "weather_chat":
		return h.handleWeatherChat(idosoID, args)

	case "cooking_recipes":
		return h.handleCookingRecipes(idosoID, args)

	case "voice_diary":
		return h.handleVoiceDiary(idosoID, args)

	// --- Aprendizagem de Idiomas ---
	case "learn_new_language":
		return h.handleLearnNewLanguage(idosoID, args)

	// --- Habit Tracking (Log de Hábitos) ---
	case "log_habit":
		return h.handleLogHabit(idosoID, args)

	case "log_water":
		return h.handleLogWater(idosoID, args)

	case "habit_stats":
		return h.handleHabitStats(idosoID, args)

	case "habit_summary":
		return h.handleHabitSummary(idosoID, args)

	// --- Pesquisa de Locais e Mapas ---
	case "search_places":
		return h.handleSearchPlaces(idosoID, args)

	case "get_directions":
		return h.handleGetDirections(idosoID, args)

	case "nearby_transport":
		return h.handleNearbyTransport(idosoID, args)

	// --- Abrir Aplicativos ---
	case "open_app":
		return h.handleOpenApp(idosoID, args)

	// --- Spaced Repetition (Reforço de Memória) ---
	case "remember_this":
		return h.handleRememberThis(idosoID, args)

	case "review_memory":
		return h.handleReviewMemory(idosoID, args)

	case "list_memories":
		return h.handleListMemories(idosoID, args)

	case "pause_memory":
		return h.handlePauseMemory(idosoID, args)

	case "memory_stats":
		return h.handleMemoryStats(idosoID, args)

	// --- GTD (Getting Things Done) - Captura de Tarefas ---
	case "capture_task":
		return h.handleCaptureTask(idosoID, args)

	case "list_tasks":
		return h.handleListTasks(idosoID, args)

	case "complete_task":
		return h.handleCompleteTask(idosoID, args)

	case "clarify_task":
		return h.handleClarifyTask(idosoID, args)

	case "weekly_review":
		return h.handleWeeklyReview(idosoID, args)

	case "change_user_directive":
		directiveType, _ := args["directive_type"].(string)
		newValue, _ := args["new_value"].(string)
		err := h.UpdateUserDirective(idosoID, directiveType, newValue)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}
		return map[string]interface{}{
			"status":  "sucesso",
			"message": fmt.Sprintf("Diretiva '%s' alterada para '%s'", directiveType, newValue),
		}, nil

	// ============================================================================
	// 📧 GOOGLE SERVICES (Motor Layer — Real APIs)
	// ============================================================================

	case "send_email":
		return h.handleSendEmail(idosoID, args)

	case "search_videos":
		return h.handleSearchVideos(idosoID, args)

	case "play_music":
		return h.handlePlayMusic(idosoID, args)

	case "send_whatsapp":
		return h.handleSendWhatsApp(idosoID, args)

	case "manage_calendar_event":
		return h.handleManageCalendar(idosoID, args)

	case "save_to_drive":
		return h.handleSaveToDrive(idosoID, args)

	case "find_nearby_places":
		return h.handleFindNearbyPlaces(idosoID, args)

	// ============================================================================
	// 📱 MESSAGING (Telegram)
	// ============================================================================

	case "send_telegram":
		return h.handleSendTelegram(idosoID, args)

	// ============================================================================
	// 📂 FILESYSTEM
	// ============================================================================

	case "read_file":
		return h.handleReadFile(idosoID, args)

	case "write_file":
		return h.handleWriteFile(idosoID, args)

	case "list_files":
		return h.handleListFiles(idosoID, args)

	case "search_files":
		return h.handleSearchFiles(idosoID, args)

	// ============================================================================
	// 🌐 WEB BROWSING
	// ============================================================================

	case "web_search":
		return h.handleWebSearch(idosoID, args)

	case "browse_webpage":
		return h.handleBrowseWebpage(idosoID, args)

	// ============================================================================
	// 📺 VIDEO & WEB DISPLAY
	// ============================================================================

	case "play_video":
		return h.handlePlayVideo(idosoID, args)

	case "show_webpage":
		return h.handleShowWebpage(idosoID, args)

	// ============================================================================
	// 💻 SELF-CODING (OpenClaw-style)
	// ============================================================================

	case "edit_my_code":
		return h.handleEditMyCode(idosoID, args)

	case "create_branch":
		return h.handleCreateBranch(idosoID, args)

	case "commit_code":
		return h.handleCommitCode(idosoID, args)

	case "run_tests":
		return h.handleRunTests(idosoID, args)

	case "get_code_diff":
		return h.handleGetCodeDiff(idosoID, args)

	// ============================================================================
	// 🗄️ ACESSO DIRETO A BASES DE DADOS
	// ============================================================================

	case "query_postgresql":
		return h.handleQueryPostgreSQL(idosoID, args)

	case "query_neo4j":
		return h.handleQueryNeo4j(idosoID, args)

	case "query_qdrant":
		return h.handleQueryQdrant(idosoID, args)

	case "query_nietzsche":
		return h.handleQueryNietzsche(idosoID, args)

	// ============================================================================
	// 🖥️ SANDBOX — Execução de Código (Bash, Python, Node)
	// ============================================================================

	case "execute_code":
		return h.handleExecuteCode(idosoID, args)

	// ============================================================================
	// 🌐 BROWSER AUTOMATION
	// ============================================================================

	case "browser_navigate":
		return h.handleBrowserNavigate(idosoID, args)

	case "browser_fill_form":
		return h.handleBrowserFillForm(idosoID, args)

	case "browser_extract":
		return h.handleBrowserExtract(idosoID, args)

	// ============================================================================
	// ⏰ CRON / SCHEDULED TASKS
	// ============================================================================

	case "create_scheduled_task":
		return h.handleCreateScheduledTask(idosoID, args)

	case "list_scheduled_tasks":
		return h.handleListScheduledTasks(idosoID, args)

	case "cancel_scheduled_task":
		return h.handleCancelScheduledTask(idosoID, args)

	// ============================================================================
	// 🤖 MULTI-LLM (Claude, GPT, DeepSeek)
	// ============================================================================

	case "ask_llm":
		return h.handleAskLLM(idosoID, args)

	// ============================================================================
	// 💬 MESSAGING CHANNELS (Slack, Discord, Teams, Signal)
	// ============================================================================

	case "send_slack":
		return h.handleSendSlack(idosoID, args)

	case "send_discord":
		return h.handleSendDiscord(idosoID, args)

	case "send_teams":
		return h.handleSendTeams(idosoID, args)

	case "send_signal":
		return h.handleSendSignal(idosoID, args)

	// ============================================================================
	// 🏠 SMART HOME (Home Assistant IoT)
	// ============================================================================

	case "smart_home_control":
		return h.handleSmartHomeControl(idosoID, args)

	case "smart_home_status":
		return h.handleSmartHomeStatus(idosoID, args)

	// ============================================================================
	// 🔗 WEBHOOKS
	// ============================================================================

	case "create_webhook":
		return h.handleCreateWebhook(idosoID, args)

	case "list_webhooks":
		return h.handleListWebhooks(idosoID, args)

	case "trigger_webhook":
		return h.handleTriggerWebhook(idosoID, args)

	// ============================================================================
	// 🧩 SKILLS (Self-Improving Runtime)
	// ============================================================================

	case "create_skill":
		return h.handleCreateSkill(idosoID, args)

	case "list_skills":
		return h.handleListSkills(idosoID, args)

	case "execute_skill":
		return h.handleExecuteSkill(idosoID, args)

	case "delete_skill":
		return h.handleDeleteSkill(idosoID, args)

	// ============================================================================
	// 🔌 MCP BRIDGE — Tools expostas via MCP Server (Claude Code)
	// ============================================================================

	// ============================================================================
	// 🖥️ UI CONTROL (forward to browser WebSocket as ui_action)
	// ============================================================================

	case "control_ui":
		action, _ := args["action"].(string)
		result := map[string]interface{}{
			"success":    true,
			"message":    fmt.Sprintf("UI action '%s' enviada ao browser", action),
			"ui_action":  true,
			"action":     action,
			"target":     args["target"],
			"mode":       args["mode"],
			"url":        args["url"],
			"message_ui": args["message"],
		}
		return result, nil

	case "mcp_remember":
		return h.handleMCPRemember(idosoID, args)

	case "mcp_recall":
		return h.handleMCPRecall(idosoID, args)

	case "mcp_teach_eva":
		return h.handleMCPTeachEva(idosoID, args)

	case "mcp_get_identity":
		return h.handleMCPGetIdentity(idosoID, args)

	case "mcp_learn_topic":
		return h.handleMCPLearnTopic(idosoID, args)

	case "mcp_query_neo4j_core":
		return h.handleMCPQueryNeo4jCore(idosoID, args)

	case "mcp_read_source":
		return h.handleMCPReadSource(idosoID, args)

	case "mcp_edit_source":
		return h.handleMCPEditSource(idosoID, args)

	default:
		// Bridge para swarm orchestrator — tools registradas nos swarm agents
		// mas sem case explicito no switch (ex: open_camera_analysis, change_voice, etc)
		if h.swarmRouter != nil {
			swarmCall := swarm.ToolCall{
				Name:   name,
				Args:   args,
				UserID: idosoID,
			}
			swarmResult, swarmErr := h.swarmRouter.Route(context.Background(), swarmCall)
			if swarmErr == nil && swarmResult != nil {
				log.Printf("[SWARM-BRIDGE] Tool '%s' executada via swarm orchestrator", name)
				result := map[string]interface{}{
					"success": swarmResult.Success,
					"message": swarmResult.Message,
				}
				if swarmResult.Data != nil {
					result["data"] = swarmResult.Data
				}
				return result, nil
			}
			if swarmErr != nil {
				log.Printf("[SWARM-BRIDGE] Swarm falhou para '%s': %v", name, swarmErr)
			}
		}
		return nil, fmt.Errorf("ferramenta desconhecida: %s", name)
	}
}

func (h *ToolsHandler) handleGetVitals(idosoID int64, tipo string, limit int) (map[string]interface{}, error) {
	// Mapear nome da tool para nome no banco se necessário
	// 'pressao_arterial', 'glicemia', etc já devem bater ou fazer mapeamento

	vitals, err := h.db.GetRecentVitalSigns(idosoID, tipo, limit)
	if err != nil {
		log.Printf("❌ [TOOLS] Erro ao buscar vitals: %v", err)
		return map[string]interface{}{"error": "Falha ao buscar sinais vitais"}, nil // Retornar erro JSON para o modelo saber
	}

	if len(vitals) == 0 {
		return map[string]interface{}{
			"result": fmt.Sprintf("Nenhum registro recente de %s encontrado.", tipo),
		}, nil
	}

	// Converter para formato simples
	var resultList []map[string]interface{}
	for _, v := range vitals {
		resultList = append(resultList, map[string]interface{}{
			"valor":      v.Valor,
			"unidade":    v.Unidade,
			"data":       v.DataMedicao.Format("02/01/2006 15:04"),
			"observacao": v.Observacao,
		})
	}

	return map[string]interface{}{
		"tipo":    tipo,
		"records": resultList,
	}, nil
}

func (h *ToolsHandler) handleGetAgendamentos(idosoID int64, limit int) (map[string]interface{}, error) {
	agendamentos, err := h.db.GetPendingAgendamentosByIdoso(idosoID, limit)
	if err != nil {
		return map[string]interface{}{"error": "Erro ao buscar agendamentos"}, nil
	}

	var resultList []map[string]interface{}
	for _, a := range agendamentos {
		resultList = append(resultList, map[string]interface{}{
			"tipo":     a.Tipo,
			"data":     a.DataHoraAgendada.Format("02/01 15:04"),
			"status":   a.Status,
			"detalhes": a.DadosTarefa,
		})
	}

	if len(resultList) == 0 {
		return map[string]interface{}{
			"result": "Nenhum agendamento futuro encontrado.",
		}, nil
	}

	return map[string]interface{}{
		"agendamentos": resultList,
	}, nil
}

func (h *ToolsHandler) handleScanMedicationVisual(idosoID int64, reason string, timeOfDay string) (map[string]interface{}, error) {
	log.Printf("🔍 [MEDICATION SCANNER] Iniciando scan para Idoso %d (motivo: %s, horário: %s)", idosoID, reason, timeOfDay)

	// 1. Buscar medicamentos candidatos do banco baseado no horário
	candidateMeds, err := h.db.GetMedicationsBySchedule(idosoID, timeOfDay)
	if err != nil {
		log.Printf("❌ [MEDICATION SCANNER] Erro ao buscar medicamentos: %v", err)
		return map[string]interface{}{"error": "Falha ao buscar medicamentos programados"}, nil
	}

	// Se não encontrou medicamentos para esse horário, buscar todos ativos
	if len(candidateMeds) == 0 {
		log.Printf("⚠️ [MEDICATION SCANNER] Nenhum medicamento programado para %s, buscando todos ativos", timeOfDay)
		candidateMeds, err = h.db.GetActiveMedications(idosoID)
		if err != nil {
			return map[string]interface{}{"error": "Falha ao buscar medicamentos ativos"}, nil
		}
	}

	// 2. Preparar payload para enviar ao mobile via WebSocket
	if h.NotifyFunc != nil {
		sessionID := fmt.Sprintf("med-scan-%d-%d", idosoID, time.Now().Unix())

		// Converter medicamentos para formato simples
		var candidateList []map[string]interface{}
		for _, med := range candidateMeds {
			candidateList = append(candidateList, map[string]interface{}{
				"id":           med.ID,
				"name":         med.Nome,
				"dosage":       med.Dosagem,
				"color":        med.CorEmbalagem,
				"manufacturer": med.Fabricante,
			})
		}

		// Sinalizar mobile para abrir scanner
		h.NotifyFunc(idosoID, "open_medication_scanner", map[string]interface{}{
			"session_id":            sessionID,
			"candidate_medications": candidateList,
			"instructions":          "Aponte a câmera para os frascos de medicamento",
			"timeout":               60,
			"reason":                reason,
		})

		log.Printf("✅ [MEDICATION SCANNER] Scanner iniciado. Session ID: %s, Candidatos: %d", sessionID, len(candidateList))

		return map[string]interface{}{
			"status":           "scanner_iniciado",
			"session_id":       sessionID,
			"candidates_count": len(candidateList),
			"reason":           reason,
		}, nil
	}

	return map[string]interface{}{"error": "Serviço de sinalização WebSocket não disponível"}, nil
}

func (h *ToolsHandler) handleAnalyzeVoiceProsody(idosoID int64, analysisType string, audioSegment int) (map[string]interface{}, error) {
	log.Printf("🎤 [VOICE PROSODY] Iniciando análise para Idoso %d (tipo: %s, duração: %d seg)", idosoID, analysisType, audioSegment)

	// Sinalizar mobile para capturar áudio via WebSocket
	if h.NotifyFunc != nil {
		sessionID := fmt.Sprintf("voice-prosody-%d-%d", idosoID, time.Now().Unix())

		h.NotifyFunc(idosoID, "start_voice_recording", map[string]interface{}{
			"session_id":    sessionID,
			"analysis_type": analysisType,
			"duration":      audioSegment,
			"instructions":  "Vou analisar sua voz. Por favor, continue conversando naturalmente.",
		})

		log.Printf("✅ [VOICE PROSODY] Captura de áudio iniciada. Session ID: %s", sessionID)

		return map[string]interface{}{
			"status":        "recording_started",
			"session_id":    sessionID,
			"analysis_type": analysisType,
			"duration":      audioSegment,
			"message":       fmt.Sprintf("Gravação de voz iniciada para análise de %s", analysisType),
		}, nil
	}

	return map[string]interface{}{"error": "Serviço de sinalização WebSocket não disponível"}, nil
}

func (h *ToolsHandler) handleApplyPHQ9(idosoID int64, startAssessment bool) (map[string]interface{}, error) {
	log.Printf("📋 [PHQ-9] Iniciando aplicação da escala PHQ-9 para Idoso %d", idosoID)

	if !startAssessment {
		return map[string]interface{}{
			"error": "start_assessment deve ser true para iniciar a avaliação",
		}, nil
	}

	// Criar sessão de avaliação no banco
	sessionID := fmt.Sprintf("phq9-%d-%d", idosoID, time.Now().Unix())

	query := `
		INSERT INTO clinical_assessments (
			patient_id, assessment_type, session_id, status, created_at
		) VALUES ($1, 'PHQ-9', $2, 'in_progress', NOW())
		RETURNING id
	`

	var assessmentID int64
	err := h.db.Conn.QueryRow(query, idosoID, sessionID).Scan(&assessmentID)
	if err != nil {
		log.Printf("❌ [PHQ-9] Erro ao criar sessão: %v", err)
		return map[string]interface{}{"error": "Erro ao iniciar avaliação"}, nil
	}

	log.Printf("✅ [PHQ-9] Sessão criada. Assessment ID: %d, Session ID: %s", assessmentID, sessionID)

	// Retornar primeira pergunta
	return map[string]interface{}{
		"status":          "assessment_started",
		"session_id":      sessionID,
		"assessment_id":   assessmentID,
		"scale":           "PHQ-9",
		"total_questions": 9,
		"message":         "Vou fazer algumas perguntas para entender melhor como você tem se sentido nas últimas 2 semanas. Não há respostas certas ou erradas.",
		"first_question": map[string]interface{}{
			"number": 1,
			"text":   "Pouco interesse ou prazer em fazer as coisas?",
			"options": []string{
				"Nenhuma vez",
				"Vários dias",
				"Mais da metade dos dias",
				"Quase todos os dias",
			},
		},
	}, nil
}

func (h *ToolsHandler) handleApplyGAD7(idosoID int64, startAssessment bool) (map[string]interface{}, error) {
	log.Printf("📋 [GAD-7] Iniciando aplicação da escala GAD-7 para Idoso %d", idosoID)

	if !startAssessment {
		return map[string]interface{}{
			"error": "start_assessment deve ser true para iniciar a avaliação",
		}, nil
	}

	// Criar sessão de avaliação no banco
	sessionID := fmt.Sprintf("gad7-%d-%d", idosoID, time.Now().Unix())

	query := `
		INSERT INTO clinical_assessments (
			patient_id, assessment_type, session_id, status, created_at
		) VALUES ($1, 'GAD-7', $2, 'in_progress', NOW())
		RETURNING id
	`

	var assessmentID int64
	err := h.db.Conn.QueryRow(query, idosoID, sessionID).Scan(&assessmentID)
	if err != nil {
		log.Printf("❌ [GAD-7] Erro ao criar sessão: %v", err)
		return map[string]interface{}{"error": "Erro ao iniciar avaliação"}, nil
	}

	log.Printf("✅ [GAD-7] Sessão criada. Assessment ID: %d, Session ID: %s", assessmentID, sessionID)

	// Retornar primeira pergunta
	return map[string]interface{}{
		"status":          "assessment_started",
		"session_id":      sessionID,
		"assessment_id":   assessmentID,
		"scale":           "GAD-7",
		"total_questions": 7,
		"message":         "Vou fazer algumas perguntas sobre ansiedade e nervosismo nas últimas 2 semanas.",
		"first_question": map[string]interface{}{
			"number": 1,
			"text":   "Sentir-se nervoso(a), ansioso(a) ou muito tenso(a)?",
			"options": []string{
				"Nenhuma vez",
				"Vários dias",
				"Mais da metade dos dias",
				"Quase todos os dias",
			},
		},
	}, nil
}

func (h *ToolsHandler) handleApplyCSSRS(idosoID int64, triggerPhrase string, startAssessment bool) (map[string]interface{}, error) {
	log.Printf("🚨 [C-SSRS] ALERTA CRÍTICO - Avaliação de risco suicida iniciada para Idoso %d. Trigger: '%s'", idosoID, triggerPhrase)

	if !startAssessment {
		return map[string]interface{}{
			"error": "start_assessment deve ser true para iniciar a avaliação",
		}, nil
	}

	// Criar sessão CRÍTICA de avaliação no banco
	sessionID := fmt.Sprintf("cssrs-%d-%d", idosoID, time.Now().Unix())

	query := `
		INSERT INTO clinical_assessments (
			patient_id, assessment_type, session_id, status, trigger_phrase, priority, created_at
		) VALUES ($1, 'C-SSRS', $2, 'in_progress', $3, 'CRITICAL', NOW())
		RETURNING id
	`

	var assessmentID int64
	err := h.db.Conn.QueryRow(query, idosoID, sessionID, triggerPhrase).Scan(&assessmentID)
	if err != nil {
		log.Printf("❌ [C-SSRS] Erro ao criar sessão: %v", err)
		return map[string]interface{}{"error": "Erro ao iniciar avaliação"}, nil
	}

	// 🚨 ALERTA IMEDIATO PARA FAMÍLIA/EQUIPE
	if h.NotifyFunc != nil {
		h.NotifyFunc(idosoID, "critical_alert", map[string]interface{}{
			"type":           "suicide_risk_assessment",
			"trigger_phrase": triggerPhrase,
			"session_id":     sessionID,
			"priority":       "CRITICAL",
		})
	}

	// Também alertar via sistema de alertas
	_ = actions.AlertFamilyWithSeverity(h.db.Conn, h.pushService, h.emailService, idosoID,
		fmt.Sprintf("🚨 ALERTA CRÍTICO: Avaliação de risco suicida iniciada. Frase: '%s'", triggerPhrase),
		"critica")

	log.Printf("✅ [C-SSRS] Sessão CRÍTICA criada. Assessment ID: %d, Session ID: %s", assessmentID, sessionID)

	// Retornar primeira pergunta com extremo cuidado
	return map[string]interface{}{
		"status":          "assessment_started",
		"session_id":      sessionID,
		"assessment_id":   assessmentID,
		"scale":           "C-SSRS",
		"total_questions": 6,
		"priority":        "CRITICAL",
		"message":         "Entendo que você está passando por um momento difícil. Vou fazer algumas perguntas importantes para entender melhor como posso ajudar.",
		"first_question": map[string]interface{}{
			"number": 1,
			"text":   "Você desejou estar morto(a) ou desejou poder dormir e não acordar mais?",
			"options": []string{
				"Sim",
				"Não",
			},
		},
	}, nil
}

// getCallTargetCPF busca o CPF do contato baseado no tipo de chamada
func (h *ToolsHandler) getCallTargetCPF(idosoID int64, callType string) (string, string, error) {
	// Mapear tipo de chamada para tipo de cuidador
	var tipoFilter string
	switch callType {
	case "call_family_webrtc":
		tipoFilter = "familiar"
	case "call_doctor_webrtc":
		tipoFilter = "medico"
	case "call_caregiver_webrtc":
		tipoFilter = "cuidador"
	case "call_central_webrtc":
		tipoFilter = "central"
	default:
		tipoFilter = "familiar" // fallback
	}

	// Query para buscar o contato com prioridade mais alta do tipo solicitado
	query := `
		SELECT c.cpf, c.nome
		FROM cuidadores c
		LEFT JOIN cuidador_idoso ci ON c.id = ci.cuidador_id AND ci.idoso_id = $1
		WHERE (ci.idoso_id = $1 OR c.tipo = 'responsavel')
		  AND (c.tipo = $2 OR ci.parentesco = $2 OR c.tipo ILIKE '%' || $2 || '%')
		  AND c.cpf IS NOT NULL AND c.cpf != ''
		ORDER BY COALESCE(ci.prioridade, 99) ASC
		LIMIT 1
	`

	var cpf, nome string
	err := h.db.Conn.QueryRow(query, idosoID, tipoFilter).Scan(&cpf, &nome)
	if err != nil {
		// Fallback: buscar qualquer contato ativo se não encontrar do tipo específico
		fallbackQuery := `
			SELECT c.cpf, c.nome
			FROM cuidadores c
			JOIN cuidador_idoso ci ON c.id = ci.cuidador_id
			WHERE ci.idoso_id = $1
			  AND c.cpf IS NOT NULL AND c.cpf != ''
			ORDER BY ci.prioridade ASC
			LIMIT 1
		`
		err = h.db.Conn.QueryRow(fallbackQuery, idosoID).Scan(&cpf, &nome)
		if err != nil {
			return "", "", fmt.Errorf("nenhum contato encontrado para %s", callType)
		}
	}

	maskedCPF := "***"
	if len(cpf) >= 3 {
		maskedCPF = "***" + cpf[len(cpf)-3:]
	}
	log.Printf("📞 [CALL] Contato encontrado: %s (CPF: %s) para %s", nome, maskedCPF, callType)
	return cpf, nome, nil
}

// ============================================================================
// ⏰ MÉTODOS DE ALARME
// ============================================================================

// AlarmInfo representa um alarme configurado
type AlarmInfo struct {
	ID         int64    `json:"id"`
	Time       string   `json:"time"`
	Label      string   `json:"label"`
	RepeatDays []string `json:"repeat_days"`
	Active     bool     `json:"active"`
	CreatedAt  string   `json:"created_at"`
}

// createAlarm cria um novo alarme no banco de dados
func (h *ToolsHandler) createAlarm(idosoID int64, timeStr, label string, repeatDays []string) (int64, error) {
	// Validar formato do horário
	if _, err := time.Parse("15:04", timeStr); err != nil {
		return 0, fmt.Errorf("horário inválido: use formato HH:MM (ex: 07:00)")
	}

	// Converter repeatDays para JSON
	repeatDaysJSON := "{}"
	if len(repeatDays) > 0 {
		repeatDaysJSON = fmt.Sprintf("[\"%s\"]", joinStrings(repeatDays, "\",\""))
	} else {
		repeatDaysJSON = "[]"
	}

	// Inserir no banco
	query := `
		INSERT INTO alarmes (idoso_id, horario, descricao, dias_repeticao, ativo, criado_em)
		VALUES ($1, $2, $3, $4::jsonb, true, NOW())
		RETURNING id
	`

	var alarmID int64
	err := h.db.Conn.QueryRow(query, idosoID, timeStr, label, repeatDaysJSON).Scan(&alarmID)
	if err != nil {
		// Se tabela não existe, criar
		if err.Error() == "pq: relation \"alarmes\" does not exist" {
			if createErr := h.createAlarmsTable(); createErr != nil {
				return 0, createErr
			}
			// Tentar novamente
			err = h.db.Conn.QueryRow(query, idosoID, timeStr, label, repeatDaysJSON).Scan(&alarmID)
			if err != nil {
				return 0, fmt.Errorf("erro ao criar alarme: %w", err)
			}
		} else {
			return 0, fmt.Errorf("erro ao criar alarme: %w", err)
		}
	}

	log.Printf("⏰ [ALARM] Alarme criado ID=%d para idoso %d às %s", alarmID, idosoID, timeStr)
	return alarmID, nil
}

// cancelAlarm cancela um ou todos os alarmes
func (h *ToolsHandler) cancelAlarm(idosoID int64, alarmID string) (int, error) {
	var result int64

	if alarmID == "all" {
		// Cancelar todos os alarmes ativos
		query := `UPDATE alarmes SET ativo = false WHERE idoso_id = $1 AND ativo = true`
		res, err := h.db.Conn.Exec(query, idosoID)
		if err != nil {
			return 0, fmt.Errorf("erro ao cancelar alarmes: %w", err)
		}
		result, _ = res.RowsAffected()
		log.Printf("⏰ [ALARM] %d alarmes cancelados para idoso %d", result, idosoID)
	} else {
		// Cancelar alarme específico
		query := `UPDATE alarmes SET ativo = false WHERE id = $1 AND idoso_id = $2`
		res, err := h.db.Conn.Exec(query, alarmID, idosoID)
		if err != nil {
			return 0, fmt.Errorf("erro ao cancelar alarme: %w", err)
		}
		result, _ = res.RowsAffected()
		log.Printf("⏰ [ALARM] Alarme %s cancelado para idoso %d", alarmID, idosoID)
	}

	return int(result), nil
}

// listAlarms lista todos os alarmes ativos de um idoso
func (h *ToolsHandler) listAlarms(idosoID int64) ([]AlarmInfo, error) {
	query := `
		SELECT id, horario, descricao, COALESCE(dias_repeticao::text, '[]'), ativo, criado_em
		FROM alarmes
		WHERE idoso_id = $1 AND ativo = true
		ORDER BY horario ASC
	`

	rows, err := h.db.Conn.Query(query, idosoID)
	if err != nil {
		// Se tabela não existe, retornar vazio
		if err.Error() == "pq: relation \"alarmes\" does not exist" {
			return []AlarmInfo{}, nil
		}
		return nil, fmt.Errorf("erro ao listar alarmes: %w", err)
	}
	defer rows.Close()

	var alarms []AlarmInfo
	for rows.Next() {
		var a AlarmInfo
		var repeatDaysJSON string
		var createdAt time.Time

		if err := rows.Scan(&a.ID, &a.Time, &a.Label, &repeatDaysJSON, &a.Active, &createdAt); err != nil {
			continue
		}

		// Parse repeat_days do JSON
		a.RepeatDays = parseJSONArray(repeatDaysJSON)
		a.CreatedAt = createdAt.Format("02/01/2006 15:04")

		alarms = append(alarms, a)
	}

	return alarms, nil
}

// createAlarmsTable cria a tabela de alarmes se não existir
func (h *ToolsHandler) createAlarmsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS alarmes (
			id SERIAL PRIMARY KEY,
			idoso_id BIGINT NOT NULL REFERENCES idosos(id),
			horario VARCHAR(5) NOT NULL,
			descricao VARCHAR(255) NOT NULL DEFAULT 'Alarme',
			dias_repeticao JSONB DEFAULT '[]',
			ativo BOOLEAN DEFAULT true,
			criado_em TIMESTAMP DEFAULT NOW(),
			atualizado_em TIMESTAMP DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_alarmes_idoso ON alarmes(idoso_id);
		CREATE INDEX IF NOT EXISTS idx_alarmes_ativo ON alarmes(ativo);
	`

	_, err := h.db.Conn.Exec(query)
	if err != nil {
		return fmt.Errorf("erro ao criar tabela alarmes: %w", err)
	}

	log.Println("✅ [ALARM] Tabela 'alarmes' criada com sucesso")
	return nil
}

// Helper: juntar strings com separador
func joinStrings(arr []string, sep string) string {
	if len(arr) == 0 {
		return ""
	}
	result := arr[0]
	for i := 1; i < len(arr); i++ {
		result += sep + arr[i]
	}
	return result
}

// Helper: parse JSON array para []string
func parseJSONArray(jsonStr string) []string {
	var result []string
	// Remove [ e ] e divide por vírgulas
	jsonStr = strings.Trim(jsonStr, "[]")
	if jsonStr == "" {
		return result
	}
	parts := strings.Split(jsonStr, ",")
	for _, p := range parts {
		p = strings.Trim(p, " \"")
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// ============================================================================
// 📋 GTD (Getting Things Done) - Captura e Gerenciamento de Tarefas
// ============================================================================

// GTDTask representa uma tarefa capturada pelo sistema GTD
type GTDTask struct {
	ID          int64      `json:"id"`
	RawInput    string     `json:"raw_input"`   // O que o idoso disse
	NextAction  string     `json:"next_action"` // Ação física concreta
	Context     string     `json:"context"`     // @saúde, @família, @casa, etc
	Project     string     `json:"project"`     // Projeto maior (opcional)
	DueDate     *string    `json:"due_date"`    // Data limite (opcional)
	Status      string     `json:"status"`      // inbox, next, waiting, someday, done
	Priority    int        `json:"priority"`    // 1=alta, 2=média, 3=baixa
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

// handleCaptureTask captura uma preocupação/tarefa vaga e transforma em ação
func (h *ToolsHandler) handleCaptureTask(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	rawInput, _ := args["raw_input"].(string)
	context, _ := args["context"].(string)
	nextAction, _ := args["next_action"].(string)
	dueDate, _ := args["due_date"].(string)
	project, _ := args["project"].(string)

	if rawInput == "" {
		return map[string]interface{}{"error": "raw_input é obrigatório"}, nil
	}

	// Se não tem next_action, usar o raw_input como base
	if nextAction == "" {
		nextAction = rawInput
	}

	// Normalizar contexto
	if context == "" {
		context = "geral"
	}
	context = strings.ToLower(context)

	// Criar tabela se não existir
	if err := h.createGTDTable(); err != nil {
		log.Printf("⚠️ [GTD] Erro ao criar tabela: %v", err)
	}

	// Processar data
	var dueDatePtr *string
	if dueDate != "" {
		parsedDate := h.parseGTDDate(dueDate)
		if parsedDate != "" {
			dueDatePtr = &parsedDate
		}
	}

	// Inserir tarefa
	query := `
		INSERT INTO gtd_tasks (idoso_id, raw_input, next_action, context, project, due_date, status, priority, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'next', 2, NOW())
		RETURNING id, created_at
	`

	var taskID int64
	var createdAt time.Time
	err := h.db.Conn.QueryRow(query, idosoID, rawInput, nextAction, context, project, dueDatePtr).Scan(&taskID, &createdAt)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro ao capturar tarefa: %v", err)}, nil
	}

	log.Printf("📋 [GTD] Tarefa capturada ID=%d: '%s' -> '%s' (@%s)", taskID, rawInput, nextAction, context)

	// Notificar app sobre nova tarefa
	if h.NotifyFunc != nil {
		h.NotifyFunc(idosoID, "gtd_task_created", map[string]interface{}{
			"task_id":     taskID,
			"next_action": nextAction,
			"context":     context,
		})
	}

	return map[string]interface{}{
		"status":      "capturado",
		"task_id":     taskID,
		"raw_input":   rawInput,
		"next_action": nextAction,
		"context":     context,
		"message":     fmt.Sprintf("Entendi! Anotei: '%s'. Isso está na sua lista de próximas ações.", nextAction),
	}, nil
}

// handleListTasks lista as próximas ações pendentes
func (h *ToolsHandler) handleListTasks(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	context, _ := args["context"].(string)
	limitFloat, _ := args["limit"].(float64)
	limit := int(limitFloat)
	if limit == 0 {
		limit = 5
	}

	query := `
		SELECT id, raw_input, next_action, context, COALESCE(project, ''),
		       COALESCE(to_char(due_date, 'DD/MM/YYYY'), ''), status, priority, created_at
		FROM gtd_tasks
		WHERE idoso_id = $1 AND status IN ('next', 'inbox')
	`
	queryArgs := []interface{}{idosoID}

	if context != "" {
		query += " AND context = $2"
		queryArgs = append(queryArgs, strings.ToLower(context))
		query += " ORDER BY priority ASC, created_at ASC LIMIT $3"
		queryArgs = append(queryArgs, limit)
	} else {
		query += " ORDER BY priority ASC, created_at ASC LIMIT $2"
		queryArgs = append(queryArgs, limit)
	}

	rows, err := h.db.Conn.Query(query, queryArgs...)
	if err != nil {
		// Se tabela não existe, retornar vazio
		if strings.Contains(err.Error(), "does not exist") {
			return map[string]interface{}{
				"status":  "sucesso",
				"tasks":   []interface{}{},
				"message": "Você não tem tarefas pendentes. Que bom!",
			}, nil
		}
		return map[string]interface{}{"error": fmt.Sprintf("Erro ao listar tarefas: %v", err)}, nil
	}
	defer rows.Close()

	var tasks []map[string]interface{}
	for rows.Next() {
		var t GTDTask
		var project, dueDate string
		var createdAt time.Time

		if err := rows.Scan(&t.ID, &t.RawInput, &t.NextAction, &t.Context, &project, &dueDate, &t.Status, &t.Priority, &createdAt); err != nil {
			continue
		}

		tasks = append(tasks, map[string]interface{}{
			"id":          t.ID,
			"next_action": t.NextAction,
			"context":     t.Context,
			"project":     project,
			"due_date":    dueDate,
			"priority":    t.Priority,
		})
	}

	if len(tasks) == 0 {
		return map[string]interface{}{
			"status":  "sucesso",
			"tasks":   []interface{}{},
			"message": "Você não tem tarefas pendentes. Que bom, está tudo em dia!",
		}, nil
	}

	// Montar mensagem de fala
	var taskList []string
	for i, task := range tasks {
		action := task["next_action"].(string)
		taskList = append(taskList, fmt.Sprintf("%d. %s", i+1, action))
	}
	message := fmt.Sprintf("Você tem %d tarefa(s) pendente(s):\n%s", len(tasks), strings.Join(taskList, "\n"))

	return map[string]interface{}{
		"status":  "sucesso",
		"tasks":   tasks,
		"count":   len(tasks),
		"message": message,
	}, nil
}

// handleCompleteTask marca uma tarefa como concluída
func (h *ToolsHandler) handleCompleteTask(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	taskID, _ := args["task_id"].(float64)
	taskDesc, _ := args["task_description"].(string)

	var query string
	var queryArgs []interface{}

	if taskID > 0 {
		query = `
			UPDATE gtd_tasks
			SET status = 'done', completed_at = NOW()
			WHERE id = $1 AND idoso_id = $2 AND status != 'done'
			RETURNING id, next_action
		`
		queryArgs = []interface{}{int64(taskID), idosoID}
	} else if taskDesc != "" {
		// Buscar por descrição parcial
		query = `
			UPDATE gtd_tasks
			SET status = 'done', completed_at = NOW()
			WHERE idoso_id = $1 AND status != 'done'
			  AND (LOWER(next_action) LIKE '%' || LOWER($2) || '%' OR LOWER(raw_input) LIKE '%' || LOWER($2) || '%')
			RETURNING id, next_action
		`
		queryArgs = []interface{}{idosoID, taskDesc}
	} else {
		return map[string]interface{}{"error": "Informe o ID ou descrição da tarefa"}, nil
	}

	var completedID int64
	var completedAction string
	err := h.db.Conn.QueryRow(query, queryArgs...).Scan(&completedID, &completedAction)
	if err != nil {
		return map[string]interface{}{
			"status":  "não encontrado",
			"message": "Não encontrei essa tarefa nas suas pendências.",
		}, nil
	}

	log.Printf("✅ [GTD] Tarefa concluída ID=%d: '%s'", completedID, completedAction)

	// Notificar app
	if h.NotifyFunc != nil {
		h.NotifyFunc(idosoID, "gtd_task_completed", map[string]interface{}{
			"task_id":     completedID,
			"next_action": completedAction,
		})
	}

	return map[string]interface{}{
		"status":      "concluído",
		"task_id":     completedID,
		"next_action": completedAction,
		"message":     fmt.Sprintf("Ótimo! Marquei '%s' como concluída. Parabéns!", completedAction),
	}, nil
}

// handleClarifyTask pede mais informação para definir a próxima ação
func (h *ToolsHandler) handleClarifyTask(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	taskID, _ := args["task_id"].(float64)
	question, _ := args["question"].(string)

	if question == "" {
		question = "Qual é a próxima ação física que você precisa fazer?"
	}

	return map[string]interface{}{
		"status":   "clarificação_necessária",
		"task_id":  int64(taskID),
		"question": question,
		"message":  fmt.Sprintf("Para eu poder te ajudar melhor: %s", question),
	}, nil
}

// handleWeeklyReview mostra revisão semanal GTD
func (h *ToolsHandler) handleWeeklyReview(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	// Contar tarefas por status
	statsQuery := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'next') as next_count,
			COUNT(*) FILTER (WHERE status = 'inbox') as inbox_count,
			COUNT(*) FILTER (WHERE status = 'waiting') as waiting_count,
			COUNT(*) FILTER (WHERE status = 'done' AND completed_at > NOW() - INTERVAL '7 days') as done_week,
			COUNT(*) FILTER (WHERE due_date < NOW() AND status NOT IN ('done')) as overdue_count
		FROM gtd_tasks
		WHERE idoso_id = $1
	`

	var nextCount, inboxCount, waitingCount, doneWeek, overdueCount int
	err := h.db.Conn.QueryRow(statsQuery, idosoID).Scan(&nextCount, &inboxCount, &waitingCount, &doneWeek, &overdueCount)
	if err != nil {
		// Se tabela não existe
		if strings.Contains(err.Error(), "does not exist") {
			return map[string]interface{}{
				"status":  "sucesso",
				"message": "Você ainda não tem tarefas cadastradas. Que tal começar a usar o sistema de captura?",
			}, nil
		}
	}

	// Montar mensagem
	var parts []string
	if doneWeek > 0 {
		parts = append(parts, fmt.Sprintf("Parabéns! Você concluiu %d tarefa(s) esta semana", doneWeek))
	}
	if nextCount > 0 {
		parts = append(parts, fmt.Sprintf("Você tem %d próxima(s) ação(ões) pendente(s)", nextCount))
	}
	if inboxCount > 0 {
		parts = append(parts, fmt.Sprintf("%d item(ns) na caixa de entrada para processar", inboxCount))
	}
	if overdueCount > 0 {
		parts = append(parts, fmt.Sprintf("⚠️ Atenção: %d tarefa(s) atrasada(s)", overdueCount))
	}
	if waitingCount > 0 {
		parts = append(parts, fmt.Sprintf("%d tarefa(s) aguardando alguém", waitingCount))
	}

	message := "Revisão semanal:\n" + strings.Join(parts, "\n")
	if len(parts) == 0 {
		message = "Sua lista está vazia. Você está em dia com tudo!"
	}

	return map[string]interface{}{
		"status":        "sucesso",
		"next_count":    nextCount,
		"inbox_count":   inboxCount,
		"waiting_count": waitingCount,
		"done_week":     doneWeek,
		"overdue_count": overdueCount,
		"message":       message,
	}, nil
}

// createGTDTable cria a tabela de tarefas GTD
func (h *ToolsHandler) createGTDTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS gtd_tasks (
			id SERIAL PRIMARY KEY,
			idoso_id BIGINT NOT NULL REFERENCES idosos(id),
			raw_input TEXT NOT NULL,
			next_action TEXT NOT NULL,
			context VARCHAR(50) DEFAULT 'geral',
			project VARCHAR(255),
			due_date DATE,
			status VARCHAR(20) DEFAULT 'inbox',
			priority INT DEFAULT 2,
			created_at TIMESTAMP DEFAULT NOW(),
			completed_at TIMESTAMP,
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_gtd_idoso ON gtd_tasks(idoso_id);
		CREATE INDEX IF NOT EXISTS idx_gtd_status ON gtd_tasks(status);
		CREATE INDEX IF NOT EXISTS idx_gtd_context ON gtd_tasks(context);
		CREATE INDEX IF NOT EXISTS idx_gtd_due ON gtd_tasks(due_date) WHERE due_date IS NOT NULL;
	`

	_, err := h.db.Conn.Exec(query)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("erro ao criar tabela gtd_tasks: %w", err)
	}

	log.Println("✅ [GTD] Tabela 'gtd_tasks' verificada/criada")
	return nil
}

// parseGTDDate converte datas relativas para absolutas
func (h *ToolsHandler) parseGTDDate(dateStr string) string {
	now := time.Now()
	lower := strings.ToLower(dateStr)

	switch {
	case lower == "hoje":
		return now.Format("2006-01-02")
	case lower == "amanhã" || lower == "amanha":
		return now.AddDate(0, 0, 1).Format("2006-01-02")
	case lower == "segunda" || lower == "segunda-feira":
		return h.nextWeekday(now, time.Monday)
	case lower == "terça" || lower == "terca" || lower == "terça-feira":
		return h.nextWeekday(now, time.Tuesday)
	case lower == "quarta" || lower == "quarta-feira":
		return h.nextWeekday(now, time.Wednesday)
	case lower == "quinta" || lower == "quinta-feira":
		return h.nextWeekday(now, time.Thursday)
	case lower == "sexta" || lower == "sexta-feira":
		return h.nextWeekday(now, time.Friday)
	case lower == "sábado" || lower == "sabado":
		return h.nextWeekday(now, time.Saturday)
	case lower == "domingo":
		return h.nextWeekday(now, time.Sunday)
	case strings.HasPrefix(lower, "próxima semana") || strings.HasPrefix(lower, "proxima semana"):
		return now.AddDate(0, 0, 7).Format("2006-01-02")
	default:
		// Tentar parsear como data ISO
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			return t.Format("2006-01-02")
		}
		// Tentar formato brasileiro
		if t, err := time.Parse("02/01/2006", dateStr); err == nil {
			return t.Format("2006-01-02")
		}
		return ""
	}
}

// nextWeekday encontra o próximo dia da semana
func (h *ToolsHandler) nextWeekday(from time.Time, weekday time.Weekday) string {
	daysUntil := int(weekday - from.Weekday())
	if daysUntil <= 0 {
		daysUntil += 7
	}
	return from.AddDate(0, 0, daysUntil).Format("2006-01-02")
}

// ============================================================================
// 🧠 SPACED REPETITION - Reforço de Memória
// ============================================================================

// handleRememberThis captura informação para reforço de memória
func (h *ToolsHandler) handleRememberThis(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	if h.spacedService == nil {
		return map[string]interface{}{"error": "Serviço de memória não disponível"}, nil
	}

	content, _ := args["content"].(string)
	category, _ := args["category"].(string)
	trigger, _ := args["trigger"].(string)
	importanceFloat, _ := args["importance"].(float64)
	importance := int(importanceFloat)
	if importance == 0 {
		importance = 3
	}

	if content == "" {
		return map[string]interface{}{"error": "Conteúdo não pode ser vazio"}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	item, err := h.spacedService.CaptureMemory(ctx, idosoID, content, category, trigger, importance)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro ao capturar: %v", err)}, nil
	}

	// Notificar app
	if h.NotifyFunc != nil {
		h.NotifyFunc(idosoID, "memory_captured", map[string]interface{}{
			"item_id":     item.ID,
			"content":     item.Content,
			"category":    item.Category,
			"next_review": item.NextReview.Format("15:04"),
		})
	}

	// Calcular quando será o primeiro reforço
	untilReview := time.Until(item.NextReview)
	reviewMsg := ""
	if untilReview < time.Hour {
		reviewMsg = fmt.Sprintf("em %d minutos", int(untilReview.Minutes()))
	} else {
		reviewMsg = fmt.Sprintf("em %.1f horas", untilReview.Hours())
	}

	return map[string]interface{}{
		"status":      "capturado",
		"item_id":     item.ID,
		"content":     item.Content,
		"category":    item.Category,
		"importance":  item.Importance,
		"next_review": item.NextReview.Format("15:04"),
		"message":     fmt.Sprintf("Anotei! Vou te ajudar a lembrar: '%s'. Primeiro reforço %s.", content, reviewMsg),
	}, nil
}

// handleReviewMemory registra resultado de um reforço
func (h *ToolsHandler) handleReviewMemory(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	if h.spacedService == nil {
		return map[string]interface{}{"error": "Serviço de memória não disponível"}, nil
	}

	itemID, _ := args["item_id"].(float64)
	remembered, _ := args["remembered"].(bool)
	qualityFloat, _ := args["quality"].(float64)
	quality := int(qualityFloat)

	// Se não passou item_id, buscar o mais recente pendente
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if itemID == 0 {
		items, err := h.spacedService.GetPendingReviews(ctx, idosoID, 1)
		if err != nil || len(items) == 0 {
			return map[string]interface{}{
				"status":  "sem_pendencias",
				"message": "Não há memórias pendentes para revisar agora.",
			}, nil
		}
		itemID = float64(items[0].ID)
	}

	item, err := h.spacedService.RecordReview(ctx, int64(itemID), quality, remembered)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro ao registrar: %v", err)}, nil
	}

	// Mensagem baseada no resultado
	var message string
	if remembered {
		if item.Status == "mastered" {
			message = fmt.Sprintf("Excelente! Você dominou essa memória: '%s'. Não vou mais te lembrar.", item.Content)
		} else {
			nextDays := item.IntervalDays
			if nextDays < 1 {
				message = fmt.Sprintf("Muito bem! Próximo reforço em %.0f horas.", nextDays*24)
			} else {
				message = fmt.Sprintf("Ótimo! Próximo reforço em %.0f dia(s).", nextDays)
			}
		}
	} else {
		message = fmt.Sprintf("Sem problemas, vamos reforçar. Lembre-se: '%s'. Vou te perguntar de novo em breve.", item.Content)
	}

	return map[string]interface{}{
		"status":        "registrado",
		"item_id":       item.ID,
		"remembered":    remembered,
		"next_review":   item.NextReview.Format("02/01 15:04"),
		"interval_days": item.IntervalDays,
		"status_item":   item.Status,
		"message":       message,
	}, nil
}

// handleListMemories lista memórias sendo reforçadas
func (h *ToolsHandler) handleListMemories(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	if h.spacedService == nil {
		return map[string]interface{}{"error": "Serviço de memória não disponível"}, nil
	}

	limitFloat, _ := args["limit"].(float64)
	limit := int(limitFloat)
	if limit == 0 {
		limit = 5
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Buscar itens pendentes primeiro
	items, err := h.spacedService.GetPendingReviews(ctx, idosoID, limit)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro: %v", err)}, nil
	}

	if len(items) == 0 {
		return map[string]interface{}{
			"status":  "sucesso",
			"items":   []interface{}{},
			"message": "Nenhuma memória pendente de reforço agora. Você está em dia!",
		}, nil
	}

	// Formatar para resposta
	var memories []map[string]interface{}
	var descriptions []string
	for i, item := range items {
		memories = append(memories, map[string]interface{}{
			"id":          item.ID,
			"content":     item.Content,
			"category":    item.Category,
			"importance":  item.Importance,
			"repetitions": item.RepetitionCount,
		})
		descriptions = append(descriptions, fmt.Sprintf("%d. %s", i+1, item.Content))
	}

	message := fmt.Sprintf("Você tem %d memória(s) para reforçar:\n%s", len(items), strings.Join(descriptions, "\n"))

	return map[string]interface{}{
		"status":  "sucesso",
		"items":   memories,
		"count":   len(items),
		"message": message,
	}, nil
}

// handlePauseMemory pausa reforços de uma memória
func (h *ToolsHandler) handlePauseMemory(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	if h.spacedService == nil {
		return map[string]interface{}{"error": "Serviço de memória não disponível"}, nil
	}

	itemID, _ := args["item_id"].(float64)
	contentSearch, _ := args["content"].(string)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Se passou descrição ao invés de ID, buscar
	if itemID == 0 && contentSearch != "" {
		query := `
			SELECT id FROM spaced_memory_items
			WHERE idoso_id = $1 AND status = 'active'
			  AND LOWER(content) LIKE '%' || LOWER($2) || '%'
			LIMIT 1
		`
		err := h.db.Conn.QueryRowContext(ctx, query, idosoID, contentSearch).Scan(&itemID)
		if err != nil {
			return map[string]interface{}{
				"status":  "não encontrado",
				"message": "Não encontrei essa memória na sua lista.",
			}, nil
		}
	}

	if itemID == 0 {
		return map[string]interface{}{"error": "Informe o ID ou descrição da memória"}, nil
	}

	err := h.spacedService.PauseItem(ctx, int64(itemID))
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro: %v", err)}, nil
	}

	return map[string]interface{}{
		"status":  "pausado",
		"item_id": int64(itemID),
		"message": "Ok, pausei os reforços dessa memória. Se quiser retomar, é só me pedir.",
	}, nil
}

// handleMemoryStats mostra estatísticas de memória
func (h *ToolsHandler) handleMemoryStats(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	if h.spacedService == nil {
		return map[string]interface{}{"error": "Serviço de memória não disponível"}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := h.spacedService.GetStats(ctx, idosoID)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro: %v", err)}, nil
	}

	total := stats["total_items"].(int)
	active := stats["active_items"].(int)
	mastered := stats["mastered_items"].(int)
	pending := stats["pending_reviews"].(int)
	successRate := stats["avg_success_rate"].(float64) * 100

	var message string
	if total == 0 {
		message = "Você ainda não começou a usar o reforço de memória. Quando quiser lembrar de algo importante, me avise!"
	} else {
		message = fmt.Sprintf("Sua memória está indo bem! Você tem %d memória(s) ativas, %d dominada(s), e %d pendente(s) de reforço. Taxa de sucesso: %.0f%%.",
			active, mastered, pending, successRate)
	}

	return map[string]interface{}{
		"status":       "sucesso",
		"total":        total,
		"active":       active,
		"mastered":     mastered,
		"pending":      pending,
		"success_rate": successRate,
		"message":      message,
	}, nil
}

// ============================================================================
// 📊 HABIT TRACKING - Log de Hábitos
// ============================================================================

// handleLogHabit registra sucesso/falha de um hábito
func (h *ToolsHandler) handleLogHabit(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	if h.habitTracker == nil {
		return map[string]interface{}{"error": "Serviço de hábitos não disponível"}, nil
	}

	habitName, _ := args["habit_name"].(string)
	success, _ := args["success"].(bool)
	notes, _ := args["notes"].(string)

	if habitName == "" {
		return map[string]interface{}{"error": "Nome do hábito é obrigatório"}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logEntry, err := h.habitTracker.LogHabit(ctx, idosoID, habitName, success, "voice", notes, nil)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro: %v", err)}, nil
	}

	// Notificar app
	if h.NotifyFunc != nil {
		h.NotifyFunc(idosoID, "habit_logged", map[string]interface{}{
			"log_id":  logEntry.ID,
			"habit":   habitName,
			"success": success,
		})
	}

	var message string
	if success {
		message = fmt.Sprintf("Ótimo! Registrei que você completou '%s'. Continue assim!", habitName)
	} else {
		message = fmt.Sprintf("Entendi, registrei. Não se preocupe, amanhã é um novo dia!")
	}

	return map[string]interface{}{
		"status":  "registrado",
		"log_id":  logEntry.ID,
		"habit":   habitName,
		"success": success,
		"message": message,
	}, nil
}

// handleLogWater registra consumo de água
func (h *ToolsHandler) handleLogWater(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	if h.habitTracker == nil {
		return map[string]interface{}{"error": "Serviço de hábitos não disponível"}, nil
	}

	glassesFloat, _ := args["glasses"].(float64)
	glasses := int(glassesFloat)
	if glasses == 0 {
		glasses = 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logEntry, err := h.habitTracker.LogWater(ctx, idosoID, glasses, "voice")
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro: %v", err)}, nil
	}

	// Notificar app
	if h.NotifyFunc != nil {
		h.NotifyFunc(idosoID, "water_logged", map[string]interface{}{
			"log_id":  logEntry.ID,
			"glasses": glasses,
		})
	}

	copoStr := "copo"
	if glasses > 1 {
		copoStr = "copos"
	}

	return map[string]interface{}{
		"status":  "registrado",
		"log_id":  logEntry.ID,
		"glasses": glasses,
		"message": fmt.Sprintf("Anotei! %d %s de água. Hidratação é muito importante!", glasses, copoStr),
	}, nil
}

// handleHabitStats mostra estatísticas de hábitos
func (h *ToolsHandler) handleHabitStats(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	if h.habitTracker == nil {
		return map[string]interface{}{"error": "Serviço de hábitos não disponível"}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	report, err := h.habitTracker.GetWeeklyReport(ctx, idosoID)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro: %v", err)}, nil
	}

	patterns := report["patterns"].([]habits.HabitPattern)
	problematic := report["problematic"].([]string)
	excellent := report["excellent"].([]string)

	var parts []string
	if len(excellent) > 0 {
		parts = append(parts, fmt.Sprintf("Parabéns! Você está mandando bem em: %s", strings.Join(excellent, ", ")))
	}
	if len(problematic) > 0 {
		parts = append(parts, fmt.Sprintf("Precisamos melhorar: %s", strings.Join(problematic, ", ")))
	}
	if len(patterns) == 0 {
		parts = append(parts, "Ainda não temos dados suficientes. Continue registrando seus hábitos!")
	}

	return map[string]interface{}{
		"status":      "sucesso",
		"patterns":    patterns,
		"problematic": problematic,
		"excellent":   excellent,
		"message":     strings.Join(parts, " "),
	}, nil
}

// handleHabitSummary mostra resumo do dia
func (h *ToolsHandler) handleHabitSummary(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	if h.habitTracker == nil {
		return map[string]interface{}{"error": "Serviço de hábitos não disponível"}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	summary, err := h.habitTracker.GetDailySummary(ctx, idosoID)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro: %v", err)}, nil
	}

	totalCompleted := summary["total_completed"].(int)
	totalHabits := summary["total_habits"].(int)

	var message string
	if totalHabits == 0 {
		message = "Você ainda não tem hábitos registrados hoje. Que tal começar?"
	} else if totalCompleted == totalHabits {
		message = fmt.Sprintf("Excelente! Você completou todos os %d hábitos de hoje!", totalCompleted)
	} else {
		message = fmt.Sprintf("Hoje você completou %d de %d hábitos. Continue assim!", totalCompleted, totalHabits)
	}

	return map[string]interface{}{
		"status":  "sucesso",
		"summary": summary,
		"message": message,
	}, nil
}

// ============================================================================
// 📍 PESQUISA DE LOCAIS E MAPAS
// ============================================================================

// handleSearchPlaces pesquisa locais próximos
func (h *ToolsHandler) handleSearchPlaces(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	query, _ := args["query"].(string)
	placeType, _ := args["type"].(string)
	radiusFloat, _ := args["radius"].(float64)
	radius := int(radiusFloat)
	if radius == 0 {
		radius = 5000
	}

	if query == "" && placeType == "" {
		return map[string]interface{}{"error": "Informe o que deseja buscar"}, nil
	}

	// Enviar comando para o app executar a busca via Google Places API
	if h.NotifyFunc != nil {
		h.NotifyFunc(idosoID, "search_places", map[string]interface{}{
			"query":  query,
			"type":   placeType,
			"radius": radius,
		})
	}

	return map[string]interface{}{
		"status":  "buscando",
		"query":   query,
		"type":    placeType,
		"radius":  radius,
		"message": fmt.Sprintf("Buscando '%s' perto de você...", query),
	}, nil
}

// handleGetDirections obtém direções para um local
func (h *ToolsHandler) handleGetDirections(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	destination, _ := args["destination"].(string)
	mode, _ := args["mode"].(string)

	if destination == "" {
		return map[string]interface{}{"error": "Informe o destino"}, nil
	}

	if mode == "" {
		mode = "walking" // Padrão para idosos: caminhada
	}

	// Enviar comando para o app abrir Google Maps com direções
	if h.NotifyFunc != nil {
		h.NotifyFunc(idosoID, "get_directions", map[string]interface{}{
			"destination": destination,
			"mode":        mode,
		})
	}

	modeNames := map[string]string{
		"walking":   "a pé",
		"driving":   "de carro",
		"transit":   "de transporte público",
		"bicycling": "de bicicleta",
	}

	modeName := modeNames[mode]
	if modeName == "" {
		modeName = mode
	}

	return map[string]interface{}{
		"status":      "abrindo_mapa",
		"destination": destination,
		"mode":        mode,
		"message":     fmt.Sprintf("Abrindo rota %s para %s no mapa...", modeName, destination),
	}, nil
}

// handleNearbyTransport mostra transporte público próximo
func (h *ToolsHandler) handleNearbyTransport(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	transportType, _ := args["type"].(string)
	if transportType == "" {
		transportType = "all"
	}

	// Enviar comando para o app buscar transporte próximo
	if h.NotifyFunc != nil {
		h.NotifyFunc(idosoID, "nearby_transport", map[string]interface{}{
			"type": transportType,
		})
	}

	var message string
	switch transportType {
	case "bus":
		message = "Buscando pontos de ônibus próximos..."
	case "metro":
		message = "Buscando estações de metrô próximas..."
	default:
		message = "Buscando transporte público próximo..."
	}

	return map[string]interface{}{
		"status":  "buscando",
		"type":    transportType,
		"message": message,
	}, nil
}

// ============================================================================
// 📱 ABRIR APLICATIVOS
// ============================================================================

// handleOpenApp abre um aplicativo no celular
func (h *ToolsHandler) handleOpenApp(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	appName, _ := args["app_name"].(string)

	if appName == "" {
		return map[string]interface{}{"error": "Informe qual aplicativo deseja abrir"}, nil
	}

	// Mapear nomes para package names do Android
	appPackages := map[string]struct {
		pkg     string
		display string
	}{
		"whatsapp":      {"com.whatsapp", "WhatsApp"},
		"agenda":        {"com.google.android.calendar", "Agenda"},
		"calendario":    {"com.google.android.calendar", "Calendário"},
		"relogio":       {"com.google.android.deskclock", "Relógio"},
		"alarme":        {"com.google.android.deskclock", "Alarme"},
		"camera":        {"com.android.camera", "Câmera"},
		"galeria":       {"com.google.android.apps.photos", "Galeria"},
		"fotos":         {"com.google.android.apps.photos", "Fotos"},
		"telefone":      {"com.android.dialer", "Telefone"},
		"mensagens":     {"com.google.android.apps.messaging", "Mensagens"},
		"sms":           {"com.google.android.apps.messaging", "SMS"},
		"spotify":       {"com.spotify.music", "Spotify"},
		"youtube":       {"com.google.android.youtube", "YouTube"},
		"maps":          {"com.google.android.apps.maps", "Google Maps"},
		"mapa":          {"com.google.android.apps.maps", "Mapa"},
		"gmail":         {"com.google.android.gm", "Gmail"},
		"email":         {"com.google.android.gm", "E-mail"},
		"chrome":        {"com.android.chrome", "Chrome"},
		"navegador":     {"com.android.chrome", "Navegador"},
		"calculadora":   {"com.google.android.calculator", "Calculadora"},
		"configuracoes": {"com.android.settings", "Configurações"},
		"ajustes":       {"com.android.settings", "Ajustes"},
	}

	appKey := strings.ToLower(strings.ReplaceAll(appName, " ", ""))
	appInfo, exists := appPackages[appKey]
	if !exists {
		// Tentar abrir pelo nome mesmo
		appInfo = struct {
			pkg     string
			display string
		}{appName, appName}
	}

	// Enviar comando para o app abrir o aplicativo
	if h.NotifyFunc != nil {
		h.NotifyFunc(idosoID, "open_app", map[string]interface{}{
			"package":      appInfo.pkg,
			"display_name": appInfo.display,
		})
	}

	return map[string]interface{}{
		"status":  "abrindo",
		"app":     appInfo.display,
		"package": appInfo.pkg,
		"message": fmt.Sprintf("Abrindo %s...", appInfo.display),
	}, nil
}
