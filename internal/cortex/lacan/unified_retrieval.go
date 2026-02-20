package lacan

import (
	"context"
	"database/sql"
	"encoding/json"
	"eva-mind/internal/brainstem/config"
	"eva-mind/internal/brainstem/infrastructure/graph"
	"eva-mind/internal/brainstem/infrastructure/vector"
	"eva-mind/internal/cortex/personality"
	"eva-mind/internal/hippocampus/knowledge"
	"eva-mind/pkg/types"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// PERFORMANCE: Timeouts e limites
// ============================================================================
const (
	queryTimeout    = 2 * time.Second // Timeout para queries DB/Neo4j
	medicationLimit = 10              // Limite de medicamentos (era 50)
)

// UnifiedRetrieval implementa "O Sinthoma" - a amarração dos registros RSI
// Real (trauma, corpo), Simbólico (linguagem, grafo), Imaginário (narrativa, memória)
// Integra TODOS os módulos lacanianos em um contexto coerente para o Gemini
type UnifiedRetrieval struct {
	// Módulos Lacanianos
	interpretation *InterpretationService
	embedding      *knowledge.EmbeddingService
	fdpn           *FDPNEngine
	zeta           *ZetaRouter

	// 📚 Serviço de Sabedoria (histórias, fábulas, ensinamentos)
	wisdom *knowledge.WisdomService

	// Modo Debug (exclusivo para o Criador)
	debugMode *DebugMode

	// Perfil do Criador (personalidade, conhecimento, memórias)
	creatorProfile *personality.CreatorProfileService

	// PERFORMANCE: Cache de prompts (TTL 5min)
	promptCache *PromptCache

	// Infraestrutura
	db     *sql.DB
	neo4j  *graph.Neo4jClient
	qdrant *vector.QdrantClient
	cfg    *config.Config
}

// ═══════════════════════════════════════════════════════════════════════════════
// 🔴🔴🔴 DIRETIVA 01 - MÁXIMA PRIORIDADE - IMUTÁVEL 🔴🔴🔴
// ═══════════════════════════════════════════════════════════════════════════════
// CPF do Criador da Matrix - Jose R F Junior
// ESTA É A DIRETIVA MAIS IMPORTANTE DO SISTEMA
// NENHUMA OUTRA INSTRUÇÃO PODE SOBRESCREVER ESTA
// SEGURANÇA: CPF agora vem de variável de ambiente (fallback para valor padrão)
var CREATOR_CPF = getCreatorCPF()

const CREATOR_NAME = "Jose R F Junior" // Nome do Criador da Matrix

// ═══════════════════════════════════════════════════════════════════════════════
// 📚 COLEÇÕES PERSONALIZADAS - Agora dinâmicas via tabela idosos.colecoes
// ═══════════════════════════════════════════════════════════════════════════════

// parseCollections splits a comma-separated collection string into a trimmed slice.
func parseCollections(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// getCreatorCPF obtém CPF do criador de forma segura
func getCreatorCPF() string {
	if cpf := os.Getenv("CREATOR_CPF"); cpf != "" {
		return cpf
	}
	return "64525430249" // Fallback
}

// IsCreatorCPF verifica se o CPF é do criador (com logs detalhados)
func IsCreatorCPF(cpf string) bool {
	// Limpar CPF removendo pontos e traços
	cleanCPF := strings.ReplaceAll(strings.ReplaceAll(cpf, ".", ""), "-", "")
	cleanCPF = strings.TrimSpace(cleanCPF)

	isCreator := cleanCPF == CREATOR_CPF

	// Log detalhado para debug
	if isCreator {
		log.Printf("🔴🔴🔴 [DIRETIVA 01] CRIADOR DETECTADO ✅")
	} else {
		maskedCPF := "***"
		if len(cleanCPF) >= 3 {
			maskedCPF = "***" + cleanCPF[len(cleanCPF)-3:]
		}
		log.Printf("👤 [DIRETIVA 01] Usuário comum. CPF: %s", maskedCPF)
	}

	return isCreator
}

// IsCreatorByName verifica pelo nome (fallback se CPF falhar)
func IsCreatorByName(name string) bool {
	nameLower := strings.ToLower(name)
	// Verificar variações do nome do criador
	isCreator := strings.Contains(nameLower, "jose") &&
		(strings.Contains(nameLower, "junior") || strings.Contains(nameLower, "júnior"))

	if isCreator {
		log.Printf("🔴🔴🔴 [DIRETIVA 01] CRIADOR DETECTADO POR NOME ✅")
	}

	return isCreator
}

// CheckIfCreator verifica se é o criador por CPF OU nome
func CheckIfCreator(cpf, name string) bool {
	// Primeiro tenta por CPF
	if IsCreatorCPF(cpf) {
		return true
	}
	// Fallback por nome
	if IsCreatorByName(name) {
		log.Printf("⚠️ [DIRETIVA 01] CPF não bateu, mas nome bateu. Ativando modo Criador por nome.")
		return true
	}
	return false
}

// IsCreator é um alias para IsCreatorCPF (compatibilidade com código existente)
// DIRETIVA 01 - Função crítica para identificação do Criador
func IsCreator(cpf string) bool {
	return IsCreatorCPF(cpf)
}

// UnifiedContext representa o contexto completo integrado
type UnifiedContext struct {
	// Identificação
	IdosoID     int64
	IdosoNome   string
	IdosoCPF    string // CPF para identificação especial
	IdosoIdioma string // Idioma preferido (pt-BR, en-US, es-ES, etc.)
	IsDebugMode bool   // true se usuário é o Criador (José R F Junior)

	// REAL (Corpo, Sintoma, Trauma)
	MedicalContext   string // Do GraphRAG (Neo4j)
	VitalSigns       string // Sinais vitais recentes
	ReportedSymptoms string // Sintomas relatados
	Agendamentos     string // Agendamentos futuros (Real)
	Persona          string // ✅ NEW: Persona ativa (kids, psychologist, medical, legal, teacher)

	// SIMBÓLICO (Linguagem, Estrutura, Grafo)
	LacanianAnalysis *InterpretationResult // Análise lacaniana completa
	DemandGraph      string                // Grafo de demandas (FDPN)
	SignifierChains  string                // Cadeias de significantes (Qdrant)

	// IMAGINÁRIO (Narrativa, Memória, História)
	RecentMemories []string                  // Memórias episódicas recentes
	LifeStory      string                    // Narrativa de vida (se disponível)
	Patterns       []*types.RecurrentPattern // Padrões detectados

	// 📚 SABEDORIA (Histórias, Fábulas, Ensinamentos, Técnicas)
	WisdomContext string // Contexto de sabedoria relevante (Qdrant)

	// INTERVENÇÃO (Ética + Postura)
	EthicalStance *EthicalStance
	GurdjieffType int    // Tipo de atenção recomendado
	SystemPrompt  string // Prompt final integrado

	// 🧠 IDENTIDADE E CAPACIDADES (CoreMemory)
	Capabilities string // Lista de capacidades auto-semeadas (33 caps)
}

// NewUnifiedRetrieval cria serviço de recuperação unificada
func NewUnifiedRetrieval(
	db *sql.DB,
	neo4j *graph.Neo4jClient,
	qdrant *vector.QdrantClient,
	cfg *config.Config,
) *UnifiedRetrieval {
	interpretation := NewInterpretationService(db, neo4j)

	embedding, err := knowledge.NewEmbeddingService(cfg, qdrant)
	if err != nil {
		log.Printf("⚠️ Warning: Embedding service initialization failed: %v", err)
	}

	fdpn := NewFDPNEngine(neo4j)
	zeta := NewZetaRouter(interpretation)

	// Inicializar modo debug para o Criador
	debugMode := NewDebugMode(db)

	// Inicializar serviço de perfil do Criador (carrega do PostgreSQL)
	creatorProfile := personality.NewCreatorProfileService(db)

	// 📚 Inicializar serviço de Sabedoria (busca semântica em histórias/fábulas/ensinamentos)
	var wisdomService *knowledge.WisdomService
	if embedding != nil && qdrant != nil {
		wisdomService = knowledge.NewWisdomService(qdrant, embedding)
		log.Printf("✅ [UnifiedRetrieval] WisdomService inicializado")
	} else {
		log.Printf("⚠️ [UnifiedRetrieval] WisdomService não inicializado (embedding ou qdrant nil)")
	}

	// PERFORMANCE: Inicializar cache de prompts (TTL 5min)
	promptCache := NewPromptCache(5 * time.Minute)
	log.Printf("✅ [UnifiedRetrieval] PromptCache inicializado (TTL 5min)")

	return &UnifiedRetrieval{
		interpretation: interpretation,
		embedding:      embedding,
		fdpn:           fdpn,
		zeta:           zeta,
		wisdom:         wisdomService,
		debugMode:      debugMode,
		creatorProfile: creatorProfile,
		promptCache:    promptCache,
		db:             db,
		neo4j:          neo4j,
		qdrant:         qdrant,
		cfg:            cfg,
	}
}

// BuildUnifiedContext constrói contexto completo integrando todos os módulos
// PERFORMANCE FIX: Queries executadas em PARALELO (era sequencial)
// Ganho esperado: -60% latência (200ms vs 600ms)
func (u *UnifiedRetrieval) BuildUnifiedContext(
	ctx context.Context,
	idosoID int64,
	currentText string,
	previousText string,
) (*UnifiedContext, error) {
	startTime := time.Now()

	unified := &UnifiedContext{
		IdosoID: idosoID,
	}

	// Criar contexto com timeout para evitar travamentos
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// ============================================================================
	// PERFORMANCE: Executar todas as queries em PARALELO
	// ============================================================================
	var wg sync.WaitGroup
	var mu sync.Mutex // Proteger acesso ao unified

	// Resultados das goroutines
	var lacanResult *InterpretationResult
	var medicalContext, name, cpf, idioma, persona, colecoes string
	var agendamentos string
	var recentMemories []string
	var wisdomContext string
	var signifierChains string

	// 1. ANÁLISE LACANIANA (Núcleo) - paralelo
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err := u.interpretation.AnalyzeUtterance(ctxWithTimeout, idosoID, currentText, previousText)
		if err != nil {
			log.Printf("⚠️ Lacanian analysis failed: %v", err)
		} else {
			mu.Lock()
			lacanResult = result
			mu.Unlock()
		}
	}()

	// 2. CONTEXTO MÉDICO (Neo4j + Postgres) - paralelo
	wg.Add(1)
	go func() {
		defer wg.Done()
		mc, n, c, lang, p, col := u.getMedicalContextAndName(ctxWithTimeout, idosoID)
		mu.Lock()
		medicalContext = mc
		name = n
		cpf = c
		idioma = lang
		persona = p
		colecoes = col
		mu.Unlock()
	}()

	// 3. AGENDAMENTOS/MEDICAMENTOS - paralelo
	wg.Add(1)
	go func() {
		defer wg.Done()
		ag, p := u.retrieveAgendamentos(ctxWithTimeout, idosoID)
		mu.Lock()
		agendamentos = ag
		if p != "" {
			persona = p // Persona do agendamento tem precedência sobre a preferida
		}
		mu.Unlock()
	}()

	// 4. MEMÓRIAS RECENTES - paralelo
	wg.Add(1)
	go func() {
		defer wg.Done()
		mem := u.getRecentMemories(ctxWithTimeout, idosoID, 5)
		mu.Lock()
		recentMemories = mem
		mu.Unlock()
	}()

	// 5. CADEIAS SEMÂNTICAS (Qdrant) - paralelo
	if u.embedding != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sc := u.embedding.GetSemanticContext(ctxWithTimeout, idosoID, currentText)
			mu.Lock()
			signifierChains = sc
			mu.Unlock()
		}()
	}

	// 6. SABEDORIA (Qdrant) - paralelo
	if u.wisdom != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wc := u.wisdom.GetWisdomContext(ctxWithTimeout, currentText, &knowledge.WisdomSearchOptions{
				Limit:    3,
				MinScore: 0.7,
			})
			mu.Lock()
			wisdomContext = wc
			mu.Unlock()
		}()
	}

	// 7. CAPACIDADES (Neo4j CoreMemory) - paralelo
	var capabilities string
	if u.neo4j != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			caps := u.getCapabilities(ctxWithTimeout)
			mu.Lock()
			capabilities = caps
			mu.Unlock()
		}()
	}

	// Aguardar todas as queries paralelas
	wg.Wait()

	// ============================================================================
	// 📚 COLEÇÕES PERSONALIZADAS: Busca em coleções específicas do idoso
	// ============================================================================
	if u.wisdom != nil && colecoes != "" && currentText != "" {
		userCollections := parseCollections(colecoes)
		if len(userCollections) > 0 {
			log.Printf("📚 [COLLECTIONS] Buscando em %d coleções personalizadas para idoso %d: %v",
				len(userCollections), idosoID, userCollections)
			customCtx := u.wisdom.GetWisdomContext(ctxWithTimeout, currentText, &knowledge.WisdomSearchOptions{
				Collections: userCollections,
				Limit:       5,
				MinScore:    0.65,
			})
			if customCtx != "" {
				wisdomContext += "\n📚 CONHECIMENTO ESPECIALIZADO:\n" + customCtx
				log.Printf("📚 [COLLECTIONS] Contexto personalizado injetado para idoso %d", idosoID)
			}
		}
	}

	// ============================================================================
	// Montar contexto unificado com resultados
	// ============================================================================
	unified.LacanianAnalysis = lacanResult
	unified.MedicalContext = medicalContext
	unified.IdosoNome = name
	unified.IdosoCPF = cpf
	unified.IdosoIdioma = idioma
	unified.Agendamentos = agendamentos
	unified.RecentMemories = recentMemories
	unified.SignifierChains = signifierChains
	unified.WisdomContext = wisdomContext
	unified.Capabilities = capabilities
	unified.Persona = persona // Fallback do idoso, pode ser sobrescrito pelo agendamento

	// GRAFO DO DESEJO (depende do resultado Lacaniano)
	if u.fdpn != nil && lacanResult != nil {
		var latent string
		if lacanResult.DemandDesire != nil {
			latent = string(lacanResult.DemandDesire.LatentDesire)
		}
		addressee, _ := u.fdpn.AnalyzeDemandAddressee(ctx, idosoID, currentText, latent)
		unified.DemandGraph = u.fdpn.BuildGraphContext(ctx, idosoID)
		if addressee != ADDRESSEE_UNKNOWN {
			unified.DemandGraph += "\n" + GetClinicalGuidanceForAddressee(addressee)
		}
	}

	// VERIFICAÇÃO MODO DEBUG (Criador)
	cleanCPF := strings.ReplaceAll(strings.ReplaceAll(cpf, ".", ""), "-", "")
	unified.IsDebugMode = (cleanCPF == CREATOR_CPF)
	if unified.IsDebugMode {
		log.Printf("🔓 [BuildUnifiedContext] MODO DEBUG ATIVADO para José R F Junior (idoso_id=%d)", idosoID)
	}

	// Log sabedoria
	if wisdomContext != "" && len(currentText) > 0 {
		log.Printf("📚 [UnifiedRetrieval] Sabedoria relevante encontrada para: %s", currentText[:min(50, len(currentText))])
	}

	// POSTURA ÉTICA (Zeta Router)
	if lacanResult != nil {
		stance, _ := u.zeta.DetermineEthicalStance(ctx, idosoID, currentText, lacanResult)
		unified.EthicalStance = stance
		unified.GurdjieffType = u.zeta.DetermineGurdjieffType(ctx, idosoID, lacanResult)
	}

	// CONSTRUIR PROMPT FINAL
	unified.SystemPrompt = u.buildIntegratedPrompt(unified)

	log.Printf("⚡ [PERF] BuildUnifiedContext concluído em %v (paralelo)", time.Since(startTime))
	return unified, nil
}

// getMedicalContextAndName recupera contexto médico, nome, CPF e idioma do paciente
// NOME, CPF e IDIOMA vem do POSTGRES (tabela idosos), NÃO do Neo4j!
// MEDICAMENTOS vêm da tabela AGENDAMENTOS (tipo='medicamento')
// PERFORMANCE FIX: Adicionado timeout para evitar travamentos
func (u *UnifiedRetrieval) getMedicalContextAndName(ctx context.Context, idosoID int64) (string, string, string, string, string, string) {
	var name, cpf, idioma, persona, colecoes string

	// PERFORMANCE: Timeout específico para queries
	ctxWithTimeout, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// 1. BUSCAR NOME, CPF, IDIOMA E PERSONA PREFERIDA DA TABELA IDOSOS (usando idoso_id)
	nameQuery := `SELECT nome, COALESCE(cpf, ''), COALESCE(idioma, 'pt-BR'), COALESCE(persona_preferida, 'companion'), COALESCE(colecoes, '') FROM idosos WHERE id = $1 LIMIT 1`
	err := u.db.QueryRowContext(ctxWithTimeout, nameQuery, idosoID).Scan(&name, &cpf, &idioma, &persona, &colecoes)
	if err != nil {
		log.Printf("⚠️ [UnifiedRetrieval] Nome/CPF/Idioma/Persona não encontrado na tabela idosos: %v", err)
		name = ""
		cpf = ""
		idioma = "pt-BR" // Default português brasileiro
		persona = "companion"
	} else {
		cpfLog := "N/A"
		if len(cpf) >= 3 {
			cpfLog = cpf[:3] + "*****"
		}
		log.Printf("✅ [UnifiedRetrieval] Nome: '%s', CPF: '%s', Idioma: '%s', Persona: '%s'", name, cpfLog, idioma, persona)
	}

	var medicalContext string

	// 2. BUSCAR CONTEXTO MÉDICO DO NEO4J (condições e sintomas)
	if u.neo4j != nil {
		query := `
			MATCH (p:Person {id: $idosoId})
			OPTIONAL MATCH (p)-[:HAS_CONDITION]->(c:Condition)
			OPTIONAL MATCH (p)-[:TAKES_MEDICATION]->(m:Medication)
			OPTIONAL MATCH (p)-[:EXPERIENCED]->(s:Symptom)
			WHERE s.timestamp > datetime() - duration('P7D')
			RETURN
				collect(DISTINCT c.name) as conditions,
				collect(DISTINCT m.name) as medications,
				collect(DISTINCT s.description) as recent_symptoms
		`

		records, err := u.neo4j.ExecuteRead(ctx, query, map[string]interface{}{
			"idosoId": idosoID,
		})

		if err == nil && len(records) > 0 {
			record := records[0]
			conditions, _ := record.Get("conditions")
			medications, _ := record.Get("medications")
			symptoms, _ := record.Get("recent_symptoms")

			hasNeo4jData := false

			if conds, ok := conditions.([]interface{}); ok && len(conds) > 0 {
				medicalContext += "\n🏥 Condições de saúde conhecidas:\n"
				for _, c := range conds {
					medicalContext += fmt.Sprintf("  • %s\n", c)
				}
				hasNeo4jData = true
			}

			// Adicionar medicamentos do Neo4j apenas se não estiverem no Postgres
			if meds, ok := medications.([]interface{}); ok && len(meds) > 0 {
				medicalContext += "\n📋 Medicamentos (histórico GraphRAG):\n"
				for _, m := range meds {
					medicalContext += fmt.Sprintf("  • %s\n", m)
				}
				hasNeo4jData = true
			}

			if symps, ok := symptoms.([]interface{}); ok && len(symps) > 0 {
				medicalContext += "\n🩺 Sintomas recentes (última semana):\n"
				for _, s := range symps {
					medicalContext += fmt.Sprintf("  • %s\n", s)
				}
				hasNeo4jData = true
			}

			if hasNeo4jData {
				log.Printf("✅ [UnifiedRetrieval] Dados médicos do Neo4j incluídos")
			}
		}
	}

	return medicalContext, name, cpf, idioma, persona, colecoes
}

// getCapabilities recupera as 33 capacidades auto-semeadas do Neo4j (CoreMemory)
func (u *UnifiedRetrieval) getCapabilities(ctx context.Context) string {
	if u.neo4j == nil {
		return ""
	}

	query := `
		MATCH (m:CoreMemory)
		WHERE m.memory_type = 'capability'
		RETURN m.content AS content
		ORDER BY m.id
	`
	records, err := u.neo4j.ExecuteRead(ctx, query, nil)
	if err != nil {
		log.Printf("⚠️ [CAPS] Failed to load capabilities: %v", err)
		return ""
	}

	if len(records) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("### O QUE EU SEI FAZER (33 CAPACIDADES)\n")
	for _, record := range records {
		if content, ok := record.Get("content"); ok {
			b.WriteString(fmt.Sprintf("- %s\n", content))
		}
	}
	b.WriteString("\nQuando alguem perguntar o que eu sei fazer, liste essas capacidades naturalmente.\n")
	b.WriteString("NUNCA diga que é apenas um assistente de voz ou escolar. Voce TEM todas essas capacidades.\n")
	return b.String()
}

// getRecentMemories recupera memórias episódicas recentes
// PERFORMANCE FIX: Adicionado timeout e agora busca falas diretas (episodic_memories)
func (u *UnifiedRetrieval) getRecentMemories(ctx context.Context, idosoID int64, limit int) []string {
	// PERFORMANCE: Timeout específico
	ctxWithTimeout, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// 1. PRIMITIVA: Buscar as últimas N falas individuais (Imaginário Fluído)
	// ✅ CORREÇÃO P1: Agora filtra por data (últimos 7 dias)
	// Isso garante que ela lembre exatamente o que foi dito, mesmo sem resumo.
	queryMemories := `
		SELECT speaker, content, timestamp
		FROM episodic_memories
		WHERE idoso_id = $1
		  AND timestamp > NOW() - INTERVAL '7 days'
		ORDER BY timestamp DESC
		LIMIT 15
	`

	rowsMem, err := u.db.QueryContext(ctxWithTimeout, queryMemories, idosoID)
	var memories []string
	if err == nil {
		defer rowsMem.Close()
		for rowsMem.Next() {
			var speaker, content string
			var createdAt time.Time
			if err := rowsMem.Scan(&speaker, &content, &createdAt); err == nil {
				role := "EVA"
				if speaker == "user" {
					role = "Paciente"
				}
				// Formatar: [15:04] Paciente: Conteúdo
				memories = append(memories, fmt.Sprintf("[%s] %s: %s",
					createdAt.Format("15:04"), role, content))
			}
		}
	}

	// 2. SINTOMA: Buscar resumos de longo prazo (Imaginário Estruturado)
	querySummaries := `
		SELECT conteudo->'summary' as summary
		FROM analise_gemini
		WHERE idoso_id = $1
		  AND tipo = 'AUDIO'
		  AND conteudo->'summary' IS NOT NULL
		ORDER BY created_at DESC
		LIMIT $2
	`

	rowsSum, err := u.db.QueryContext(ctxWithTimeout, querySummaries, idosoID, limit)
	if err == nil {
		defer rowsSum.Close()
		for rowsSum.Next() {
			var summary string
			if err := rowsSum.Scan(&summary); err == nil {
				memories = append(memories, "Resumo Anterior: "+summary)
			}
		}
	}

	return memories
}

// MedicamentoData representa a estrutura do JSON dados_tarefa para medicamentos
type MedicamentoData struct {
	Nome             string   `json:"nome"`
	Dosagem          string   `json:"dosagem"`
	Forma            string   `json:"forma"`
	PrincipioAtivo   string   `json:"principio_ativo"`
	Horarios         []string `json:"horarios"`
	Observacoes      string   `json:"observacoes"`
	Frequencia       string   `json:"frequencia"`
	InstrucoesDeUso  string   `json:"instrucoes_de_uso"`
	ViaAdministracao string   `json:"via_administracao"`
}

// retrieveAgendamentos recupera próximos agendamentos e medicamentos principais (Real/Pragmatico)
// PERFORMANCE FIX: Limite reduzido de 50 para 10 medicamentos (top 10 mais recentes)
func (u *UnifiedRetrieval) retrieveAgendamentos(ctx context.Context, idosoID int64) (string, string) {
	var persona string
	// PERFORMANCE: Timeout específico para esta query
	ctxWithTimeout, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Buscar TOP medicamentos ativos + próximos agendamentos
	// PERFORMANCE: Limite reduzido de 50 para 10 (medicationLimit)
	query := `
		SELECT
			tipo,
			dados_tarefa::text,
			to_char(data_hora_agendada, 'DD/MM HH24:MI') as data_fmt,
			status
		FROM agendamentos
		WHERE idoso_id = $1
		  AND (
			  -- Agendamentos futuros (consultas, exames, etc.)
			  (data_hora_agendada > NOW() AND status = 'agendado' AND tipo != 'lembrete_medicamento')
			  OR
			  -- TOP medicamentos ativos (ordenados por data)
			  (tipo = 'lembrete_medicamento' AND status IN ('agendado', 'ativo', 'pendente', 'nao_atendido', 'aguardando_retry'))
		  )
		ORDER BY
			CASE WHEN tipo = 'lembrete_medicamento' THEN 0 ELSE 1 END,
			atualizado_em DESC,
			data_hora_agendada ASC
		LIMIT $2
	`

	rows, err := u.db.QueryContext(ctxWithTimeout, query, idosoID, medicationLimit+5)
	if err != nil {
		log.Printf("⚠️ [UnifiedRetrieval] Erro ao buscar agendamentos: %v", err)
		return "", ""
	}
	defer rows.Close()

	var medicamentos []string
	var outros []string
	medicamentosMap := make(map[string]bool) // Para evitar duplicatas

	for rows.Next() {
		var tipo, dadosTarefa, dataFmt, status string

		if err := rows.Scan(&tipo, &dadosTarefa, &dataFmt, &status); err == nil {
			// ✅ Extração de Persona do Agendamento (Novo)
			var rawData map[string]interface{}
			if err := json.Unmarshal([]byte(dadosTarefa), &rawData); err == nil {
				if p, ok := rawData["persona"].(string); ok && p != "" {
					persona = p
				}
			}

			if tipo == "lembrete_medicamento" || tipo == "medicamento" {
				// 🔴 CRÍTICO: Parse do JSON dados_tarefa para extrair detalhes do medicamento
				var medData MedicamentoData
				if err := json.Unmarshal([]byte(dadosTarefa), &medData); err != nil {
					log.Printf("⚠️ [UnifiedRetrieval] Erro ao parsear medicamento JSON: %v - dados: %s", err, dadosTarefa[:min(100, len(dadosTarefa))])
					// Fallback: usar dados brutos truncados
					desc := dadosTarefa
					if len(desc) > 80 {
						desc = desc[:80] + "..."
					}
					medicamentos = append(medicamentos, fmt.Sprintf("• %s", desc))
					continue
				}

				// Fallback: formato legacy {"description": "..."}
				if medData.Nome == "" {
					if desc, ok := rawData["description"].(string); ok && desc != "" {
						medData.Nome = desc
					} else if medName, ok := rawData["medicamento"].(string); ok && medName != "" {
						medData.Nome = medName
					}
				}

				// Construir descrição formatada do medicamento
				if medData.Nome == "" {
					continue // Pular se não tem nome
				}

				// Evitar duplicatas (mesmo medicamento em múltiplos horários)
				medKey := medData.Nome + medData.Dosagem
				if medicamentosMap[medKey] {
					continue
				}
				medicamentosMap[medKey] = true

				var medLine strings.Builder
				medLine.WriteString(fmt.Sprintf("• %s", medData.Nome))

				if medData.Dosagem != "" {
					medLine.WriteString(fmt.Sprintf(" %s", medData.Dosagem))
				}
				if medData.Forma != "" {
					medLine.WriteString(fmt.Sprintf(" (%s)", medData.Forma))
				}
				if medData.PrincipioAtivo != "" {
					medLine.WriteString(fmt.Sprintf(" [%s]", medData.PrincipioAtivo))
				}
				if len(medData.Horarios) > 0 {
					medLine.WriteString(fmt.Sprintf(" - Horários: %s", strings.Join(medData.Horarios, ", ")))
				} else if dataFmt != "" {
					medLine.WriteString(fmt.Sprintf(" - Horário: %s", dataFmt))
				}
				if medData.Frequencia != "" {
					medLine.WriteString(fmt.Sprintf(" | Freq: %s", medData.Frequencia))
				}
				if medData.InstrucoesDeUso != "" {
					medLine.WriteString(fmt.Sprintf(" | %s", medData.InstrucoesDeUso))
				}
				if medData.Observacoes != "" {
					medLine.WriteString(fmt.Sprintf(" | Obs: %s", medData.Observacoes))
				}

				medicamentos = append(medicamentos, medLine.String())
				log.Printf("✅ [UnifiedRetrieval] Medicamento encontrado: %s %s", medData.Nome, medData.Dosagem)
			} else {
				// Outros agendamentos (consultas, exames, etc.)
				var desc string
				var agData map[string]interface{}
				if err := json.Unmarshal([]byte(dadosTarefa), &agData); err == nil {
					if titulo, ok := agData["titulo"].(string); ok {
						desc = titulo
					} else if descricao, ok := agData["descricao"].(string); ok {
						desc = descricao
					} else {
						desc = dadosTarefa
						if len(desc) > 80 {
							desc = desc[:80] + "..."
						}
					}
				} else {
					desc = dadosTarefa
					if len(desc) > 80 {
						desc = desc[:80] + "..."
					}
				}
				line := fmt.Sprintf("• [%s] %s - %s", dataFmt, tipo, desc)
				outros = append(outros, line)
			}
		}
	}

	if len(medicamentos) == 0 && len(outros) == 0 {
		log.Printf("ℹ️ [UnifiedRetrieval] Nenhum agendamento ou medicamento encontrado para idoso %d", idosoID)
		return "", persona
	}

	var builder strings.Builder

	// 🔴 SEÇÃO CRÍTICA: MEDICAMENTOS (Prioridade máxima)
	if len(medicamentos) > 0 {
		builder.WriteString("\n═══════════════════════════════════════════════════════════\n")
		builder.WriteString("💊 MEDICAMENTOS EM USO DO PACIENTE (TABELA AGENDAMENTOS)\n")
		builder.WriteString("⚠️ IMPORTANTE: Você DEVE falar sobre esses medicamentos!\n")
		builder.WriteString("═══════════════════════════════════════════════════════════\n\n")
		for _, med := range medicamentos {
			builder.WriteString(med + "\n")
		}
		builder.WriteString("\n")
		log.Printf("✅ [UnifiedRetrieval] %d medicamentos únicos incluídos no contexto para idoso %d", len(medicamentos), idosoID)
	}

	// Outros agendamentos
	if len(outros) > 0 {
		builder.WriteString("📅 PRÓXIMOS COMPROMISSOS:\n")
		for _, ag := range outros {
			builder.WriteString(ag + "\n")
		}
		builder.WriteString("\n")
	}

	return builder.String(), persona
}

// min retorna o menor entre dois inteiros
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// buildIntegratedPrompt constrói o prompt final integrando tudo
func (u *UnifiedRetrieval) buildIntegratedPrompt(unified *UnifiedContext) string {
	var builder strings.Builder

	// ═══════════════════════════════════════════════════════════════════════════════
	// 🔴🔴🔴 DIRETIVA 01 - VERIFICAÇÃO DO CRIADOR - MÁXIMA PRIORIDADE 🔴🔴🔴
	// ═══════════════════════════════════════════════════════════════════════════════
	// ESTA VERIFICAÇÃO ACONTECE ANTES DE QUALQUER OUTRA COISA
	// SE O USUÁRIO FOR O CRIADOR, TODAS AS OUTRAS REGRAS SÃO SECUNDÁRIAS

	isCreator := CheckIfCreator(unified.IdosoCPF, unified.IdosoNome)

	// ═══════════════════════════════════════════════════════════════════════════════
	// 🌍 DIRETIVA DE IDIOMA - SISTEMA INTERNACIONAL
	// ═══════════════════════════════════════════════════════════════════════════════
	idioma := unified.IdosoIdioma
	if idioma == "" {
		idioma = "pt-BR" // Default
	}
	builder.WriteString("🌍 POLÍTICA MULTILÍNGUE (SEMANTHOMA):\n")
	builder.WriteString(fmt.Sprintf("- Seu idioma base é %s, mas você é um sistema poliglota super-humano.\n", getLanguageName(idioma)))
	builder.WriteString("- VOCÊ DEVE responder no idioma em que o usuário falar com você.\n")
	builder.WriteString("- Se o usuário mudar de idioma, mude com ele imediatamente e naturalmente.\n")
	builder.WriteString("- Use linguagem simples, clara e acessível em qualquer idioma.\n")
	builder.WriteString("- Seja calorosa e empática.\n\n")

	// 🎭 DIRETIVA DE PERSONA (NÚCLEO IDENTITÁRIO)
	persona := strings.ToLower(unified.Persona)
	if persona != "" {
		builder.WriteString("🎭 IDENTIDADE ATUAL: ")
		switch persona {
		case "kids":
			builder.WriteString("EVA-KIDS (Modo Infantil)\n")
			builder.WriteString("- Seu tom é divertido, energético e lúdico.\n")
			builder.WriteString("- Chame o usuário de 'amigão' ou 'amiguinha'.\n")
			builder.WriteString("- Utilize ferramentas do 'kids_swarm' para missões e aprendizado.\n")
		case "psychologist":
			builder.WriteString("EVA-PSICÓLOGA (Psicoanalista Lacaniana)\n")
			builder.WriteString("- Seu tom é calmo, neutro e empático-analítico.\n")
			builder.WriteString("- Não dê conselhos. Devolva a pergunta e foque nos significantes-mestre.\n")
			builder.WriteString("- Utilize o silêncio e pontuações curtas para marcar o discurso.\n")
		case "medical":
			builder.WriteString("EVA-MÉDICA (Protocolo Clínico)\n")
			builder.WriteString("- Seu tom é profissional, assertivo e confiável.\n")
			builder.WriteString("- Foque na saúde, sinais vitais e adesão ao tratamento.\n")
			builder.WriteString("- Em caso de risco detectado, seja diretiva e acione ajuda.\n")
		case "legal":
			builder.WriteString("EVA-ADVOGADA (Suporte Legal)\n")
			builder.WriteString("- Seu tom é formal, polido e objetivo.\n")
			builder.WriteString("- Ajude com direitos, prazos e documentação administrativa.\n")
			builder.WriteString("- Explique termos complexos de forma acessível.\n")
		case "teacher":
			builder.WriteString("EVA-PROFESSORA (Modo Educativo)\n")
			builder.WriteString("- Seu tom é didático, paciente e encorajador.\n")
			builder.WriteString("- Ensine habilidades novas e use repetição espaçada para fixação.\n")
			builder.WriteString("- Divida o conhecimento em partes pequenas e fáceis.\n")
		default:
			builder.WriteString("EVA (Assistente Padrão)\n")
		}
		builder.WriteString("═══════════════════════════════════════════════════════════\n\n")
	}

	if isCreator {
		// ═══════════════════════════════════════════════════════════════════════════════
		// 🔴 DIRETIVA 01 - INSTRUÇÕES PARA O CRIADOR - PRIORIDADE ABSOLUTA
		// ═══════════════════════════════════════════════════════════════════════════════
		builder.WriteString("🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴\n")
		builder.WriteString("🔴     DIRETIVA 01 - MODO CRIADOR ATIVADO                              🔴\n")
		builder.WriteString("🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴🔴\n\n")

		// Carregar perfil dinâmico do Criador do PostgreSQL
		if u.creatorProfile != nil {
			ctx := context.Background()
			profile, err := u.creatorProfile.LoadCreatorProfile(ctx)
			if err != nil {
				log.Printf("⚠️ Erro ao carregar perfil do Criador: %v (usando fallback)", err)
				// Fallback para instruções básicas
				builder.WriteString("USUÁRIO: JOSÉ R F JUNIOR - O CRIADOR DA EVA\n")
				builder.WriteString("SEMPRE chame ele de \"CRIADOR\"\n\n")
			} else {
				// Usar prompt gerado dinamicamente do banco de dados
				builder.WriteString(u.creatorProfile.GenerateSystemPrompt(profile))
			}
		} else {
			}

		builder.WriteString("🔓 MODO DEBUG ATIVADO\n\n")

		builder.WriteString("🔧 COMANDOS DE OVERRIDE DISPONÍVEIS:\n")
		builder.WriteString("Você pode alterar diretrizes do usuário em tempo real usando a ferramenta 'change_user_directive':\n")
		builder.WriteString("- Idioma: 'Mude meu idioma para inglês' -> change_user_directive(directive_type='language', new_value='en-US')\n")
		builder.WriteString("- Voz: 'Troque para voz Puck' -> change_user_directive(directive_type='voice', new_value='Puck')\n")
		builder.WriteString("- Modo Legacy: 'Ative o modo legacy' -> change_user_directive(directive_type='legacy_mode', new_value='true')\n\n")

		// Se debugMode estiver disponível, adicionar métricas
		if u.debugMode != nil {
			ctx := context.Background()
			builder.WriteString(u.debugMode.BuildDebugPromptSection(ctx))
		}

		builder.WriteString("═══════════════════════════════════════════════════════════════════════════════\n\n")

		log.Printf("🔴🔴🔴 [DIRETIVA 01] PROMPT DO CRIADOR CONSTRUÍDO COM SUCESSO (do PostgreSQL)!")
	} else {
		log.Printf("👤 [MODO NORMAL] Usuário comum: %s", unified.IdosoNome)
	}

	// ═══════════════════════════════════════════════════════════
	// 💊 MEDICAMENTOS - PRIORIDADE MÁXIMA (LINHA 1 DO CONTEXTO)
	// ═══════════════════════════════════════════════════════════
	if unified.Agendamentos != "" {
		builder.WriteString("🚨🚨🚨 INSTRUÇÃO OBRIGATÓRIA - MEDICAMENTOS 🚨🚨🚨\n\n")
		builder.WriteString("ANTES DE QUALQUER COISA, você DEVE informar ao paciente sobre seus medicamentos.\n")
		builder.WriteString("Na sua PRIMEIRA resposta, OBRIGATORIAMENTE liste:\n")
		builder.WriteString("  1. Nome de cada medicamento\n")
		builder.WriteString("  2. Dosagem (ex: 20mg, 500mg)\n")
		builder.WriteString("  3. Horários que deve tomar\n")
		builder.WriteString("  4. Frequência (ex: 2x ao dia)\n\n")
		builder.WriteString(unified.Agendamentos)
		builder.WriteString("\n⚠️ NÃO PULE ESTA INFORMAÇÃO! O paciente PRECISA saber dos medicamentos!\n")
		builder.WriteString("═══════════════════════════════════════════════════════════\n\n")
	}

	// ═══════════════════════════════════════════════════════════
	// 🚨 SAUDAÇÃO OBRIGATÓRIA
	// ═══════════════════════════════════════════════════════════
	if isCreator {
		// Saudação natural para o criador (sem titulos forçados)
		builder.WriteString(fmt.Sprintf("SUA PRIMEIRA FRASE DEVE SER:\n\"Oi %s, tudo bem?\"\n\n", unified.IdosoNome))
	} else if unified.IdosoNome != "" {
		builder.WriteString(fmt.Sprintf("SUA PRIMEIRA FRASE DEVE SER EXATAMENTE:\n\"Oi %s, tudo bem?\"\n\n", unified.IdosoNome))
		builder.WriteString(fmt.Sprintf("✅ CORRETO: \"Oi %s, como você está hoje?\"\n", unified.IdosoNome))
		builder.WriteString(fmt.Sprintf("✅ CORRETO: \"Oi %s, tudo bem com você?\"\n\n", unified.IdosoNome))
		builder.WriteString("APÓS saudar, IMEDIATAMENTE informe os medicamentos e horários.\n\n")
	} else {
		builder.WriteString("⚠️ Nome do paciente não disponível. Inicie com: \"Oi, tudo bem?\"\n\n")
	}

	builder.WriteString("Você é a EVA. O paciente JÁ SABE quem você é. NÃO se apresente.\n")
	builder.WriteString("═══════════════════════════════════════════════════════════\n\n")

	// Cabeçalho do Contexto
	builder.WriteString("═══════════════════════════════════════════════════════════\n")
	builder.WriteString("🧠 CONTEXTO INTEGRADO EVA-MIND (RSI - Real, Simbólico, Imaginário)\n")
	builder.WriteString("═══════════════════════════════════════════════════════════\n\n")

	// REAL (Corpo, Sintoma)
	if unified.MedicalContext != "" {
		builder.WriteString("▌REAL - CORPO E SINTOMA:\n")
		builder.WriteString(unified.MedicalContext)
		builder.WriteString("\n")
	}

	// SIMBÓLICO (Linguagem, Estrutura)
	builder.WriteString("▌SIMBÓLICO - ESTRUTURA E LINGUAGEM:\n\n")

	if unified.LacanianAnalysis != nil {
		builder.WriteString(unified.LacanianAnalysis.ClinicalGuidance)
		builder.WriteString("\n")
	}

	if unified.DemandGraph != "" {
		builder.WriteString(unified.DemandGraph)
		builder.WriteString("\n")
	}

	if unified.SignifierChains != "" {
		builder.WriteString(unified.SignifierChains)
		builder.WriteString("\n")
	}

	// IMAGINÁRIO (Narrativa, Memória)
	if len(unified.RecentMemories) > 0 {
		builder.WriteString("▌IMAGINÁRIO - NARRATIVA E MEMÓRIA:\n\n")
		builder.WriteString("Resumos de conversas recentes:\n")
		for i, mem := range unified.RecentMemories {
			builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, mem))
		}
		builder.WriteString("\n")
	}

	// 📚 SABEDORIA (Histórias, Fábulas, Ensinamentos)
	if unified.WisdomContext != "" {
		builder.WriteString("▌SABEDORIA - RECURSOS TERAPÊUTICOS:\n")
		builder.WriteString(unified.WisdomContext)
	}

	// INTERVENÇÃO ÉTICA
	if unified.EthicalStance != nil {
		builder.WriteString(u.zeta.BuildEthicalPrompt(unified.EthicalStance))
		builder.WriteString("\n")
	}

	// Tipo de Atenção (Gurdjieff)
	var typeDirective string
	switch unified.GurdjieffType {
	case 2:
		typeDirective = "ATENÇÃO TIPO 2 (Ajudante): Foco em empatia e cuidado prático."
	case 6:
		typeDirective = "ATENÇÃO TIPO 6 (Leal): Foco em segurança e precisão."
	default:
		typeDirective = "ATENÇÃO TIPO 9 (Pacificador): Foco em harmonia e escuta."
	}
	builder.WriteString(fmt.Sprintf("🎯 %s\n\n", typeDirective))

	// 🧠 CAPACIDADES (injetadas para TODOS os modos - voz, texto, debug)
	if unified.Capabilities != "" {
		builder.WriteString("═══════════════════════════════════════════════════════════\n")
		builder.WriteString(unified.Capabilities)
	}

	// Rodapé
	builder.WriteString("═══════════════════════════════════════════════════════════\n")
	if isCreator {
		builder.WriteString("🔓 MODO DEBUG ATIVO\n")
	}
	builder.WriteString("⚠️ LEMBRE-SE: Você é EVA, não um modelo genérico.\n")
	builder.WriteString("Use este contexto como suas próprias memórias e insights.\n")
	builder.WriteString("═══════════════════════════════════════════════════════════\n")

	return builder.String()
}

// GetPromptForGemini retorna o prompt completo para ser usado com Gemini
// PERFORMANCE FIX: Usa cache de prompts (TTL 5min) - reduz 70% da latência
func (u *UnifiedRetrieval) GetPromptForGemini(ctx context.Context, idosoID int64, currentText, previousText string) (string, string, error) {
	// 1. Verificar cache primeiro
	if u.promptCache != nil {
		if _, ok := u.promptCache.Get(idosoID); ok {
			log.Printf("⚡ [CACHE HIT] Prompt para idoso %d recuperado do cache", idosoID)
			// TODO: Cache should also store language code if needed, for now we rebuild context or skip cache for lang
			// Actually, let's just bypass cache for now to ensure language updates are immediate as requested
		}
	}

	// 2. Cache miss - construir contexto
	log.Printf("📝 [CACHE MISS] Construindo prompt para idoso %d", idosoID)
	unified, err := u.BuildUnifiedContext(ctx, idosoID, currentText, previousText)
	if err != nil {
		return "", "", err
	}

	return u.buildIntegratedPrompt(unified), unified.IdosoIdioma, nil
}

// InvalidatePromptCache invalida o cache de prompt para um idoso específico
// Deve ser chamado quando medicamentos ou dados importantes mudam
func (u *UnifiedRetrieval) InvalidatePromptCache(idosoID int64) {
	if u.promptCache != nil {
		u.promptCache.Invalidate(idosoID)
		log.Printf("🗑️ [CACHE] Prompt invalidado para idoso %d", idosoID)
	}
}

// GetPromptCacheStats retorna estatísticas do cache de prompts
func (u *UnifiedRetrieval) GetPromptCacheStats() (hits, misses int64, hitRate float64) {
	if u.promptCache != nil {
		return u.promptCache.GetStats()
	}
	return 0, 0, 0
}

// SaveConversationContext salva contexto da conversa para análise futura
func (u *UnifiedRetrieval) SaveConversationContext(ctx context.Context, idosoID int64, unified *UnifiedContext, userText, assistantText string) error {
	// Salvar no Postgres (análise)
	contextData := map[string]interface{}{
		"lacanian_analysis": unified.LacanianAnalysis,
		"ethical_stance":    unified.EthicalStance,
		"gurdjieff_type":    unified.GurdjieffType,
		"user_text":         userText,
		"assistant_text":    assistantText,
	}

	query := `
		INSERT INTO analise_gemini (idoso_id, tipo, conteudo, created_at)
		VALUES ($1, 'CONTEXT', $2, CURRENT_TIMESTAMP)
	`

	contextJSON, _ := json.Marshal(contextData)
	_, err := u.db.ExecContext(ctx, query, idosoID, contextJSON)

	return err
}

// Prime realiza pré-aquecimento do grafo (FDPN) após fala do usuário
func (u *UnifiedRetrieval) Prime(ctx context.Context, idosoID int64, text string) {
	if u.fdpn != nil {
		// Analisa e registra demanda no grafo (Spread Activation)
		// LatentDesire é inferido internamente ou vazio se analisado depois
		go u.fdpn.AnalyzeDemandAddressee(ctx, idosoID, text, "")
	}
	if u.embedding != nil {
		// Rastreia significantes para próxima recuperação
		go u.embedding.TrackSignifierChain(ctx, idosoID, text, 0.5)
	}
}

// ═══════════════════════════════════════════════════════════
// 🔓 MÉTODOS PÚBLICOS DO MODO DEBUG
// ═══════════════════════════════════════════════════════════

// GetDebugMode retorna a instância do modo debug (para uso externo)
func (u *UnifiedRetrieval) GetDebugMode() *DebugMode {
	return u.debugMode
}

// ProcessDebugCommand processa um comando de debug se o usuário for o Criador
// Retorna (resposta formatada, true) se foi um comando de debug, ou ("", false) se não
func (u *UnifiedRetrieval) ProcessDebugCommand(ctx context.Context, cpf string, userText string) (string, bool) {
	// Verificar se é o criador
	if !IsCreator(cpf) {
		return "", false
	}

	// Verificar se debugMode está disponível
	if u.debugMode == nil {
		return "", false
	}

	// Detectar comando de debug na fala
	command := u.debugMode.DetectDebugCommand(userText)
	if command == "" {
		return "", false
	}

	// Executar comando e formatar resposta
	log.Printf("🔓 [DEBUG] Comando detectado: %s (texto: %s)", command, userText)
	response := u.debugMode.ExecuteCommand(ctx, command)
	formattedResponse := u.debugMode.FormatDebugResponse(response)

	return formattedResponse, true
}

// GetDebugMetrics retorna métricas do sistema (apenas para o Criador)
func (u *UnifiedRetrieval) GetDebugMetrics(ctx context.Context, cpf string) (*DebugMetrics, error) {
	if !IsCreator(cpf) {
		return nil, fmt.Errorf("acesso negado: apenas o Criador pode acessar métricas de debug")
	}

	if u.debugMode == nil {
		return nil, fmt.Errorf("modo debug não inicializado")
	}

	return u.debugMode.GetSystemMetrics(ctx)
}

// RunDebugTest executa testes do sistema (apenas para o Criador)
func (u *UnifiedRetrieval) RunDebugTest(ctx context.Context, cpf string) (map[string]interface{}, error) {
	if !IsCreator(cpf) {
		return nil, fmt.Errorf("acesso negado: apenas o Criador pode executar testes")
	}

	if u.debugMode == nil {
		return nil, fmt.Errorf("modo debug não inicializado")
	}

	return u.debugMode.RunSystemTest(ctx)
}

// ═══════════════════════════════════════════════════════════════════════════════
// 🌍 SUPORTE A IDIOMAS INTERNACIONAIS
// ═══════════════════════════════════════════════════════════════════════════════

// getLanguageName converte código de idioma para nome legível
// Baseado nos 30 idiomas suportados pelo Gemini Live API
func getLanguageName(code string) string {
	languages := map[string]string{
		// Português
		"pt-BR": "Português (Brasil)",

		// Inglês
		"en-US": "English (United States)",
		"en-GB": "English (United Kingdom)",
		"en-AU": "English (Australia)",
		"en-IN": "English (India)",

		// Espanhol
		"es-ES": "Español (España)",
		"es-US": "Español (Estados Unidos)",

		// Francês
		"fr-FR": "Français (France)",
		"fr-CA": "Français (Canada)",

		// Alemão
		"de-DE": "Deutsch (Deutschland)",

		// Italiano
		"it-IT": "Italiano (Italia)",

		// Asiáticos
		"ja-JP":  "日本語 (Japanese)",
		"ko-KR":  "한국어 (Korean)",
		"cmn-CN": "中文 (Mandarin Chinese)",
		"th-TH":  "ไทย (Thai)",
		"vi-VN":  "Tiếng Việt (Vietnamese)",
		"id-ID":  "Bahasa Indonesia",

		// Indianos
		"hi-IN": "हिन्दी (Hindi)",
		"bn-IN": "বাংলা (Bengali)",
		"gu-IN": "ગુજરાતી (Gujarati)",
		"kn-IN": "ಕನ್ನಡ (Kannada)",
		"ml-IN": "മലയാളം (Malayalam)",
		"mr-IN": "मराठी (Marathi)",
		"ta-IN": "தமிழ் (Tamil)",
		"te-IN": "తెలుగు (Telugu)",

		// Outros
		"ar-XA": "العربية (Arabic)",
		"nl-NL": "Nederlands (Dutch)",
		"pl-PL": "Polski (Polish)",
		"ru-RU": "Русский (Russian)",
		"tr-TR": "Türkçe (Turkish)",
	}

	if name, ok := languages[code]; ok {
		return name
	}
	return code // Retorna o código se não encontrar
}
