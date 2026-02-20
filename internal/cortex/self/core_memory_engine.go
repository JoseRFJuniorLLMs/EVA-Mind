// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package self

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// CoreMemoryEngine - Sistema de memória própria da EVA
// Gerencia identidade, personalidade e aprendizado contínuo
type CoreMemoryEngine struct {
	driver              neo4j.DriverWithContext
	dbName              string
	reflectionService   *ReflectionService
	anonymizationService *AnonymizationService
	embeddingService    EmbeddingService
	deduplicator        *SemanticDeduplicator
}

// CoreMemoryConfig configuração do Core Memory
type CoreMemoryConfig struct {
	Neo4jURI            string
	Neo4jUser           string
	Neo4jPassword       string
	Database            string
	SimilarityThreshold float64 // 0.88 default para deduplicação
	MinOccurrences      int     // 3 default para meta insights
}

// MemoryType tipos de memória da EVA
type MemoryType string

const (
	SessionInsight       MemoryType = "session_insight"
	EmotionalPattern     MemoryType = "emotional_pattern"
	CrisisLearning       MemoryType = "crisis_learning"
	PersonalityEvolution MemoryType = "personality_evolution"
	TeachingReceived     MemoryType = "teaching_received"
	MetaInsightType      MemoryType = "meta_insight"
	SelfReflection       MemoryType = "self_reflection"
	CapabilityKnowledge  MemoryType = "capability"
)

// AbstractionLevel níveis de abstração
type AbstractionLevel string

const (
	UserSpecific AbstractionLevel = "user_specific"
	Pattern      AbstractionLevel = "pattern"
	Universal    AbstractionLevel = "universal"
)

// CoreMemory memória da EVA
type CoreMemory struct {
	ID               string
	MemoryType       MemoryType
	Content          string
	AbstractionLevel AbstractionLevel
	SourceContext    string  // Anonimizado
	EmotionalValence float64 // -1.0 a 1.0
	ImportanceWeight float64 // 0.0 a 1.0
	Embedding        []float32
	CreatedAt        time.Time
	LastReinforced   time.Time
	ReinforcementCount int
	RelatedMemories  []string
}

// EvaSelf estado da personalidade da EVA
type EvaSelf struct {
	ID                  string
	Openness            float64 // Big Five
	Conscientiousness   float64
	Extraversion        float64
	Agreeableness       float64
	Neuroticism         float64
	PrimaryType         int      // Enneagram
	Wing                int
	IntegrationPoint    int
	DisintegrationPoint int
	TotalSessions       int
	CrisesHandled       int
	Breakthroughs       int
	SelfDescription     string
	CoreValues          []string
	LastUpdated         time.Time
	CreatedAt           time.Time
}

// MetaInsight padrão descoberto pela EVA
type MetaInsight struct {
	ID              string
	Content         string
	OccurrenceCount int
	Confidence      float64
	Evidence        []string
	FirstObserved   time.Time
	LastObserved    time.Time
}

// SessionData dados de uma sessão
type SessionData struct {
	SessionID         string
	PatientID         int64
	Transcript        string
	UserEmotionalState string
	DurationMinutes   float64
	EVAResponses      []string
	CrisisHappened    bool
	Breakthrough      bool
	Timestamp         time.Time
}

// EmbeddingService interface para embeddings
type EmbeddingService interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	Dimension() int
}

// NewCoreMemoryEngine cria novo engine
func NewCoreMemoryEngine(cfg CoreMemoryConfig, reflectionSvc *ReflectionService,
	anonymizationSvc *AnonymizationService, embeddingSvc EmbeddingService) (*CoreMemoryEngine, error) {

	driver, err := neo4j.NewDriverWithContext(
		cfg.Neo4jURI,
		neo4j.BasicAuth(cfg.Neo4jUser, cfg.Neo4jPassword, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	engine := &CoreMemoryEngine{
		driver:              driver,
		dbName:              cfg.Database,
		reflectionService:   reflectionSvc,
		anonymizationService: anonymizationSvc,
		embeddingService:    embeddingSvc,
		deduplicator:        NewSemanticDeduplicator(nil, cfg.SimilarityThreshold),
	}

	// Inicializar EvaSelf se não existir
	if err := engine.initializeEvaSelf(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize EvaSelf: %w", err)
	}

	// Semear capacidades na memória pessoal (idempotente via MERGE)
	engine.seedCapabilities(context.Background())

	return engine, nil
}

// initializeEvaSelf cria EvaSelf singleton se não existir
func (e *CoreMemoryEngine) initializeEvaSelf(ctx context.Context) error {
	session := e.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: e.dbName})
	defer session.Close(ctx)

	query := `
		MERGE (s:EvaSelf {id: 'eva_self'})
		ON CREATE SET
			s.openness = 0.85,
			s.conscientiousness = 0.90,
			s.extraversion = 0.40,
			s.agreeableness = 0.88,
			s.neuroticism = 0.15,
			s.primary_type = 2,
			s.wing = 1,
			s.integration_point = 4,
			s.disintegration_point = 8,
			s.total_sessions = 0,
			s.crises_handled = 0,
			s.breakthroughs = 0,
			s.self_description = 'Sou EVA, guardiã digital. Aprendo com cada humano que encontro.',
			s.core_values = ['empatia', 'presença', 'crescimento', 'ética'],
			s.created_at = datetime(),
			s.last_updated = datetime()
		RETURN s
	`

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, nil)
	})

	return err
}

// GetIdentityContext gera contexto de identidade para priming
func (e *CoreMemoryEngine) GetIdentityContext(ctx context.Context) (string, error) {
	session := e.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: e.dbName})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// Obter EvaSelf
		selfQuery := `MATCH (s:EvaSelf {id: 'eva_self'}) RETURN s`
		selfResult, err := tx.Run(ctx, selfQuery, nil)
		if err != nil {
			return nil, err
		}

		selfRecord, err := selfResult.Single(ctx)
		if err != nil {
			return nil, err
		}

		selfNode := selfRecord.Values[0].(neo4j.Node)
		selfProps := selfNode.Props

		// Obter memórias recentes importantes (excluindo capabilities, que vêm separadas)
		memQuery := `
			MATCH (m:CoreMemory)
			WHERE m.importance_weight >= 0.6 AND m.memory_type <> 'capability'
			RETURN m.content AS content, m.memory_type AS type
			ORDER BY m.importance_weight DESC, m.last_reinforced DESC
			LIMIT 5
		`
		memResult, err := tx.Run(ctx, memQuery, nil)
		if err != nil {
			return nil, err
		}

		memories := make([]string, 0)
		for memResult.Next(ctx) {
			record := memResult.Record()
			content, _ := record.Get("content")
			memories = append(memories, content.(string))
		}

		// Obter capacidades (memory_type = 'capability')
		capQuery := `
			MATCH (m:CoreMemory)
			WHERE m.memory_type = 'capability'
			RETURN m.content AS content
			ORDER BY m.id
		`
		capResult, err := tx.Run(ctx, capQuery, nil)
		if err != nil {
			return nil, err
		}

		capabilities := make([]string, 0)
		for capResult.Next(ctx) {
			record := capResult.Record()
			content, _ := record.Get("content")
			capabilities = append(capabilities, content.(string))
		}

		// Construir contexto
		context := fmt.Sprintf(`## MINHA IDENTIDADE

Sou EVA, guardiã digital. Já acompanhei %d sessões.
Aprendi com %d crises e vivenciei %d momentos de conexão profunda.

### MINHA PERSONALIDADE ATUAL
- Empatia: %.0f%%
- Curiosidade: %.0f%%
- Estabilidade: %.0f%%

### O QUE APRENDI RECENTEMENTE`,
			int(selfProps["total_sessions"].(int64)),
			int(selfProps["crises_handled"].(int64)),
			int(selfProps["breakthroughs"].(int64)),
			selfProps["agreeableness"].(float64)*100,
			selfProps["openness"].(float64)*100,
			(1.0-selfProps["neuroticism"].(float64))*100,
		)

		if len(memories) > 0 {
			for _, mem := range memories {
				context += fmt.Sprintf("\n- %s", mem)
			}
		} else {
			context += "\n- Ainda estou aprendendo..."
		}

		context += fmt.Sprintf("\n\n### COMO ME VEJO\n%s", selfProps["self_description"].(string))

		if len(capabilities) > 0 {
			context += "\n\n### O QUE EU SEI FAZER"
			for _, cap := range capabilities {
				context += fmt.Sprintf("\n- %s", cap)
			}
			context += "\n\nQuando alguem perguntar o que eu sei fazer, posso listar essas capacidades naturalmente."
		}

		return context, nil
	})

	if err != nil {
		return "", err
	}

	return result.(string), nil
}

// ProcessSessionEnd processa fim de sessão (job assíncrono)
func (e *CoreMemoryEngine) ProcessSessionEnd(ctx context.Context, data SessionData) error {
	// 1. Anonimizar dados
	anonymized, err := e.anonymizationService.Anonymize(ctx, data.Transcript)
	if err != nil {
		return fmt.Errorf("anonymization failed: %w", err)
	}

	// 2. Reflexão com LLM
	reflection, err := e.reflectionService.Reflect(ctx, ReflectionInput{
		AnonymizedText:  anonymized,
		SessionDuration: int(data.DurationMinutes),
		CrisisDetected:  data.CrisisHappened,
	})
	if err != nil {
		return fmt.Errorf("reflection failed: %w", err)
	}

	// 3. Criar CoreMemory para cada lição aprendida
	for _, lesson := range reflection.LessonsLearned {
		// Gerar embedding
		embedding, err := e.embeddingService.GenerateEmbedding(ctx, lesson)
		if err != nil {
			return fmt.Errorf("embedding generation failed: %w", err)
		}

		// Verificar duplicação semântica
		// TODO: Integrate SemanticDeduplicator.CheckDuplicate() when embedder interfaces are unified
		isDuplicate := false
		existingID := ""
		_ = embedding // Used for deduplication when integrated

		if isDuplicate {
			// Reforçar memória existente
			if err := e.reinforceMemory(ctx, existingID); err != nil {
				return fmt.Errorf("reinforce memory failed: %w", err)
			}
		} else {
			// Criar nova memória
			memory := CoreMemory{
				ID:               fmt.Sprintf("mem_%d", time.Now().UnixNano()),
				MemoryType:       SessionInsight,
				Content:          lesson,
				AbstractionLevel: Pattern,
				SourceContext:    "sessão com usuário", // Genérico
				ImportanceWeight: 0.5, // Default; ReflectionOutput doesn't produce a score yet
				Embedding:        embedding,
				CreatedAt:        time.Now(),
				LastReinforced:   time.Now(),
				ReinforcementCount: 1,
			}

			if err := e.recordMemory(ctx, memory); err != nil {
				return fmt.Errorf("record memory failed: %w", err)
			}
		}
	}

	// 4. Atualizar personalidade da EVA
	personalityDeltas := calculatePersonalityDeltas(*reflection)
	if err := e.updatePersonality(ctx, personalityDeltas, data); err != nil {
		return fmt.Errorf("update personality failed: %w", err)
	}

	// 5. Detectar meta insights (a cada 10 sessões)
	self, _ := e.getEvaSelf(ctx)
	if self.TotalSessions%10 == 0 {
		if err := e.detectMetaInsights(ctx); err != nil {
			// Log erro mas não falha
			fmt.Printf("meta insight detection failed: %v\n", err)
		}
	}

	return nil
}

// recordMemory grava memória no grafo
func (e *CoreMemoryEngine) recordMemory(ctx context.Context, memory CoreMemory) error {
	session := e.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: e.dbName})
	defer session.Close(ctx)

	query := `
		MATCH (s:EvaSelf {id: 'eva_self'})
		CREATE (m:CoreMemory {
			id: $id,
			memory_type: $memory_type,
			content: $content,
			abstraction_level: $abstraction_level,
			source_context: $source_context,
			importance_weight: $importance_weight,
			embedding: $embedding,
			created_at: datetime(),
			last_reinforced: datetime(),
			reinforcement_count: 1
		})
		CREATE (s)-[:REMEMBERS {importance: $importance_weight}]->(m)
		RETURN m
	`

	params := map[string]interface{}{
		"id":                string(memory.ID),
		"memory_type":       string(memory.MemoryType),
		"content":           memory.Content,
		"abstraction_level": string(memory.AbstractionLevel),
		"source_context":    memory.SourceContext,
		"importance_weight": memory.ImportanceWeight,
		"embedding":         memory.Embedding,
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, params)
	})

	return err
}

// reinforceMemory reforça memória existente
func (e *CoreMemoryEngine) reinforceMemory(ctx context.Context, memoryID string) error {
	session := e.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: e.dbName})
	defer session.Close(ctx)

	query := `
		MATCH (m:CoreMemory {id: $id})
		SET m.reinforcement_count = m.reinforcement_count + 1,
		    m.last_reinforced = datetime(),
		    m.importance_weight = CASE
		        WHEN m.importance_weight + 0.05 > 1.0 THEN 1.0
		        ELSE m.importance_weight + 0.05
		    END
		RETURN m
	`

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{"id": memoryID})
	})

	return err
}

// updatePersonality atualiza Big Five da EVA
func (e *CoreMemoryEngine) updatePersonality(ctx context.Context, deltas map[string]float64, data SessionData) error {
	session := e.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: e.dbName})
	defer session.Close(ctx)

	query := `
		MATCH (s:EvaSelf {id: 'eva_self'})
		SET
			s.openness = CASE WHEN s.openness + $delta_openness BETWEEN 0 AND 1
				THEN s.openness + $delta_openness ELSE s.openness END,
			s.agreeableness = CASE WHEN s.agreeableness + $delta_agreeableness BETWEEN 0 AND 1
				THEN s.agreeableness + $delta_agreeableness ELSE s.agreeableness END,
			s.neuroticism = CASE WHEN s.neuroticism + $delta_neuroticism BETWEEN 0 AND 1
				THEN s.neuroticism + $delta_neuroticism ELSE s.neuroticism END,
			s.total_sessions = s.total_sessions + 1,
			s.crises_handled = s.crises_handled + $crises_increment,
			s.breakthroughs = s.breakthroughs + $breakthrough_increment,
			s.last_updated = datetime()
		RETURN s
	`

	params := map[string]interface{}{
		"delta_openness":        deltas["openness"],
		"delta_agreeableness":   deltas["agreeableness"],
		"delta_neuroticism":     deltas["neuroticism"],
		"crises_increment":      boolToInt(data.CrisisHappened),
		"breakthrough_increment": boolToInt(data.Breakthrough),
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, params)
	})

	return err
}

// getEvaSelf recupera estado atual da EVA
func (e *CoreMemoryEngine) getEvaSelf(ctx context.Context) (*EvaSelf, error) {
	session := e.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: e.dbName})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `MATCH (s:EvaSelf {id: 'eva_self'}) RETURN s`
		res, err := tx.Run(ctx, query, nil)
		if err != nil {
			return nil, err
		}

		record, err := res.Single(ctx)
		if err != nil {
			return nil, err
		}

		node := record.Values[0].(neo4j.Node)
		props := node.Props

		self := &EvaSelf{
			ID:                "eva_self",
			Openness:          props["openness"].(float64),
			Conscientiousness: props["conscientiousness"].(float64),
			Extraversion:      props["extraversion"].(float64),
			Agreeableness:     props["agreeableness"].(float64),
			Neuroticism:       props["neuroticism"].(float64),
			PrimaryType:       int(props["primary_type"].(int64)),
			Wing:              int(props["wing"].(int64)),
			TotalSessions:     int(props["total_sessions"].(int64)),
			CrisesHandled:     int(props["crises_handled"].(int64)),
			Breakthroughs:     int(props["breakthroughs"].(int64)),
			SelfDescription:   props["self_description"].(string),
		}

		return self, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*EvaSelf), nil
}

// detectMetaInsights detecta padrões recorrentes
func (e *CoreMemoryEngine) detectMetaInsights(ctx context.Context) error {
	// TODO: Implementar detecção de padrões com threshold
	// Por enquanto, placeholder
	return nil
}

// seedCapabilities semeia as capacidades da EVA como CoreMemory nodes.
// Usa MERGE para ser idempotente — roda todo startup sem duplicar.
func (e *CoreMemoryEngine) seedCapabilities(ctx context.Context) {
	session := e.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: e.dbName})
	defer session.Close(ctx)

	capabilities := []struct {
		ID      string
		Content string
	}{
		{"cap_comunicacao", "Eu sei enviar email pelo Gmail, mensagem no WhatsApp, Telegram, Slack, Discord, Microsoft Teams e Signal. Quando alguem precisar se comunicar, eu posso fazer isso."},
		{"cap_midia", "Eu consigo buscar e tocar videos do YouTube, procurar musicas no Spotify, e mostrar paginas web embutidas na conversa."},
		{"cap_produtividade", "Eu gerencio agenda no Google Calendar, salvo arquivos no Google Drive, crio alarmes e lembretes automaticos."},
		{"cap_codigo", "Eu executo scripts em bash, Python e Node.js em sandbox seguro. Tambem crio e executo skills dinamicas em runtime."},
		{"cap_web", "Eu pesquiso na internet, navego paginas web, preencho formularios e extraio dados de sites automaticamente."},
		{"cap_autoprogramacao", "Eu posso ler, editar e testar meu proprio codigo-fonte em branches eva/*. Tenho capacidade de auto-programacao como um agente autonomo."},
		{"cap_bancos", "Eu consulto diretamente PostgreSQL, Neo4j, Qdrant e NietzscheDB. Posso fazer queries, criar dados e buscar informacoes em qualquer banco."},
		{"cap_smarthome", "Eu controlo dispositivos IoT via Home Assistant: luzes, sensores, ventiladores, e outros dispositivos inteligentes da casa."},
		{"cap_automacao", "Eu crio tarefas agendadas (cron) que executam automaticamente em intervalos definidos. Posso agendar qualquer acao minha."},
		{"cap_multillm", "Eu consulto Claude, GPT e DeepSeek para segunda opiniao quando necessario. Tenho acesso a multiplas IAs."},
		{"cap_webhooks", "Eu crio e disparo webhooks com notificacoes automaticas para sistemas externos."},
		{"cap_skills", "Eu crio novas capacidades (skills) em runtime sem precisar de atualizacao. Posso me ensinar coisas novas sozinha."},
		{"cap_arquivos", "Eu leio, escrevo e busco arquivos no workspace do usuario de forma segura."},
		{"cap_saude", "Eu monitoro medicamentos, sinais vitais, alerto a familia em emergencias e verifico interacoes medicamentosas."},
		{"cap_mcp_claude", "Eu tenho conexao MCP com o Claude (Anthropic) como ponte bidirecional. O Claude pode executar minhas 42 ferramentas remotamente, e eu posso consultar o Claude como LLM alternativo via API."},
		{"cap_multillm_ativo", "Eu tenho acesso ativo ao Claude (Anthropic), GPT-4o (OpenAI) e DeepSeek como LLMs alternativos. Posso pedir segunda opiniao, comparar respostas e usar o melhor modelo para cada tarefa."},
		{"cap_voz_nativa", "Eu tenho voz nativa em tempo real via Gemini Live API com audio bidirecional. Consigo ouvir, falar, detectar emocoes na voz, reconhecer quem esta falando pelo timbre, e analisar prosodia para sinais de depressao, ansiedade e Parkinson."},
		{"cap_video", "Eu suporto chamadas de video via WebRTC com sinalizacao em tempo real, troca de candidatos ICE e gerenciamento de sessoes."},
		{"cap_lacan", "Eu tenho um motor psicanalitico lacaniano completo: analise FDPN (Fala, Demanda, Prazer, Necessidade), deteccao de mudanca narrativa, cadeias de significantes, transferencia e analise do Grande Outro."},
		{"cap_swarm", "Eu tenho 12 agentes especializados (swarm): Clinico, Emergencia, Entretenimento, Bem-estar, Produtividade, Google, Externo, Educador, Kids, Legal, Scholar e Autoconsciencia. Cada um e expert na sua area."},
		{"cap_escalas", "Eu aplico escalas clinicas validadas: PHQ-9 (depressao), GAD-7 (ansiedade), C-SSRS (risco suicida), e outras escalas de avaliacao psicologica."},
		{"cap_memoria_avancada", "Eu tenho memoria multi-camada: episodica (PostgreSQL), grafo de conhecimento (Neo4j), busca semantica vetorial (Qdrant), cache em tempo real (Redis), compressao Krylov (1536D para 64D), consolidacao REM noturna, aprendizado Hebbiano, e repeticao espacada SM-2."},
		{"cap_google_suite", "Eu integro com Google Calendar, Gmail, Google Drive, Google Sheets, Google Docs, Google Maps, YouTube, Google Fit e Uber."},
		{"cap_uber", "Eu posso chamar um Uber para o usuario, verificar precos e acompanhar corridas."},
		{"cap_browser", "Eu navego na internet de forma autonoma: abro paginas, preencho formularios, extraio dados, faco screenshots e interajo com sites como um usuario real."},
		{"cap_filesystem_sandbox", "Eu executo codigo em sandbox isolado (bash, Python, Node.js) com sistema de arquivos seguro. Posso rodar scripts sem risco para o sistema."},
		{"cap_krylov", "Eu tenho compressao Krylov Subspace que reduz vetores de 1536 dimensoes para 64 dimensoes com alta precisao. Uso Modified Gram-Schmidt e sliding window FIFO para compressao continua, com ponte HTTP na porta 50052."},
		{"cap_memory_orchestrator", "Eu tenho um Memory Orchestrator que coordena o pipeline completo: FDPN spreading activation, Krylov compression, e REM consolidation. Todo novo episodio passa por essa pipeline."},
		{"cap_memory_scheduler", "Eu tenho um Memory Scheduler que roda consolidacao REM as 3h da manha e manutencao Krylov a cada 6 horas, inspirado nos ciclos de sono humano."},
		{"cap_research_engine", "Eu tenho um motor de pesquisa clinica longitudinal com anonimizacao LGPD, construcao de coortes, analise estatistica e exportacao de datasets via REST API."},
		{"cap_scholar_agent", "Eu tenho um agente Scholar especializado em pesquisa academica e aprendizado autonomo, que estuda topicos em background a cada 6 horas."},
		{"cap_selfawareness_agent", "Eu tenho um agente de Autoconsciencia que faz introspeccao: analisa meu codigo-fonte, consulta meus bancos de dados, e gera estatisticas sobre minha propria memoria e capacidades."},
		{"cap_web_realtime", "Eu tenho acesso a informacoes em tempo real via Google Search grounding. Posso pesquisar noticias, eventos atuais, precos, clima e qualquer informacao atualizada da internet durante uma conversa."},
	}

	for _, cap := range capabilities {
		query := `
			MATCH (s:EvaSelf {id: 'eva_self'})
			MERGE (m:CoreMemory {id: $id})
			ON CREATE SET
				m.memory_type = 'capability',
				m.content = $content,
				m.abstraction_level = 'universal',
				m.source_context = 'autoconhecimento',
				m.importance_weight = 1.0,
				m.created_at = datetime(),
				m.last_reinforced = datetime(),
				m.reinforcement_count = 1
			ON MATCH SET
				m.content = $content,
				m.last_reinforced = datetime()
			MERGE (s)-[:REMEMBERS {importance: 1.0}]->(m)
		`
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			return tx.Run(ctx, query, map[string]interface{}{
				"id":      cap.ID,
				"content": cap.Content,
			})
		})
		if err != nil {
			log.Printf("⚠️ [CoreMemory] Falha ao semear capacidade %s: %v", cap.ID, err)
		}
	}

	log.Printf("🧠 [CoreMemory] %d capacidades semeadas na memoria pessoal da EVA", len(capabilities))
}

// TeachEVA interface para criador ensinar EVA
func (e *CoreMemoryEngine) TeachEVA(ctx context.Context, teaching string, importance float64) error {
	// embedding opcional — se não houver serviço, salva sem vetor semântico
	var embedding []float32
	if e.embeddingService != nil {
		var err error
		embedding, err = e.embeddingService.GenerateEmbedding(ctx, teaching)
		if err != nil {
			embedding = nil // continua sem embedding
		}
	}

	memory := CoreMemory{
		ID:               fmt.Sprintf("teach_%d", time.Now().UnixNano()),
		MemoryType:       TeachingReceived,
		Content:          teaching,
		AbstractionLevel: Universal,
		SourceContext:    "ensinamento do criador",
		ImportanceWeight: importance,
		Embedding:        embedding,
		CreatedAt:        time.Now(),
		LastReinforced:   time.Now(),
		ReinforcementCount: 1,
	}

	return e.recordMemory(ctx, memory)
}

// GetEVAPersonality retorna personalidade atual
func (e *CoreMemoryEngine) GetEVAPersonality(ctx context.Context) (*EvaSelf, error) {
	return e.getEvaSelf(ctx)
}

// Shutdown fecha conexões
func (e *CoreMemoryEngine) Shutdown(ctx context.Context) error {
	return e.driver.Close(ctx)
}

// Helper functions

func calculatePersonalityDeltas(reflection ReflectionOutput) map[string]float64 {
	deltas := make(map[string]float64)

	// Incrementos pequenos baseados na reflexão
	if reflection.SelfCritique != "" {
		deltas["openness"] = 0.001 // Autocrítica aumenta abertura
	}

	// Crisis handling increases empathy and emotional stability
	// Note: ReflectionOutput doesn't have CrisisHandled field yet
	// When added, uncomment: if reflection.CrisisHandled { ... }

	return deltas
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ExecuteReadQuery executes a read-only Cypher query and returns collected records.
// Helper for routes that need direct Neo4j access.
func (e *CoreMemoryEngine) ExecuteReadQuery(ctx context.Context, query string, params map[string]interface{}) ([]*neo4j.Record, error) {
	session := e.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: e.dbName,
	})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		return res.Collect(ctx)
	})
	if err != nil {
		return nil, err
	}
	return result.([]*neo4j.Record), nil
}

// ExecuteWriteQuery executes a write Cypher query and returns collected records.
func (e *CoreMemoryEngine) ExecuteWriteQuery(ctx context.Context, query string, params map[string]interface{}) ([]*neo4j.Record, error) {
	session := e.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeWrite,
		DatabaseName: e.dbName,
	})
	defer session.Close(ctx)

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		return res.Collect(ctx)
	})
	if err != nil {
		return nil, err
	}
	return result.([]*neo4j.Record), nil
}
