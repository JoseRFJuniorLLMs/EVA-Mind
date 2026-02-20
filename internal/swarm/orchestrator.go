// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package swarm

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Orchestrator roteia tool calls para o swarm agent correto
type Orchestrator struct {
	registry *Registry
	breaker  *CircuitBreaker
	deps     *Dependencies

	// Métricas globais
	totalCalls   atomic.Int64
	totalSuccess atomic.Int64
	totalFailed  atomic.Int64

	// Timeouts por prioridade
	timeouts map[Priority]time.Duration

	// Preempção
	mu          sync.RWMutex
	activeSwarm string
}

// NewOrchestrator cria um novo orchestrator
func NewOrchestrator(deps *Dependencies) *Orchestrator {
	return &Orchestrator{
		registry: NewRegistry(),
		breaker:  NewCircuitBreaker(10, 15*time.Second), // Mais tolerante: 10 falhas antes de abrir, recovery em 15s (web search pode ter falhas transientes)
		deps:     deps,
		timeouts: map[Priority]time.Duration{
			PriorityCritical: 2 * time.Second,
			PriorityHigh:     5 * time.Second,
			PriorityMedium:   15 * time.Second,
			PriorityLow:      60 * time.Second, // Scholar (web search) precisa de mais tempo — Gemini+Google Search grounding leva ~10-30s
		},
	}
}

// Register registra um swarm agent e inicializa com dependencies
func (o *Orchestrator) Register(agent SwarmAgent) error {
	if err := agent.Init(o.deps); err != nil {
		return fmt.Errorf("falha ao inicializar swarm '%s': %w", agent.Name(), err)
	}
	return o.registry.Register(agent)
}

// Route executa a tool call no swarm correto
func (o *Orchestrator) Route(ctx context.Context, call ToolCall) (*ToolResult, error) {
	start := time.Now()
	o.totalCalls.Add(1)

	// 1. Encontrar swarm responsável
	agent, err := o.registry.FindSwarm(call.Name)
	if err != nil {
		o.totalFailed.Add(1)
		return nil, fmt.Errorf("routing failed: %w", err)
	}

	swarmName := agent.Name()

	// 2. Verificar circuit breaker
	if o.breaker.IsOpen(swarmName) {
		o.totalFailed.Add(1)
		log.Printf("⚡ [SWARM] Circuit OPEN para %s - tool '%s' bloqueada", swarmName, call.Name)
		return &ToolResult{
			Success: false,
			Message: fmt.Sprintf("Serviço %s temporariamente indisponível", swarmName),
		}, nil
	}

	// 3. Definir timeout baseado na prioridade
	timeout, ok := o.timeouts[agent.Priority()]
	if !ok {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 4. Registrar swarm ativo (para preempção)
	o.mu.Lock()
	o.activeSwarm = swarmName
	o.mu.Unlock()

	// 5. Executar
	log.Printf("🔄 [SWARM] %s → %s.%s", call.Name, swarmName, call.Name)
	result, err := agent.Execute(ctx, call)

	latency := time.Since(start)

	// 6. Registrar resultado no circuit breaker
	if err != nil {
		o.breaker.RecordFailure(swarmName)
		o.totalFailed.Add(1)
		log.Printf("❌ [SWARM] %s.%s falhou em %v: %v", swarmName, call.Name, latency, err)
		return nil, err
	}

	o.breaker.RecordSuccess(swarmName)
	o.totalSuccess.Add(1)
	log.Printf("✅ [SWARM] %s.%s completou em %v", swarmName, call.Name, latency)

	// 7. Processar handoff se necessário
	if result != nil && result.Handoff != nil {
		log.Printf("🔀 [SWARM] Handoff: %s → %s (razão: %s)",
			swarmName, result.Handoff.TargetSwarm, result.Handoff.Reason)
		return o.executeHandoff(ctx, result.Handoff)
	}

	// 8. Processar side effects
	if result != nil && len(result.SideEffects) > 0 {
		go o.processSideEffects(result.SideEffects)
	}

	return result, nil
}

// executeHandoff transfere execução para outro swarm
func (o *Orchestrator) executeHandoff(ctx context.Context, handoff *HandoffRequest) (*ToolResult, error) {
	agent, ok := o.registry.GetSwarm(handoff.TargetSwarm)
	if !ok {
		return nil, fmt.Errorf("handoff target '%s' não encontrado", handoff.TargetSwarm)
	}

	// Injetar contexto do handoff
	call := handoff.ToolCall
	if call.Context == nil {
		call.Context = &ConversationContext{}
	}
	if call.Context.Metadata == nil {
		call.Context.Metadata = make(map[string]interface{})
	}
	call.Context.Metadata["handoff_from"] = handoff.Reason
	call.Context.Metadata["handoff_context"] = handoff.Context

	return agent.Execute(ctx, call)
}

// processSideEffects processa efeitos colaterais em background
func (o *Orchestrator) processSideEffects(effects []SideEffect) {
	for _, effect := range effects {
		switch effect.Type {
		case "log":
			log.Printf("📝 [SIDE_EFFECT] %v", effect.Payload)
		case "alert":
			log.Printf("🚨 [SIDE_EFFECT] Alert: %v", effect.Payload)
		case "metric":
			log.Printf("📊 [SIDE_EFFECT] Metric: %v", effect.Payload)
		case "notification":
			log.Printf("🔔 [SIDE_EFFECT] Notification: %v", effect.Payload)
		}
	}
}

// GetAllTools retorna todas as tools no formato Gemini
func (o *Orchestrator) GetAllTools() []interface{} {
	return o.registry.AllTools()
}

// Stats retorna estatísticas do orchestrator
func (o *Orchestrator) Stats() map[string]interface{} {
	swarms := o.registry.AllSwarms()
	swarmStats := make([]map[string]interface{}, 0, len(swarms))

	for _, s := range swarms {
		metrics := s.Metrics()
		swarmStats = append(swarmStats, map[string]interface{}{
			"name":         s.Name(),
			"priority":     s.Priority().String(),
			"health":       s.HealthCheck(),
			"tools":        len(s.Tools()),
			"total_calls":  metrics.TotalCalls,
			"success":      metrics.SuccessCalls,
			"failed":       metrics.FailedCalls,
			"avg_latency":  metrics.AvgLatency.String(),
			"circuit_open": o.breaker.IsOpen(s.Name()),
		})
	}

	return map[string]interface{}{
		"total_calls":   o.totalCalls.Load(),
		"total_success": o.totalSuccess.Load(),
		"total_failed":  o.totalFailed.Load(),
		"swarm_count":   o.registry.SwarmCount(),
		"tool_count":    o.registry.ToolCount(),
		"swarms":        swarmStats,
	}
}

// Shutdown desliga todos os swarms gracefully
func (o *Orchestrator) Shutdown() {
	log.Println("🛑 [SWARM] Shutting down all swarms...")
	for _, agent := range o.registry.AllSwarms() {
		if err := agent.Shutdown(); err != nil {
			log.Printf("⚠️ [SWARM] Erro ao desligar %s: %v", agent.Name(), err)
		}
	}
	log.Println("✅ [SWARM] All swarms shutdown complete")
}
