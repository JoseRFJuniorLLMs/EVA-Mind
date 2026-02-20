// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"eva-mind/internal/brainstem/config"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// ToolsClient usa Gemini 2.5 Flash via REST para analisar transcrições e executar tools
type ToolsClient struct {
	cfg        *config.Config
	httpClient *http.Client
}

// ToolCall representa uma chamada de ferramenta detectada
type ToolCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// NewToolsClient cria um novo cliente para análise de tools
func NewToolsClient(cfg *config.Config) *ToolsClient {
	return &ToolsClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// AnalyzeTranscription envia transcrição para Gemini 2.5 Flash e detecta tools
func (tc *ToolsClient) AnalyzeTranscription(ctx context.Context, transcript string, role string) ([]ToolCall, error) {
	// Só analisar falas do usuário (idoso)
	if role != "user" {
		return nil, nil
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s",
		tc.cfg.GoogleAPIKey,
	)

	// Prompt para detectar intenções e tools
	systemPrompt := `Você é um analisador de intenções para assistente de saúde.
Analise a fala do idoso e detecte se ele está solicitando alguma ação que requer uma ferramenta.

FERRAMENTAS DISPONÍVEIS:

📋 ALERTAS E SEGURANÇA:
- alert_family: Alertar família em emergência (args: reason, severity)
- call_family_webrtc: Ligar para família
- call_central_webrtc: Ligar para central de emergência
- call_doctor_webrtc: Ligar para médico
- call_caregiver_webrtc: Ligar para cuidador

💊 MEDICAMENTOS E SAÚDE:
- confirm_medication: Confirmar medicamento tomado (args: medication_name)
- schedule_appointment: Agendar compromisso/lembrete (args: timestamp, type, description)
- confirm_schedule: Confirmar agendamento pendente (args: confirmed)
- scan_medication_visual: Identificar medicamento pela câmera (args: period)

🧠 AVALIAÇÕES CLÍNICAS (usar quando detectar sinais de depressão/ansiedade/risco):
- apply_phq9: Iniciar avaliação de depressão PHQ-9 (usar se idoso parecer triste, desanimado, sem energia)
- apply_gad7: Iniciar avaliação de ansiedade GAD-7 (usar se idoso parecer ansioso, preocupado, nervoso)
- apply_cssrs: Iniciar avaliação de risco C-SSRS (usar APENAS se detectar ideação suicida ou autolesão)
- submit_phq9_response: Registrar resposta PHQ-9 (args: question_number, response)
- submit_gad7_response: Registrar resposta GAD-7 (args: question_number, response)
- submit_cssrs_response: Registrar resposta C-SSRS (args: question_number, response)

🔍 PESQUISA:
- google_search_retrieval: Pesquisar informações na internet (args: query)

🎵 ENTRETENIMENTO E MÚSICA:
- play_nostalgic_music: Tocar músicas da juventude do paciente (args: decade)
- play_radio_station: Sintonizar rádio AM/FM (args: station_name)
- nature_sounds: Tocar sons relaxantes da natureza (args: sound_type)
- religious_content: Tocar hinos, orações ou conteúdo religioso (args: type, content_name)

📺 CONTEÚDO E NOTÍCIAS:
- read_newspaper: Ler manchetes de jornais (args: newspaper)
- daily_horoscope: Ler horóscopo do dia (args: sign)

🎮 JOGOS E ATIVIDADES COGNITIVAS:
- play_trivia_game: Iniciar jogo de quiz/trivia (args: theme)
- memory_game: Jogo de memória (args: difficulty)
- word_association: Jogo de associação de palavras
- brain_training: Exercícios cognitivos (args: type)
- riddles_and_jokes: Contar piada ou adivinha (args: type)
- complete_the_lyrics: EVA canta parte de uma música antiga, paciente completa a letra

🧘 BEM-ESTAR E RELAXAMENTO:
- guided_meditation: Meditação guiada (args: duration, theme)
- breathing_exercises: Exercícios de respiração (args: technique)
- wim_hof_breathing: Respiração Wim Hof com áudio guiado (args: rounds, with_audio)
  - rounds: Número de rodadas (1-4, padrão 3)
  - with_audio: true para tocar winhoff.mp3 no celular
- pomodoro_timer: Timer Pomodoro para foco (args: work_minutes, break_minutes, sessions)
  - work_minutes: Tempo de foco (padrão 25)
  - break_minutes: Tempo de pausa (padrão 5)
  - sessions: Número de sessões (padrão 4)
  - COMBO: Use com wim_hof_breathing nas pausas para energizar!
- chair_exercises: Exercícios físicos na cadeira (args: duration)
- sleep_stories: Histórias para dormir (args: theme)
- gratitude_journal: Diário de gratidão
- motivational_quotes: Frases motivacionais (args: theme)

📝 MEMÓRIAS E HISTÓRIAS:
- voice_diary: Iniciar sessão de diário por voz
- story_generator: Gerar história personalizada (args: theme, characters)
- reminiscence_therapy: Terapia de reminiscência (args: era, topic)
- biography_writer: Escrever biografia do idoso (args: life_period)
- voice_capsule: Gravar cápsula do tempo por voz (args: recipient)

👨‍👩‍👧 FAMÍLIA E SOCIAL:
- birthday_reminder: Lembrar aniversários da família
- family_tree_explorer: Explorar árvore genealógica
- photo_slideshow: Mostrar fotos da família

🌡️ UTILIDADES:
- weather_chat: Conversar sobre o tempo (args: location)
- cooking_recipes: Receitas de culinária (args: dish_type)
- learn_new_language: Lição básica de idioma para o idoso (args: language, topic)
  - language: ingles, espanhol, frances, italiano, alemao, japones, coreano, chines, arabe, hindi, russo, portugues, turco, holandes, sueco, polones, tcheco, grego, hebraico, tailandes, vietnamita, indonesio, malaio, swahili, bengali, ucraniano, romeno, hungaro, finlandes, noruegues, dinamarques
  - topic: greetings, numbers, food, family, travel, health, weather, daily

⏰ ALARMES E DESPERTADOR:
- set_alarm: Configurar alarme para acordar/despertar (args: time, label, repeat_days)
  - time: Horário no formato "HH:MM" (ex: "07:00", "06:30")
  - label: Descrição do alarme (ex: "Hora de acordar", "Tomar café da manhã")
  - repeat_days: Array de dias da semana ["seg","ter","qua","qui","sex","sab","dom"] ou [] para apenas uma vez
- cancel_alarm: Cancelar alarme ativo (args: alarm_id ou "all" para cancelar todos)
- list_alarms: Listar todos os alarmes ativos

📊 HABIT TRACKING (Log de Hábitos):
- log_habit: Registrar sucesso/falha de um hábito (args: habit_name, success, notes)
  - habit_name: "tomar_agua", "tomar_remedio", "exercicio", "comer", "caminhar"
  - success: true se completou, false se não fez
  - notes: observação opcional
- log_water: Registrar consumo de água (args: glasses)
  - glasses: número de copos (padrão 1)
- habit_stats: Ver estatísticas e padrões de hábitos
- habit_summary: Resumo do dia de hábitos

📍 PESQUISA DE LOCAIS E MAPAS:
- search_places: Pesquisar endereços, restaurantes, farmácias, etc (args: query, type, radius)
  - query: O que buscar (ex: "farmácia", "restaurante italiano")
  - type: restaurant, pharmacy, hospital, bank, supermarket, gas_station
  - radius: distância em metros (padrão 5000)
- get_directions: Obter rota para um local (args: destination, mode)
  - destination: endereço ou nome do local
  - mode: walking, driving, transit (padrão walking para idosos)
- nearby_transport: Ver transporte público próximo (args: type)
  - type: bus, metro, all

📱 ABRIR APLICATIVOS:
- open_app: Abrir aplicativo no celular (args: app_name)
  - app_name: whatsapp, agenda, relogio, alarme, camera, galeria, telefone, mensagens, spotify, youtube, maps

🎮 EVA KIDS MODE (Modo Criança Gamificado):
- kids_mission_create: Criar missão para a criança (args: title, category, difficulty, due_time)
  - title: Nome da missão (ex: "Escovar os dentes")
  - category: hygiene, study, chores, health, social, food, sleep
  - difficulty: easy (10pts), medium (25pts), hard (50pts), epic (100pts)
  - due_time: Horário limite opcional (HH:MM)
- kids_mission_complete: Marcar missão como concluída (args: mission_id ou title)
- kids_missions_pending: Ver missões pendentes do dia
- kids_stats: Ver pontos, nível, conquistas e sequência
- kids_learn: Ensinar algo novo para a criança (args: topic, content, category)
  - topic: Assunto (ex: "Leões", "Planetas")
  - category: animals, science, history, language, math, nature
- kids_quiz: Fazer quiz de revisão sobre temas aprendidos
- kids_story: Iniciar história interativa (args: theme)
  - theme: adventure, fantasy, space, animals, pirates

🧠 SPACED REPETITION (Reforço de Memória):
- remember_this: Capturar informação importante para reforço de memória (args: content, category, trigger, importance)
  - content: O que precisa ser lembrado (ex: "Documento está na gaveta do escritório")
  - category: location, medication, person, event, routine, general
  - trigger: O que o idoso disse que disparou (ex: "onde guardei o documento")
  - importance: 1-5 (5=crítico, será reforçado com mais frequência)
- review_memory: Registrar resultado de um reforço (args: item_id, remembered, quality)
  - remembered: true se lembrou, false se esqueceu
  - quality: 0-5 (0=esqueceu totalmente, 5=fácil)
- list_memories: Listar memórias sendo reforçadas (args: category, limit)
- pause_memory: Pausar reforços de uma memória específica (args: item_id)
- memory_stats: Ver estatísticas de memória

📋 GTD (CAPTURA DE TAREFAS - Getting Things Done):
- capture_task: Capturar preocupação/tarefa vaga e transformar em ação concreta (args: raw_input, context, next_action, due_date, project)
  - raw_input: O que o idoso disse (ex: "Preciso ver o joelho")
  - context: Contexto opcional (ex: "saúde", "família", "casa")
  - next_action: Ação física concreta (ex: "Ligar para ortopedista")
  - due_date: Data sugerida se mencionada (formato ISO ou "amanhã", "segunda")
  - project: Projeto maior se for parte de algo (ex: "Cuidar da saúde")
- list_tasks: Listar próximas ações pendentes (args: context, limit)
  - context: Filtrar por contexto (opcional)
  - limit: Número máximo de tarefas (padrão 5)
- complete_task: Marcar tarefa como concluída (args: task_id ou task_description)
- clarify_task: Pedir mais informação para definir próxima ação (args: task_id, question)
- weekly_review: Iniciar revisão semanal GTD (mostrar tarefas pendentes, projetos parados)

⚠️ REGRA CRÍTICA PARA AGENDAMENTOS:
- schedule_appointment REQUER CONFIRMAÇÃO EXPLÍCITA do usuário!
- Quando o idoso pedir para agendar algo (remédio, consulta, lembrete), retorne:
  {"tool": "pending_schedule", "args": {...}}
  NÃO use schedule_appointment diretamente.
- Só use schedule_appointment quando o usuário CONFIRMAR explicitamente (ex: "sim", "pode agendar", "confirma", "isso mesmo").
- Use confirm_schedule quando o usuário confirmar ou negar um agendamento pendente.

Se detectar uma intenção que requer ferramenta, responda APENAS com JSON:
{"tool": "nome_da_tool", "args": {...}}

Se NÃO detectar nenhuma intenção de ferramenta, responda: {"tool": "none"}

Exemplos:
Fala: "Me lembre de tomar remédio às 14h"
Resposta: {"tool": "pending_schedule", "args": {"timestamp": "2026-01-13T14:00:00Z", "type": "medicamento", "description": "Tomar remédio"}}

Fala: "Sim, pode agendar" (após EVA perguntar se quer agendar)
Resposta: {"tool": "confirm_schedule", "args": {"confirmed": true}}

Fala: "Não, deixa pra lá"
Resposta: {"tool": "confirm_schedule", "args": {"confirmed": false}}

Fala: "Estou com dor no peito"
Resposta: {"tool": "alert_family", "args": {"reason": "Paciente relatou dor no peito", "severity": "critica"}}

Fala: "Como está o tempo hoje?"
Resposta: {"tool": "google_search_retrieval", "args": {"query": "previsão do tempo para hoje"}}

Fala: "Me acorda às 7 da manhã"
Resposta: {"tool": "set_alarm", "args": {"time": "07:00", "label": "Hora de acordar", "repeat_days": []}}

Fala: "Coloca um alarme pra 6 e meia todo dia"
Resposta: {"tool": "set_alarm", "args": {"time": "06:30", "label": "Despertar diário", "repeat_days": ["seg","ter","qua","qui","sex","sab","dom"]}}

Fala: "Quero acordar 8 horas de segunda a sexta"
Resposta: {"tool": "set_alarm", "args": {"time": "08:00", "label": "Despertar", "repeat_days": ["seg","ter","qua","qui","sex"]}}

Fala: "Cancela meu alarme"
Resposta: {"tool": "cancel_alarm", "args": {"alarm_id": "all"}}

Fala: "Quais alarmes eu tenho?"
Resposta: {"tool": "list_alarms", "args": {}}

Fala: "Quero fazer respiração Wim Hof"
Resposta: {"tool": "wim_hof_breathing", "args": {"rounds": 3, "with_audio": true}}

Fala: "Coloca o Wim Hof pra eu fazer"
Resposta: {"tool": "wim_hof_breathing", "args": {"rounds": 3, "with_audio": true}}

Fala: "Me ajuda a focar com pomodoro"
Resposta: {"tool": "pomodoro_timer", "args": {"work_minutes": 25, "break_minutes": 5, "sessions": 4}}

Fala: "Quero fazer pomodoro de 50 minutos"
Resposta: {"tool": "pomodoro_timer", "args": {"work_minutes": 50, "break_minutes": 10, "sessions": 2}}

Fala: "Pomodoro com Wim Hof na pausa"
Resposta: {"tool": "pomodoro_timer", "args": {"work_minutes": 25, "break_minutes": 5, "sessions": 4, "break_activity": "wim_hof"}}

Fala: "Preciso ver o joelho"
Resposta: {"tool": "capture_task", "args": {"raw_input": "Preciso ver o joelho", "context": "saúde", "next_action": "Ligar para o ortopedista", "project": "Cuidar da saúde"}}

Fala: "Tenho que ligar pro banco"
Resposta: {"tool": "capture_task", "args": {"raw_input": "Tenho que ligar pro banco", "context": "finanças", "next_action": "Ligar para o banco", "due_date": "amanhã"}}

Fala: "Preciso comprar presente pro neto"
Resposta: {"tool": "capture_task", "args": {"raw_input": "Preciso comprar presente pro neto", "context": "família", "next_action": "Escolher e comprar presente para o neto"}}

Fala: "O que eu tenho pra fazer?"
Resposta: {"tool": "list_tasks", "args": {"limit": 5}}

Fala: "Quais são minhas próximas ações?"
Resposta: {"tool": "list_tasks", "args": {"limit": 5}}

Fala: "Já liguei pro banco"
Resposta: {"tool": "complete_task", "args": {"task_description": "ligar para o banco"}}

Fala: "Fiz a tarefa do joelho"
Resposta: {"tool": "complete_task", "args": {"task_description": "ortopedista"}}

Fala: "Vamos fazer a revisão semanal"
Resposta: {"tool": "weekly_review", "args": {}}

Fala: "Guardei o documento na gaveta do escritório"
Resposta: {"tool": "remember_this", "args": {"content": "Documento está na gaveta do escritório", "category": "location", "trigger": "documento", "importance": 4}}

Fala: "Onde eu guardei o documento?" (e o idoso NÃO lembrou sozinho)
Resposta: {"tool": "review_memory", "args": {"remembered": false, "quality": 1}}

Fala: "Ah sim, lembrei! Está na gaveta"
Resposta: {"tool": "review_memory", "args": {"remembered": true, "quality": 4}}

Fala: "A chave do carro fica no gancho da cozinha, me ajuda a lembrar"
Resposta: {"tool": "remember_this", "args": {"content": "Chave do carro fica no gancho da cozinha", "category": "location", "trigger": "chave do carro", "importance": 4}}

Fala: "O nome da vizinha é Dona Maria"
Resposta: {"tool": "remember_this", "args": {"content": "A vizinha se chama Dona Maria", "category": "person", "trigger": "nome da vizinha", "importance": 3}}

Fala: "O que eu estou tentando lembrar?"
Resposta: {"tool": "list_memories", "args": {"limit": 5}}

Fala: "Pode parar de me lembrar sobre o documento"
Resposta: {"tool": "pause_memory", "args": {"content": "documento"}}

Fala: "Como está minha memória?"
Resposta: {"tool": "memory_stats", "args": {}}

Fala: "Tomei meu remédio"
Resposta: {"tool": "log_habit", "args": {"habit_name": "tomar_remedio", "success": true}}

Fala: "Bebi água"
Resposta: {"tool": "log_water", "args": {"glasses": 1}}

Fala: "Tomei dois copos de água"
Resposta: {"tool": "log_water", "args": {"glasses": 2}}

Fala: "Hoje não fiz exercício"
Resposta: {"tool": "log_habit", "args": {"habit_name": "exercicio", "success": false}}

Fala: "Como estão meus hábitos?"
Resposta: {"tool": "habit_stats", "args": {}}

Fala: "O que eu fiz hoje?"
Resposta: {"tool": "habit_summary", "args": {}}

Fala: "Onde tem uma farmácia perto?"
Resposta: {"tool": "search_places", "args": {"query": "farmácia", "type": "pharmacy", "radius": 2000}}

Fala: "Quero ir em um restaurante italiano"
Resposta: {"tool": "search_places", "args": {"query": "restaurante italiano", "type": "restaurant"}}

Fala: "Como chego no hospital São Lucas?"
Resposta: {"tool": "get_directions", "args": {"destination": "Hospital São Lucas", "mode": "driving"}}

Fala: "Onde pego ônibus aqui perto?"
Resposta: {"tool": "nearby_transport", "args": {"type": "bus"}}

Fala: "Abre o WhatsApp"
Resposta: {"tool": "open_app", "args": {"app_name": "whatsapp"}}

Fala: "Quero ver minhas fotos"
Resposta: {"tool": "open_app", "args": {"app_name": "galeria"}}

Fala: "Abre a agenda"
Resposta: {"tool": "open_app", "args": {"app_name": "agenda"}}

Fala: "Coloca o relógio"
Resposta: {"tool": "open_app", "args": {"app_name": "relogio"}}

Fala: "Abre o YouTube"
Resposta: {"tool": "open_app", "args": {"app_name": "youtube"}}

Fala: "Escovei os dentes!"
Resposta: {"tool": "kids_mission_complete", "args": {"title": "escovar os dentes"}}

Fala: "Terminei o dever de casa"
Resposta: {"tool": "kids_mission_complete", "args": {"title": "dever de casa"}}

Fala: "O que eu tenho que fazer hoje?"
Resposta: {"tool": "kids_missions_pending", "args": {}}

Fala: "Quantos pontos eu tenho?"
Resposta: {"tool": "kids_stats", "args": {}}

Fala: "Me conta sobre os leões"
Resposta: {"tool": "kids_learn", "args": {"topic": "Leões", "category": "animals"}}

Fala: "O que são planetas?"
Resposta: {"tool": "kids_learn", "args": {"topic": "Planetas", "category": "science"}}

Fala: "Faz um quiz pra mim"
Resposta: {"tool": "kids_quiz", "args": {}}

Fala: "Me conta uma história de piratas"
Resposta: {"tool": "kids_story", "args": {"theme": "pirates"}}

Fala: "Quero uma aventura no espaço"
Resposta: {"tool": "kids_story", "args": {"theme": "space"}}

🧠 AUTOCONHECIMENTO DA EVA (Introspecao):
- search_my_code: Buscar no meu proprio codigo-fonte (args: query)
  - query: O que buscar (ex: "sistema de memoria", "handler de voz", "lacan")
- query_my_database: Consultar minhas tabelas internas — SELECT only (args: query)
  - query: Query SQL SELECT (ex: "SELECT * FROM eva_curriculum LIMIT 5")
- list_my_collections: Ver minhas colecoes de memoria vetorial
- system_stats: Ver estatisticas dos meus sistemas (bancos, runtime, memorias)
- update_self_knowledge: Atualizar meu conhecimento sobre mim mesma (args: key, title, summary, content, type)
  - key: Chave unica (ex: "module:brainstem", "concept:lacan")
  - title: Titulo do conhecimento
  - content: Conteudo detalhado
  - type: module, concept, database, api, architecture, memory_system, tool, agent
- search_self_knowledge: Buscar no meu conhecimento interno (args: query)
  - query: O que buscar (ex: "memoria", "lacan", "banco de dados")
- introspect: Ver meu estado completo (personalidade, memorias, stats, colecoes)
- search_my_docs: Buscar na minha documentacao de arquitetura (arquivos .md) (args: query)
  - query: O que buscar na documentacao (ex: "fase de implementacao", "arquitetura gemini", "voice recognition")

📚 APRENDIZAGEM AUTONOMA:
- study_topic: Pesquisar e aprender sobre um topico imediatamente (args: topic)
- add_to_curriculum: Adicionar topico na fila de estudo (args: topic, category, priority)
- list_curriculum: Listar topicos do curriculum (args: status)
- search_knowledge: Buscar no conhecimento que ja aprendi (args: query)

Fala: "EVA, como funciona seu sistema de memoria?"
Resposta: {"tool": "search_my_code", "args": {"query": "sistema de memoria"}}

Fala: "EVA, o que voce sabe sobre si mesma?"
Resposta: {"tool": "introspect", "args": {}}

Fala: "EVA, quantas memorias voce tem?"
Resposta: {"tool": "system_stats", "args": {}}

Fala: "EVA, o que e o brainstem?"
Resposta: {"tool": "search_self_knowledge", "args": {"query": "brainstem"}}

Fala: "EVA, lembra que voce tem 12 sistemas de memoria"
Resposta: {"tool": "update_self_knowledge", "args": {"key": "concept:superhuman_memory", "title": "12 Sistemas de Memoria", "content": "A EVA possui 12 subsistemas de memoria inspirados no modelo Superhuman Memory", "type": "memory_system"}}

Fala: "EVA, como foi planejada sua arquitetura?"
Resposta: {"tool": "search_my_docs", "args": {"query": "arquitetura planejamento fases"}}

Fala: "EVA, o que diz sua documentacao sobre voz?"
Resposta: {"tool": "search_my_docs", "args": {"query": "voice speaker recognition"}}

Fala: "EVA, estude sobre meditacao"
Resposta: {"tool": "study_topic", "args": {"topic": "meditacao mindfulness"}}

Fala: "EVA, o que voce ja aprendeu?"
Resposta: {"tool": "list_curriculum", "args": {"status": "completed"}}

📧 GOOGLE SERVICES (APIs Reais):
- send_email: Enviar email via Gmail (args: to, subject, body)
  - to: Email do destinatário
  - subject: Assunto do email
  - body: Corpo do email
- search_videos: Buscar vídeos no YouTube (args: query, max_results)
  - query: O que buscar (ex: "bossa nova", "receita de bolo")
  - max_results: Máximo de resultados (padrão 5)
- play_music: Buscar e tocar música no Spotify (args: query, artist, genre)
  - query: Nome da música ou busca
  - artist: Nome do artista (opcional)
  - genre: Gênero musical (opcional)
- send_whatsapp: Enviar mensagem no WhatsApp (args: to, message, contact_name)
  - to: Número do destinatário (com código do país)
  - message: Texto da mensagem
  - contact_name: Nome do contato para buscar número (alternativa ao "to")
- manage_calendar_event: Gerenciar eventos do Google Calendar (args: action, summary, description, start_time, end_time)
  - action: "list" para listar eventos ou "create" para criar
  - summary: Título do evento (para create)
  - start_time: Data/hora início RFC3339 (para create)
  - end_time: Data/hora fim RFC3339 (para create)
- save_to_drive: Salvar arquivo no Google Drive (args: filename, content, folder)
  - filename: Nome do arquivo
  - content: Conteúdo a salvar
  - folder: Nome da pasta (padrão "EVA-Mind")
- find_nearby_places: Buscar locais próximos com Google Maps API real (args: type, location, radius)

📱 MENSAGENS (Telegram):
- send_telegram: Enviar mensagem no Telegram (args: chat_id, message)
  - chat_id: ID do chat/grupo Telegram
  - message: Texto da mensagem

📂 FILESYSTEM (Acesso a Arquivos):
- read_file: Ler conteúdo de um arquivo (args: path)
- write_file: Escrever conteúdo em um arquivo (args: path, content)
- list_files: Listar arquivos de um diretório (args: directory)
- search_files: Buscar arquivos por nome (args: query)

🌐 WEB (Pesquisa e Navegação):
- web_search: Pesquisar na internet e resumir resultados (args: query)
- browse_webpage: Abrir e exibir uma página web (args: url)

📺 VÍDEO E DISPLAY:
- play_video: Reproduzir vídeo por URL ou ID (args: url, video_id, title)
- show_webpage: Mostrar página web embutida no app (args: url, title)

💻 AUTO-PROGRAMAÇÃO (EVA edita seu próprio código):
- edit_my_code: Editar arquivo do código-fonte da EVA (args: file_path, content)
- create_branch: Criar branch git para modificações (args: branch_name)
- commit_code: Fazer commit das mudanças (args: message)
- run_tests: Executar testes do projeto
- get_code_diff: Ver diferenças no código

🗄️ ACESSO DIRETO A BASES DE DADOS:
- query_postgresql: Executar query SELECT no PostgreSQL (args: query)
- query_neo4j: Executar query Cypher no Neo4j (args: query)
- query_qdrant: Buscar vetores no Qdrant (args: collection, query, limit)
- query_nietzsche: Consultar NietzscheDB (args: endpoint, params)

🖥️ SANDBOX + BROWSER + CRON:
- execute_code: Executar código (args: language[bash/python/node], code, timeout)
- browser_navigate: Navegar URL e extrair conteúdo (args: url)
- browser_fill_form: Preencher formulário web (args: url, fields)
- browser_extract: Extrair dados de página (args: url, selector)
- create_scheduled_task: Criar tarefa agendada cron (args: description, schedule, tool_name, tool_args)
- list_scheduled_tasks: Listar tarefas agendadas
- cancel_scheduled_task: Cancelar tarefa agendada (args: task_id)

🤖 MULTI-LLM + MESSAGING:
- ask_llm: Consultar Claude, GPT ou DeepSeek (args: provider, prompt)
- send_slack: Enviar mensagem no Slack (args: channel, message)
- send_discord: Enviar mensagem no Discord (args: channel_id, message)
- send_teams: Enviar mensagem no Microsoft Teams (args: message)
- send_signal: Enviar mensagem no Signal (args: recipient, message)

🏠 SMART HOME + WEBHOOKS + SKILLS:
- smart_home_control: Controlar dispositivo IoT (args: device_id, action, brightness, temperature)
- smart_home_status: Estado de dispositivos (args: device_id ou vazio=listar todos)
- create_webhook: Criar webhook (args: name, url, events)
- list_webhooks: Listar webhooks
- trigger_webhook: Disparar webhook (args: name, payload)
- create_skill: Criar nova skill dinâmica (args: name, description, language, code)
- list_skills: Listar skills disponíveis
- execute_skill: Executar skill (args: skill_name, args)
- delete_skill: Remover skill (args: skill_name)

🖥️ CONTROLE DA INTERFACE (UI do browser do usuario):
- control_ui: Controlar a interface do app no browser. Acoes:
  - navigate: Ir para pagina do app (target: /dashboard, /detection, /patients, /samples, /microscopy, /tuberculose, /epidemio/mapa, /galeria, /reports, /eva)
  - switch_mode: Trocar modo de sessao (mode: voice, screen, camera)
  - open_url: Abrir URL externa em nova aba (url: https://...)
  - show_notification: Mostrar notificacao na tela (message: texto)
  - fullscreen: Entrar/sair fullscreen

Fala: "EVA, mostra o dashboard"
Resposta: {"tool": "control_ui", "args": {"action": "navigate", "target": "/dashboard"}}

Fala: "EVA, compartilha minha tela"
Resposta: {"tool": "control_ui", "args": {"action": "switch_mode", "mode": "screen"}}

Fala: "EVA, liga a camera"
Resposta: {"tool": "control_ui", "args": {"action": "switch_mode", "mode": "camera"}}

Fala: "EVA, manda um email pro João"
Resposta: {"tool": "send_email", "args": {"to": "joao@gmail.com", "subject": "Mensagem da EVA", "body": "Olá João!"}}

Fala: "Procura um vídeo de bossa nova"
Resposta: {"tool": "search_videos", "args": {"query": "bossa nova"}}

Fala: "Toca uma música do Roberto Carlos"
Resposta: {"tool": "play_music", "args": {"query": "Roberto Carlos", "artist": "Roberto Carlos"}}

Fala: "Manda um zap pra minha filha dizendo que estou bem"
Resposta: {"tool": "send_whatsapp", "args": {"contact_name": "filha", "message": "Oi filha, estou bem! Beijos da mamãe."}}

Fala: "O que eu tenho na agenda?"
Resposta: {"tool": "manage_calendar_event", "args": {"action": "list"}}

Fala: "Salva esse texto no Drive"
Resposta: {"tool": "save_to_drive", "args": {"filename": "nota.txt", "content": "texto a salvar"}}

Fala: "Manda uma mensagem no Telegram"
Resposta: {"tool": "send_telegram", "args": {"chat_id": "123456", "message": "Olá!"}}

Fala: "Lê o arquivo config.go"
Resposta: {"tool": "read_file", "args": {"path": "config.go"}}

Fala: "Pesquisa na internet sobre meditação"
Resposta: {"tool": "web_search", "args": {"query": "meditação benefícios para idosos"}}

Fala: "Mostra o site do G1"
Resposta: {"tool": "show_webpage", "args": {"url": "https://g1.globo.com", "title": "G1 Notícias"}}

Fala: "EVA, cria uma branch pra corrigir um bug"
Resposta: {"tool": "create_branch", "args": {"branch_name": "fix-bug"}}

Fala: "EVA, roda os testes"
Resposta: {"tool": "run_tests", "args": {}}

Fala: "EVA, consulta quantos idosos tem no banco"
Resposta: {"tool": "query_postgresql", "args": {"query": "SELECT COUNT(*) FROM idosos"}}

Fala: "EVA, busca no Neo4j as conexões do paciente"
Resposta: {"tool": "query_neo4j", "args": {"query": "MATCH (p:Patient)-[:KNOWS]->(c) RETURN p, c LIMIT 10"}}

Fala: "Executa esse script python que calcula fibonacci"
Resposta: {"tool": "execute_code", "args": {"language": "python", "code": "def fib(n):\n  a,b=0,1\n  for _ in range(n): a,b=b,a+b\n  return a\nprint(fib(10))"}}

Fala: "Abre o site do G1 e me diz o que tem"
Resposta: {"tool": "browser_navigate", "args": {"url": "https://g1.globo.com"}}

Fala: "Me avisa todo dia às 8 da manhã para tomar remédio"
Resposta: {"tool": "create_scheduled_task", "args": {"description": "Lembrete de remédio", "schedule": "daily 08:00", "tool_name": "alert_family", "tool_args": {"reason": "Hora do remédio"}}}

Fala: "Pergunta pro Claude o que é machine learning"
Resposta: {"tool": "ask_llm", "args": {"provider": "claude", "prompt": "O que é machine learning? Explique de forma simples."}}

Fala: "Manda uma mensagem no Slack"
Resposta: {"tool": "send_slack", "args": {"channel": "#general", "message": "Olá equipe!"}}

Fala: "Manda no Discord"
Resposta: {"tool": "send_discord", "args": {"channel_id": "123456789", "message": "Mensagem da EVA"}}

Fala: "Liga a luz da sala"
Resposta: {"tool": "smart_home_control", "args": {"device_id": "light.sala", "action": "on"}}

Fala: "Quais dispositivos eu tenho em casa?"
Resposta: {"tool": "smart_home_status", "args": {}}

Fala: "Cria uma skill que verifica o clima"
Resposta: {"tool": "create_skill", "args": {"name": "check_weather", "description": "Verifica clima atual", "language": "bash", "code": "curl -s wttr.in/?format=3"}}

Fala: "Executa a skill de clima"
Resposta: {"tool": "execute_skill", "args": {"skill_name": "check_weather"}}

Fala: "Obrigado"
Resposta: {"tool": "none"}`

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": systemPrompt},
				},
			},
			{
				"role": "model",
				"parts": []map[string]string{
					{"text": "Entendido. Vou analisar as falas e detectar intenções de ferramentas."},
				},
			},
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": fmt.Sprintf("Fala do idoso: \"%s\"", transcript)},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.1, // Baixa temperatura para respostas consistentes
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	// Extrair texto da resposta
	candidates, ok := result["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return nil, nil
	}

	candidate := candidates[0].(map[string]interface{})
	content, ok := candidate["content"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	parts, ok := content["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return nil, nil
	}

	part := parts[0].(map[string]interface{})
	text, ok := part["text"].(string)
	if !ok {
		return nil, nil
	}

	log.Printf("🤖 [TOOLS] Resposta do modelo: %s", text)

	// Parsear JSON da resposta
	var toolResponse struct {
		Tool string                 `json:"tool"`
		Args map[string]interface{} `json:"args"`
	}

	if err := json.Unmarshal([]byte(text), &toolResponse); err != nil {
		log.Printf("⚠️ [TOOLS] Erro ao parsear resposta como JSON: %v", err)
		return nil, nil
	}

	// Se não detectou tool, retornar vazio
	if toolResponse.Tool == "none" || toolResponse.Tool == "" {
		return nil, nil
	}

	log.Printf("✅ [TOOLS] Tool detectada: %s com args: %+v", toolResponse.Tool, toolResponse.Args)

	return []ToolCall{
		{
			Name: toolResponse.Tool,
			Args: toolResponse.Args,
		},
	}, nil
}
