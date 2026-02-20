// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

// tridb_benchmark.go - Benchmark comparativo: Neo4j vs Qdrant vs NietzscheDB
// Mede Recall@K, MRR, Precision@K e latência (P50/P95/P99)
// usando o mesmo dataset sintético de 20 memórias + 20 queries
package benchmark

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// BackendType identifica o backend sendo testado
type BackendType string

const (
	BackendNeo4j      BackendType = "neo4j"
	BackendQdrant     BackendType = "qdrant"
	BackendNietzsche  BackendType = "nietzschedb"
)

// BackendAdapter interface que todo backend de benchmark deve implementar
type BackendAdapter interface {
	Name() BackendType
	Setup(ctx context.Context, memories []SyntheticMemory) error
	Search(ctx context.Context, query string, k int) ([]RetrievedItem, error)
	Cleanup(ctx context.Context) error
}

// TriDBReport resultado comparativo dos 3 backends
type TriDBReport struct {
	Timestamp time.Time                       `json:"timestamp"`
	Dataset   DatasetInfo                     `json:"dataset"`
	Results   map[BackendType]*BackendResult  `json:"results"`
	Ranking   []RankedBackend                 `json:"ranking"`
	Summary   string                          `json:"summary"`
}

// DatasetInfo informações do dataset usado
type DatasetInfo struct {
	TotalMemories int `json:"total_memories"`
	TotalQueries  int `json:"total_queries"`
	QueryTypes    map[QueryType]int `json:"query_types"`
}

// BackendResult resultado de um backend
type BackendResult struct {
	Backend       BackendType                `json:"backend"`
	Available     bool                       `json:"available"`
	SetupTime     time.Duration              `json:"setup_time"`
	Metrics       GlobalMetrics              `json:"metrics"`
	ByType        map[QueryType]*Metrics     `json:"by_type"`
	Latency       LatencyStats               `json:"latency"`
	WriteLatency  LatencyStats               `json:"write_latency"`
	TotalQueries  int                        `json:"total_queries"`
	FailedQueries int                        `json:"failed_queries"`
	Queries       []QueryResult              `json:"queries,omitempty"`
	Error         string                     `json:"error,omitempty"`
}

// RankedBackend backend com score ponderado para ranking
type RankedBackend struct {
	Backend      BackendType `json:"backend"`
	CompositeScore float64  `json:"composite_score"`
	RecallAt10   float64    `json:"recall_at_10"`
	MRR          float64    `json:"mrr"`
	P50Latency   time.Duration `json:"p50_latency"`
}

// RunTriDBBenchmark executa o benchmark comparativo completo
func RunTriDBBenchmark(ctx context.Context, adapters []BackendAdapter) (*TriDBReport, error) {
	memories, queries := GenerateSyntheticDataset()

	// Info do dataset
	queryTypes := make(map[QueryType]int)
	for _, q := range queries {
		queryTypes[q.Type]++
	}

	report := &TriDBReport{
		Timestamp: time.Now(),
		Dataset: DatasetInfo{
			TotalMemories: len(memories),
			TotalQueries:  len(queries),
			QueryTypes:    queryTypes,
		},
		Results: make(map[BackendType]*BackendResult),
	}

	for _, adapter := range adapters {
		result := benchmarkSingleBackend(ctx, adapter, memories, queries)
		report.Results[adapter.Name()] = result
	}

	// Calcular ranking
	report.Ranking = calculateRanking(report.Results)
	report.Summary = generateSummary(report)

	return report, nil
}

// benchmarkSingleBackend executa benchmark para um único backend
func benchmarkSingleBackend(ctx context.Context, adapter BackendAdapter, memories []SyntheticMemory, queries []BenchmarkQuery) *BackendResult {
	result := &BackendResult{
		Backend:  adapter.Name(),
		ByType:   make(map[QueryType]*Metrics),
	}

	// 1. Setup: inserir memórias
	setupStart := time.Now()
	if err := adapter.Setup(ctx, memories); err != nil {
		result.Available = false
		result.Error = fmt.Sprintf("setup failed: %v", err)
		return result
	}
	result.SetupTime = time.Since(setupStart)
	result.Available = true

	// Medir write latency (inserções individuais) — já incluído no setup
	result.WriteLatency = LatencyStats{
		Avg: result.SetupTime / time.Duration(len(memories)),
		P50: result.SetupTime / time.Duration(len(memories)),
	}

	// 2. Executar queries
	var latencies []time.Duration
	typeResults := make(map[QueryType][]QueryResult)
	failCount := 0

	for _, q := range queries {
		start := time.Now()
		items, err := adapter.Search(ctx, q.Query, 20)
		latency := time.Since(start)
		latencies = append(latencies, latency)

		if err != nil {
			failCount++
			continue
		}

		retrievedIDs := make([]string, len(items))
		for i, item := range items {
			retrievedIDs[i] = item.ID
		}

		qr := QueryResult{
			QueryID:      q.ID,
			Query:        q.Query,
			Type:         q.Type,
			RecallAt5:    RecallAtK(q.ExpectedIDs, retrievedIDs, 5),
			RecallAt10:   RecallAtK(q.ExpectedIDs, retrievedIDs, 10),
			MRR:          MeanReciprocalRank(q.ExpectedIDs, retrievedIDs),
			Latency:      latency,
			RetrievedIDs: retrievedIDs,
			ExpectedIDs:  q.ExpectedIDs,
		}

		result.Queries = append(result.Queries, qr)
		typeResults[q.Type] = append(typeResults[q.Type], qr)
	}

	result.TotalQueries = len(queries)
	result.FailedQueries = failCount

	// 3. Agregar métricas
	result.Metrics = aggregateGlobalMetrics(result.Queries)
	result.Latency = computeLatencyStats(latencies)

	for qtype, qResults := range typeResults {
		m := &Metrics{Count: len(qResults)}
		for _, r := range qResults {
			m.RecallAt5 += r.RecallAt5
			m.RecallAt10 += r.RecallAt10
			m.Precision5 += PrecisionAtK(r.ExpectedIDs, r.RetrievedIDs, 5)
			m.MRR += r.MRR
		}
		if m.Count > 0 {
			m.RecallAt5 /= float64(m.Count)
			m.RecallAt10 /= float64(m.Count)
			m.Precision5 /= float64(m.Count)
			m.MRR /= float64(m.Count)
		}
		result.ByType[qtype] = m
	}

	// 4. Cleanup
	if err := adapter.Cleanup(ctx); err != nil {
		result.Error = fmt.Sprintf("cleanup warning: %v", err)
	}

	return result
}

// calculateRanking calcula ranking ponderado dos backends
// Score = 0.35*Recall@10 + 0.25*MRR + 0.20*(1-normalizedLatency) + 0.20*Recall@5
func calculateRanking(results map[BackendType]*BackendResult) []RankedBackend {
	var ranked []RankedBackend

	// Encontrar max latency para normalização
	var maxLatency time.Duration
	for _, r := range results {
		if r.Available && r.Latency.P50 > maxLatency {
			maxLatency = r.Latency.P50
		}
	}
	if maxLatency == 0 {
		maxLatency = 1 * time.Millisecond // evitar div/0
	}

	for backend, r := range results {
		if !r.Available {
			ranked = append(ranked, RankedBackend{
				Backend:        backend,
				CompositeScore: 0,
			})
			continue
		}

		// Normalizar latência: 0=lento, 1=rápido
		normalizedLatency := 1.0 - (float64(r.Latency.P50) / float64(maxLatency))
		normalizedLatency = math.Max(0, normalizedLatency)

		score := 0.35*r.Metrics.RecallAt10 +
			0.25*r.Metrics.MRR +
			0.20*normalizedLatency +
			0.20*r.Metrics.RecallAt5

		ranked = append(ranked, RankedBackend{
			Backend:        backend,
			CompositeScore: score,
			RecallAt10:     r.Metrics.RecallAt10,
			MRR:            r.Metrics.MRR,
			P50Latency:     r.Latency.P50,
		})
	}

	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].CompositeScore > ranked[j].CompositeScore
	})

	return ranked
}

// generateSummary gera resumo textual do benchmark
func generateSummary(report *TriDBReport) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("╔══════════════════════════════════════════════════════════════════╗\n"))
	sb.WriteString(fmt.Sprintf("║  EVA-Mind TriDB Benchmark — %s  ║\n", report.Timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("╠══════════════════════════════════════════════════════════════════╣\n"))
	sb.WriteString(fmt.Sprintf("║  Dataset: %d memórias, %d queries                               ║\n",
		report.Dataset.TotalMemories, report.Dataset.TotalQueries))
	sb.WriteString(fmt.Sprintf("╚══════════════════════════════════════════════════════════════════╝\n\n"))

	// Tabela comparativa
	sb.WriteString(fmt.Sprintf("┌─────────────┬───────────┬──────────┬──────────┬─────────┬──────────┬──────────┐\n"))
	sb.WriteString(fmt.Sprintf("│ Backend     │ Recall@5  │ Recall@10│ MRR      │ P50     │ P95      │ Score    │\n"))
	sb.WriteString(fmt.Sprintf("├─────────────┼───────────┼──────────┼──────────┼─────────┼──────────┼──────────┤\n"))

	for _, r := range report.Ranking {
		res := report.Results[r.Backend]
		if !res.Available {
			sb.WriteString(fmt.Sprintf("│ %-11s │    --     │    --    │    --    │   --    │    --    │  0.000   │\n", r.Backend))
			continue
		}

		medal := " "
		if r.CompositeScore == report.Ranking[0].CompositeScore {
			medal = "*"
		}

		sb.WriteString(fmt.Sprintf("│%s%-11s │  %5.1f%%   │  %5.1f%%  │  %.3f   │ %6s  │  %6s  │  %.3f   │\n",
			medal,
			r.Backend,
			res.Metrics.RecallAt5*100,
			res.Metrics.RecallAt10*100,
			res.Metrics.MRR,
			formatDuration(res.Latency.P50),
			formatDuration(res.Latency.P95),
			r.CompositeScore,
		))
	}

	sb.WriteString(fmt.Sprintf("└─────────────┴───────────┴──────────┴──────────┴─────────┴──────────┴──────────┘\n"))
	sb.WriteString(fmt.Sprintf("  * = vencedor\n\n"))

	// Detalhamento por tipo de query
	sb.WriteString("POR TIPO DE QUERY:\n")
	queryTypes := []QueryType{QueryTemporal, QueryEntity, QuerySemantic, QueryMixed}

	for _, qt := range queryTypes {
		sb.WriteString(fmt.Sprintf("\n  [%s]\n", qt))
		for _, r := range report.Ranking {
			res := report.Results[r.Backend]
			if !res.Available {
				continue
			}
			if m, ok := res.ByType[qt]; ok {
				sb.WriteString(fmt.Sprintf("    %-12s R@5=%5.1f%%  R@10=%5.1f%%  MRR=%.3f  (n=%d)\n",
					r.Backend, m.RecallAt5*100, m.RecallAt10*100, m.MRR, m.Count))
			}
		}
	}

	// Vencedor
	if len(report.Ranking) > 0 && report.Ranking[0].CompositeScore > 0 {
		winner := report.Ranking[0]
		sb.WriteString(fmt.Sprintf("\nVENCEDOR: %s (score=%.3f, R@10=%.1f%%, MRR=%.3f, P50=%s)\n",
			winner.Backend, winner.CompositeScore,
			winner.RecallAt10*100, winner.MRR,
			formatDuration(winner.P50Latency)))
	}

	return sb.String()
}

// formatDuration formata duração para display compacto
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.0fus", float64(d.Microseconds()))
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// FormatTriDBReport retorna relatório formatado para impressão
func FormatTriDBReport(r *TriDBReport) string {
	return r.Summary
}
