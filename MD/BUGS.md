# EVA-Mind — Auditoria Completa

**Data**: 2026-02-20
**Auditor**: Claude Opus 4.6

---

## 1. BUGS CONHECIDOS

### BUG 1: `tipo` — Código procura valor que não existe (ALTO)

```
CHECK constraint permite: 'lembrete_medicamento' (o que está no banco)
Código debug_mode.go:419:    WHERE tipo = 'medicamento'      <-- NUNCA encontra
Código unified_retrieval.go: WHERE tipo = 'medicamento'      <-- NUNCA encontra
```

**Impacto**: EVA nunca identifica medicamentos corretamente via unified_retrieval.

### BUG 2: `status` — Browser handler retorna ZERO linhas (ALTO)

```
Query browser_voice_handler.go:155: WHERE status IN ('agendado','ativo','pendente')
Banco: 0 registros com status 'agendado', 'ativo', ou 'pendente'
CHECK constraint NEM PERMITE os valores 'ativo' e 'pendente'
```

**Impacto**: EVA NUNCA vê medicamentos/agendamentos da pessoa no browser handler.

### BUG 3: `updated_at` vs `atualizado_em` (MÉDIO)

```
unified_retrieval.go: ORDER BY updated_at DESC
Banco: updated_at = NULL em 100% dos registros
Banco: atualizado_em = populado em 100%
```

**Impacto**: Ordenação quebrada — resultados em ordem indefinida.

### BUG 4: `dados_tarefa` formato incompatível (MÉDIO)

```
Banco usa: {medicamento, mensagem} ou {remedios, instrucoes}
UnifiedRetrieval espera: {nome, dosagem, forma, principio_ativo, horarios[], ...}
```

**Impacto**: Parsing resulta em struct com TODOS campos vazios.

### BUG 5: `escalation_policy` — Nunca lido (BAIXO)

```
Banco: 100% = 'alert_family' (com CHECK constraint)
Código: NENHUM handler lê esse campo
```

### BUG 6: `medicamento_confirmado` — Coluna não existe (ALTO)

```
actions.go:220-226: UPDATE agendamentos SET medicamento_confirmado = true
Banco: coluna real é 'medicamento_tomado'
```

**Impacto**: Confirmação de medicamento gera erro SQL silencioso.

### BUG 7: `gemini-3-flash` / `gemini-3-pro` em vm.env e cloud-run-env.yaml (MÉDIO)

IDs de modelo inválidos que causariam falha de API se carregados.

### BUG 8: Dados sujos em `condicoes_medicas` (MÉDIO)

Valores como "MORTO", "FEIO", "ALEJADO", "IA" injetados diretamente no prompt do Gemini.

---

## 2. PROBLEMAS DE SEGURANÇA

### CRÍTICO: Credenciais expostas no Git
- `.env` e `vm.env` commitados com TODAS as credenciais de produção
- Google API Key, DB password, JWT secret, Twilio tokens, OAuth secrets, Claude API key

### CRÍTICO: AuthMiddleware NUNCA aplicado
- Todas as rotas `/api/v1/*` (dados de pacientes, CPF) acessíveis sem autenticação

### ALTO: WebSocket upgraders aceitam qualquer origem
- `CheckOrigin: func(r *http.Request) bool { return true }` em todos os upgraders

### MÉDIO: Sandbox sem isolamento real
- Filtragem por string matching, sem Docker/seccomp/namespace

### MÉDIO: Self-Code permite EVA commitar no próprio repo

### BAIXO: Role `operator` não existe no `ValidRoles`

---

## 3. RACE CONDITIONS

### ALTO: VideoSessionManager
- `session.AttendantConn` acessado sob lock do manager mas sem lock por sessão

### MÉDIO: voice_change_helper.go
- `rows.Err()` nunca verificado após loop

---

## 4. O QUE ESTÁ FUNCIONANDO (22 funcionalidades)

1. Voz em tempo real (WebSocket + Gemini Live API)
2. Chat REST para Malária-Angola
3. Chat WebSocket (`/ws/eva`) com core memory
4. CRUD de Pacientes (PostgreSQL, 47 colunas)
5. JWT Auth (bcrypt, access 15min + refresh 7d)
6. Agendamento de medicamentos + push FCM
7. Escalação de emergência (push → email)
8. Neo4j grafo semântico de pacientes
9. Qdrant busca vetorial (1536D)
10. Swarm multi-agente (12 agentes, circuit breaker)
11. 150+ tools com function calling
12. Escalas clínicas PHQ-9, GAD-7, C-SSRS
13. Motor Lacaniano (FDPN, significantes, demanda/desejo)
14. Personalidade (Big Five + Enneagram)
15. Core Memory da EVA (identidade, reflexão)
16. Multi-LLM (Claude, GPT, DeepSeek)
17. WebRTC videochamada
18. Scheduler (lembretes em background)
19. Aprendizado autônomo (a cada 6h)
20. MCP Server (remember/recall)
21. Prometheus + Grafana
22. CI/CD GitHub Actions → GCP VM

---

## 5. PARCIALMENTE IMPLEMENTADO (corrigido em 2026-02-20)

### Ativados no main.go (eram código morto):
- Global Workspace (Consciência) — instanciado com LacanModule, PersonalityModule, EthicsModule
- Situational Modulator — instanciado e consolidado (duplicata removida)
- RAM Engine — instanciado com adapters para LLM, Embedding, Retrieval, Graph, Hebbian, Database

### Corrigidos:
- 10 TODOs de extração Neo4j em edge_zones.go, dual_weights.go, hebbian_realtime.go
- ExtractFacts() implementado com Gemini LLM real
- DetectContradictions() implementado com Gemini LLM real
- MCP recall com busca vetorial (Qdrant + fallback ILIKE)

### Implementados (backlog):
- Cliente gRPC do NietzscheDB — Go SDK pronto: `sdk-papa-caolho` (Nietzsche-Database/sdks/go/, 22 RPCs). Falta: reescrever `internal/brainstem/infrastructure/nietzsche/client.go` para usar o SDK
- Gerador de PDF real (gofpdf)
- Endpoints FHIR R4
- NER no detector de mentiras
- Aprendizado de CPTs na rede Bayesiana
- Sandbox com isolamento Docker
- Script de limpeza de dados sujos

---

## 6. CÓDIGO MORTO REMANESCENTE

| Módulo | Status |
|--------|--------|
| Memory Orchestrator | Pipeline parcial (Qdrant + PostgreSQL writes são stubs) |
| Memory Scheduler | Krylov maintenance é stub |

---

## 7. QUALIDADE DE CÓDIGO

- Dual JWT dependency (v4 e v5)
- Mixed logging (stdlib vs zerolog)
- Binary `eva-mcp-server.exe~` no Git
- Duplicate migration numbers (001, 002)
- EVA Workspace hardcoded para Linux paths
