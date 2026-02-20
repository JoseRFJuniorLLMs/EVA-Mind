// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package benchmark

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// MOCK ADAPTERS (usados em testes unitários, sem dependência de infra)
// ============================================================================

// mockNeo4jAdapter simula Neo4j com busca por keywords em propriedades
type mockNeo4jAdapter struct {
	memories []SyntheticMemory
	latency  time.Duration
}

func (a *mockNeo4jAdapter) Name() BackendType { return BackendNeo4j }

func (a *mockNeo4jAdapter) Setup(ctx context.Context, memories []SyntheticMemory) error {
	a.memories = memories
	return nil
}

func (a *mockNeo4jAdapter) Search(ctx context.Context, query string, k int) ([]RetrievedItem, error) {
	time.Sleep(a.latency) // simular latência de rede

	queryLower := strings.ToLower(query)
	type scored struct {
		id    string
		score float64
	}
	var results []scored

	for _, m := range a.memories {
		score := 0.0
		contentLower := strings.ToLower(m.Content)

		// Neo4j: busca por propriedades (CONTAINS) — simula full-text index
		words := strings.Fields(queryLower)
		for _, word := range words {
			if len(word) < 3 {
				continue
			}
			if strings.Contains(contentLower, word) {
				score += 0.3
			}
		}

		// Boost por match de topics
		for _, topic := range m.Topics {
			if strings.Contains(queryLower, topic) {
				score += 0.4
			}
		}

		// Boost por match de emoção
		if m.Emotion != "" && strings.Contains(queryLower, m.Emotion) {
			score += 0.3
		}

		// Boost temporal (Neo4j é bom com grafos temporais)
		daysSince := time.Since(m.Timestamp).Hours() / 24
		if strings.Contains(queryLower, "ontem") || strings.Contains(queryLower, "yesterday") {
			if daysSince <= 2 {
				score += 0.5
			}
		}
		if strings.Contains(queryLower, "hoje") || strings.Contains(queryLower, "today") {
			if daysSince <= 1 {
				score += 0.5
			}
		}
		if strings.Contains(queryLower, "semana") || strings.Contains(queryLower, "week") {
			if daysSince <= 7 {
				score += 0.3
			}
		}

		if score > 0 {
			results = append(results, scored{id: m.ID, score: score})
		}
	}

	// Sort by score desc
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	var items []RetrievedItem
	for i, r := range results {
		if i >= k {
			break
		}
		items = append(items, RetrievedItem{ID: r.id, Similarity: r.score})
	}

	return items, nil
}

func (a *mockNeo4jAdapter) Cleanup(ctx context.Context) error { return nil }

// mockQdrantAdapter simula Qdrant com busca vetorial (cosine similarity)
type mockQdrantAdapter struct {
	memories   []SyntheticMemory
	embeddings map[string][]float32
	latency    time.Duration
}

func (a *mockQdrantAdapter) Name() BackendType { return BackendQdrant }

func (a *mockQdrantAdapter) Setup(ctx context.Context, memories []SyntheticMemory) error {
	a.memories = memories
	a.embeddings = make(map[string][]float32)

	// Gerar embeddings fake (bag-of-words normalizado)
	for _, m := range memories {
		a.embeddings[m.ID] = fakeEmbed(m.Content)
	}
	return nil
}

func (a *mockQdrantAdapter) Search(ctx context.Context, query string, k int) ([]RetrievedItem, error) {
	time.Sleep(a.latency) // simular latência gRPC

	queryEmbed := fakeEmbed(query)

	type scored struct {
		id    string
		score float64
	}
	var results []scored

	for _, m := range a.memories {
		memEmbed := a.embeddings[m.ID]
		sim := cosineSim(queryEmbed, memEmbed)
		if sim > 0.05 {
			results = append(results, scored{id: m.ID, score: sim})
		}
	}

	// Sort by score desc
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	var items []RetrievedItem
	for i, r := range results {
		if i >= k {
			break
		}
		items = append(items, RetrievedItem{ID: r.id, Similarity: r.score})
	}

	return items, nil
}

func (a *mockQdrantAdapter) Cleanup(ctx context.Context) error { return nil }

// mockNietzscheAdapter simula NietzscheDB com busca híbrida (keyword + decay temporal)
type mockNietzscheAdapter struct {
	memories []SyntheticMemory
	latency  time.Duration
}

func (a *mockNietzscheAdapter) Name() BackendType { return BackendNietzsche }

func (a *mockNietzscheAdapter) Setup(ctx context.Context, memories []SyntheticMemory) error {
	a.memories = memories
	return nil
}

func (a *mockNietzscheAdapter) Search(ctx context.Context, query string, k int) ([]RetrievedItem, error) {
	time.Sleep(a.latency) // simular latência HTTP

	queryLower := strings.ToLower(query)

	type scored struct {
		id    string
		score float64
	}
	var results []scored

	for _, m := range a.memories {
		score := 0.0
		contentLower := strings.ToLower(m.Content)

		// NietzscheDB: busca híbrida keyword + semântica + temporal decay
		words := strings.Fields(queryLower)
		for _, word := range words {
			if len(word) < 3 {
				continue
			}
			if strings.Contains(contentLower, word) {
				score += 0.25
			}
		}

		// Topic match (simulando embeddings do NietzscheDB)
		for _, topic := range m.Topics {
			if strings.Contains(queryLower, topic) {
				score += 0.35
			}
		}

		// NietzscheDB feature: biological temporal decay
		// Memórias recentes têm peso maior (consolidação por sono)
		daysSince := time.Since(m.Timestamp).Hours() / 24
		temporalBoost := math.Exp(-daysSince / 30.0) // decay exponencial 30 dias
		score *= (1.0 + temporalBoost*0.5)

		// Boost para queries temporais
		if strings.Contains(queryLower, "ontem") && daysSince <= 2 {
			score += 0.6
		}
		if strings.Contains(queryLower, "hoje") && daysSince <= 1 {
			score += 0.6
		}
		if strings.Contains(queryLower, "semana") && daysSince <= 7 {
			score += 0.4
		}

		// NietzscheDB: emotion-weighted recall
		if m.Emotion != "" {
			emotionWords := map[string][]string{
				"tristeza":    {"triste", "tristeza", "medo", "dá medo"},
				"alegria":     {"feliz", "alegria", "bonita", "festa"},
				"preocupação": {"preocup", "saúde", "remédio", "pressão"},
				"saudade":     {"saudade", "infância", "lembrança"},
				"ansiedade":   {"ansiedade", "medo", "hospital"},
			}
			if words, ok := emotionWords[m.Emotion]; ok {
				for _, w := range words {
					if strings.Contains(queryLower, w) {
						score += 0.3
						break
					}
				}
			}
		}

		if score > 0.1 {
			results = append(results, scored{id: m.ID, score: score})
		}
	}

	// Sort by score desc
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	var items []RetrievedItem
	for i, r := range results {
		if i >= k {
			break
		}
		items = append(items, RetrievedItem{ID: r.id, Similarity: r.score})
	}

	return items, nil
}

func (a *mockNietzscheAdapter) Cleanup(ctx context.Context) error { return nil }

// ============================================================================
// HELPER: Fake embeddings (bag-of-words com vocabulário fixo)
// ============================================================================

var vocabulary = []string{
	"remédio", "medicamento", "pressão", "médico", "consulta", "hospital",
	"família", "filho", "neta", "maria", "carlos", "joão",
	"saúde", "glicose", "sono", "exercício", "fisioterapia",
	"triste", "alegria", "saudade", "medo", "ansiedade", "preocupação",
	"ontem", "hoje", "semana", "manhã", "amanhã", "noite",
	"comida", "café", "almoço", "dieta", "sal",
	"visita", "festa", "aniversário", "bolo",
	"caminhar", "parque", "farmácia", "choveu",
	"marido", "faleceu", "luto", "infância", "avó",
	"esqueci", "lembrei", "tomou", "tomei",
}

func fakeEmbed(text string) []float32 {
	textLower := strings.ToLower(text)
	vec := make([]float32, len(vocabulary))

	for i, word := range vocabulary {
		if strings.Contains(textLower, word) {
			vec[i] = 1.0
		}
	}

	// Normalizar
	var norm float64
	for _, v := range vec {
		norm += float64(v * v)
	}
	if norm > 0 {
		normF := float32(math.Sqrt(norm))
		for i := range vec {
			vec[i] /= normF
		}
	}

	return vec
}

func cosineSim(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// ============================================================================
// TESTES
// ============================================================================

// TestTriDBBenchmark_Mock executa benchmark comparativo com mocks
func TestTriDBBenchmark_Mock(t *testing.T) {
	ctx := context.Background()

	adapters := []BackendAdapter{
		&mockNeo4jAdapter{latency: 2 * time.Millisecond},
		&mockQdrantAdapter{latency: 1 * time.Millisecond},
		&mockNietzscheAdapter{latency: 3 * time.Millisecond},
	}

	report, err := RunTriDBBenchmark(ctx, adapters)
	require.NoError(t, err)

	// Verificações básicas
	assert.Equal(t, 3, len(report.Results))
	assert.Equal(t, 3, len(report.Ranking))

	for _, r := range report.Results {
		assert.True(t, r.Available, "%s should be available", r.Backend)
		assert.Equal(t, 20, r.TotalQueries)
		assert.Equal(t, 0, r.FailedQueries)
		assert.Greater(t, r.Metrics.RecallAt5, 0.0, "%s should have some recall@5", r.Backend)
		assert.Greater(t, r.Metrics.MRR, 0.0, "%s should have some MRR", r.Backend)
	}

	// O ranking deve ter scores > 0
	for _, ranked := range report.Ranking {
		assert.Greater(t, ranked.CompositeScore, 0.0, "%s should have score > 0", ranked.Backend)
	}

	// Imprimir relatório
	fmt.Println(report.Summary)
}

// TestTriDBBenchmark_DetailedComparison verifica métricas detalhadas por tipo de query
func TestTriDBBenchmark_DetailedComparison(t *testing.T) {
	ctx := context.Background()

	adapters := []BackendAdapter{
		&mockNeo4jAdapter{latency: 2 * time.Millisecond},
		&mockQdrantAdapter{latency: 1 * time.Millisecond},
		&mockNietzscheAdapter{latency: 3 * time.Millisecond},
	}

	report, err := RunTriDBBenchmark(ctx, adapters)
	require.NoError(t, err)

	// Cada backend deve ter métricas por tipo
	for _, backend := range []BackendType{BackendNeo4j, BackendQdrant, BackendNietzsche} {
		r := report.Results[backend]
		assert.NotEmpty(t, r.ByType, "%s should have per-type metrics", backend)

		for _, qt := range []QueryType{QueryTemporal, QueryEntity, QuerySemantic, QueryMixed} {
			m, ok := r.ByType[qt]
			if ok {
				assert.Greater(t, m.Count, 0, "%s/%s should have queries", backend, qt)
				t.Logf("%s/%s: R@5=%.1f%% R@10=%.1f%% MRR=%.3f (n=%d)",
					backend, qt, m.RecallAt5*100, m.RecallAt10*100, m.MRR, m.Count)
			}
		}
	}
}

// TestTriDBBenchmark_LatencyComparison verifica que latências são medidas corretamente
func TestTriDBBenchmark_LatencyComparison(t *testing.T) {
	ctx := context.Background()

	// Usar latência pequena para garantir medição >0 mesmo no Windows
	// (timer resolution ~1ms no Windows, ~100ns no Linux)
	adapters := []BackendAdapter{
		&mockNeo4jAdapter{latency: 100 * time.Microsecond},
		&mockQdrantAdapter{latency: 100 * time.Microsecond},
		&mockNietzscheAdapter{latency: 100 * time.Microsecond},
	}

	report, err := RunTriDBBenchmark(ctx, adapters)
	require.NoError(t, err)

	// Verificar que todas as latências foram medidas corretamente
	for _, backend := range []BackendType{BackendNeo4j, BackendQdrant, BackendNietzsche} {
		r := report.Results[backend]
		// Com 100µs de sleep, P50 deve ser >= 100µs
		assert.GreaterOrEqual(t, r.Latency.P50, 100*time.Microsecond,
			"%s P50 should be >= 100µs", backend)
		assert.GreaterOrEqual(t, r.Latency.Avg, 100*time.Microsecond,
			"%s Avg should be >= 100µs", backend)
		// Ordenação P50 <= P95 <= P99
		assert.LessOrEqual(t, r.Latency.P50, r.Latency.P95, "%s P50 should be <= P95", backend)
		assert.LessOrEqual(t, r.Latency.P95, r.Latency.P99, "%s P95 should be <= P99", backend)

		t.Logf("%s: P50=%v, P95=%v, P99=%v, Avg=%v",
			backend, r.Latency.P50, r.Latency.P95, r.Latency.P99, r.Latency.Avg)
	}
}

// TestTriDBBenchmark_UnavailableBackend verifica comportamento quando backend falha
func TestTriDBBenchmark_UnavailableBackend(t *testing.T) {
	ctx := context.Background()

	adapters := []BackendAdapter{
		&mockNeo4jAdapter{latency: 1 * time.Millisecond},
		&failingAdapter{name: BackendQdrant},
		&mockNietzscheAdapter{latency: 1 * time.Millisecond},
	}

	report, err := RunTriDBBenchmark(ctx, adapters)
	require.NoError(t, err)

	assert.True(t, report.Results[BackendNeo4j].Available)
	assert.False(t, report.Results[BackendQdrant].Available)
	assert.True(t, report.Results[BackendNietzsche].Available)
	assert.Contains(t, report.Results[BackendQdrant].Error, "setup failed")

	// Ranking deve ter Qdrant com score 0
	for _, r := range report.Ranking {
		if r.Backend == BackendQdrant {
			assert.Equal(t, 0.0, r.CompositeScore)
		}
	}
}

// TestTriDBRanking verifica cálculo de ranking
func TestTriDBRanking(t *testing.T) {
	results := map[BackendType]*BackendResult{
		BackendNeo4j: {
			Available: true,
			Metrics:   GlobalMetrics{RecallAt5: 0.6, RecallAt10: 0.75, MRR: 0.5},
			Latency:   LatencyStats{P50: 5 * time.Millisecond},
		},
		BackendQdrant: {
			Available: true,
			Metrics:   GlobalMetrics{RecallAt5: 0.8, RecallAt10: 0.9, MRR: 0.7},
			Latency:   LatencyStats{P50: 2 * time.Millisecond},
		},
		BackendNietzsche: {
			Available: true,
			Metrics:   GlobalMetrics{RecallAt5: 0.85, RecallAt10: 0.92, MRR: 0.75},
			Latency:   LatencyStats{P50: 8 * time.Millisecond},
		},
	}

	ranking := calculateRanking(results)

	assert.Equal(t, 3, len(ranking))
	// O primeiro deve ter o maior score
	assert.GreaterOrEqual(t, ranking[0].CompositeScore, ranking[1].CompositeScore)
	assert.GreaterOrEqual(t, ranking[1].CompositeScore, ranking[2].CompositeScore)

	t.Logf("Ranking:")
	for i, r := range ranking {
		t.Logf("  #%d %s: score=%.3f R@10=%.1f%% MRR=%.3f P50=%v",
			i+1, r.Backend, r.CompositeScore, r.RecallAt10*100, r.MRR, r.P50Latency)
	}
}

// ============================================================================
// BENCHMARK INTEGRATION TEST (requer Neo4j, Qdrant, NietzscheDB rodando)
// ============================================================================

// TestTriDBBenchmark_Integration executa contra backends reais
// Pule com: go test -run TestTriDBBenchmark_Integration -tags integration
func TestTriDBBenchmark_Integration(t *testing.T) {
	// Só roda se variáveis de ambiente estiverem setadas
	neo4jURI := os.Getenv("NEO4J_URI")
	qdrantHost := os.Getenv("QDRANT_HOST")
	nietzscheURL := os.Getenv("NIETZSCHE_URL")

	if neo4jURI == "" && qdrantHost == "" && nietzscheURL == "" {
		t.Skip("Skipping integration test: set NEO4J_URI, QDRANT_HOST, NIETZSCHE_URL to run")
	}

	t.Logf("Integration test com: Neo4j=%s, Qdrant=%s, NietzscheDB=%s",
		neo4jURI, qdrantHost, nietzscheURL)

	// TODO: Instanciar adapters reais aqui quando infra estiver disponível
	// adapters := []BackendAdapter{}
	// if neo4jURI != "" { adapters = append(adapters, NewNeo4jBenchAdapter(neo4jURI, user, pass)) }
	// if qdrantHost != "" { adapters = append(adapters, NewQdrantBenchAdapter(qdrantHost, port)) }
	// if nietzscheURL != "" { adapters = append(adapters, NewNietzscheBenchAdapter(nietzscheURL)) }
	// report, err := RunTriDBBenchmark(ctx, adapters)

	t.Log("Integration test placeholder — instanciar adapters reais quando infra estiver up")
}

// ============================================================================
// GO BENCHMARKS (performance measurement)
// ============================================================================

func BenchmarkNeo4jSearch(b *testing.B) {
	adapter := &mockNeo4jAdapter{latency: 0}
	memories, queries := GenerateSyntheticDataset()
	ctx := context.Background()
	_ = adapter.Setup(ctx, memories)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := queries[i%len(queries)]
		adapter.Search(ctx, q.Query, 10)
	}
}

func BenchmarkQdrantSearch(b *testing.B) {
	adapter := &mockQdrantAdapter{latency: 0}
	memories, queries := GenerateSyntheticDataset()
	ctx := context.Background()
	_ = adapter.Setup(ctx, memories)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := queries[i%len(queries)]
		adapter.Search(ctx, q.Query, 10)
	}
}

func BenchmarkNietzscheSearch(b *testing.B) {
	adapter := &mockNietzscheAdapter{latency: 0}
	memories, queries := GenerateSyntheticDataset()
	ctx := context.Background()
	_ = adapter.Setup(ctx, memories)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := queries[i%len(queries)]
		adapter.Search(ctx, q.Query, 10)
	}
}

func BenchmarkTriDBFull(b *testing.B) {
	adapters := []BackendAdapter{
		&mockNeo4jAdapter{latency: 0},
		&mockQdrantAdapter{latency: 0},
		&mockNietzscheAdapter{latency: 0},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RunTriDBBenchmark(ctx, adapters)
	}
}

// ============================================================================
// HELPERS
// ============================================================================

type failingAdapter struct {
	name BackendType
}

func (a *failingAdapter) Name() BackendType { return a.name }
func (a *failingAdapter) Setup(ctx context.Context, memories []SyntheticMemory) error {
	return fmt.Errorf("connection refused")
}
func (a *failingAdapter) Search(ctx context.Context, query string, k int) ([]RetrievedItem, error) {
	return nil, fmt.Errorf("not available")
}
func (a *failingAdapter) Cleanup(ctx context.Context) error { return nil }
