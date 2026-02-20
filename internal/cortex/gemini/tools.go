// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package gemini

import (
	"database/sql"
	"eva-mind/internal/brainstem/push"
	"fmt"
	"log"
)

func GetDefaultTools() []interface{} {
	return []interface{}{
		map[string]interface{}{
			"function_declarations": []interface{}{
				map[string]interface{}{
					"name":        "alert_family",
					"description": "Alerta a família em caso de emergência detectada na conversa com o idoso",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"reason": map[string]interface{}{
								"type":        "string",
								"description": "Motivo do alerta (ex: 'Paciente relatou dor no peito', 'Idoso parece confuso')",
							},
							"severity": map[string]interface{}{
								"type":        "string",
								"description": "Severidade do alerta: critica, alta, media, baixa",
								"enum":        []string{"critica", "alta", "media", "baixa"},
							},
						},
						"required": []string{"reason"},
					},
				},
				map[string]interface{}{
					"name":        "confirm_medication",
					"description": "Confirma que o idoso tomou o remédio",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"medication_name": map[string]interface{}{
								"type":        "string",
								"description": "Nome do medicamento tomado",
							},
						},
						"required": []string{"medication_name"},
					},
				},
				map[string]interface{}{
					"name":        "schedule_appointment",
					"description": "Agenda um compromisso, consulta, medicamento ou chamada para o idoso",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"timestamp": map[string]interface{}{
								"type":        "string",
								"description": "Data e hora do agendamento no formato ISO 8601 (ex: 2024-02-25T14:30:00Z)",
							},
							"type": map[string]interface{}{
								"type":        "string",
								"description": "Tipo do agendamento",
								"enum":        []string{"consulta", "medicamento", "ligacao", "atividade", "outro"},
							},
							"description": map[string]interface{}{
								"type":        "string",
								"description": "Descrição detalhada do compromisso ou tarefa",
							},
						},
						"required": []string{"timestamp", "type", "description"},
					},
				},
				map[string]interface{}{
					"name":        "call_family_webrtc",
					"description": "Inicia uma chamada de vídeo para a família do idoso",
					"parameters": map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
					},
				},
				map[string]interface{}{
					"name":        "call_central_webrtc",
					"description": "Inicia uma chamada de vídeo de emergência para a Central EVA-Mind",
					"parameters": map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
					},
				},
				map[string]interface{}{
					"name":        "call_doctor_webrtc",
					"description": "Inicia uma chamada de vídeo para o médico responsável",
					"parameters": map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
					},
				},
				map[string]interface{}{
					"name":        "call_caregiver_webrtc",
					"description": "Inicia uma chamada de vídeo para o cuidador",
					"parameters": map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
					},
				},
				map[string]interface{}{
					"name":        "open_camera_analysis",
					"description": "Ativa a câmera do dispositivo do idoso para analisar visualmente um objeto, remédio ou ambiente",
					"parameters": map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
					},
				},
			},
		},
		map[string]interface{}{
			"function_declarations": []interface{}{
				map[string]interface{}{
					"name":        "manage_calendar_event",
					"description": "Gerencia eventos no Google Calendar (cria ou lista eventos)",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"action": map[string]interface{}{
								"type":        "string",
								"description": "Ação a realizar: 'create' ou 'list'",
								"enum":        []string{"create", "list"},
							},
							"summary": map[string]interface{}{
								"type":        "string",
								"description": "Título do evento (Obrigatório para 'create'). Ex: 'Consulta Cardiologista'",
							},
							"description": map[string]interface{}{
								"type":        "string",
								"description": "Descrição detalhada do evento",
							},
							"start_time": map[string]interface{}{
								"type":        "string",
								"description": "Horário de início (ISO 8601). Ex: '2024-12-25T14:00:00-03:00'",
							},
							"end_time": map[string]interface{}{
								"type":        "string",
								"description": "Horário de término (ISO 8601). Ex: '2024-12-25T15:00:00-03:00'",
							},
						},
						"required": []string{"action"},
					},
				},
			},
		},
		// --- CATEGORIA ENTRETENIMENTO ---
		map[string]interface{}{
			"function_declarations": []interface{}{
				map[string]interface{}{
					"name":        "play_nostalgic_music",
					"description": "Toca músicas da época de ouro (juventude) do paciente baseada no seu ano de nascimento",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"decade": map[string]interface{}{
								"type":        "string",
								"description": "Década preferida (opcional, ex: 'anos 60')",
							},
						},
					},
				},
				map[string]interface{}{
					"name":        "play_radio_station",
					"description": "Sintoniza estações de rádio AM/FM via streaming",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"station_name": map[string]interface{}{
								"type":        "string",
								"description": "Nome da rádio (ex: 'Antena 1', 'Rádio Nacional')",
							},
						},
						"required": []string{"station_name"},
					},
				},
				map[string]interface{}{
					"name":        "nature_sounds",
					"description": "Toca sons de natureza ou white noise para relaxamento e sono",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"sound_type": map[string]interface{}{
								"type":        "string",
								"description": "Tipo de som (chuva, mar, floresta, lareira)",
								"enum":        []string{"chuva", "mar", "floresta", "lareira", "sino_tibetano"},
							},
						},
						"required": []string{"sound_type"},
					},
				},
				map[string]interface{}{
					"name":        "religious_content",
					"description": "Reproduz hinos religiosos ou guia orações (terço, salmos)",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"type": map[string]interface{}{
								"type":        "string",
								"description": "Tipo: hino, oracao, terço, salmo",
							},
							"content_name": map[string]interface{}{
								"type":        "string",
								"description": "Nome específico (ex: 'Salmo 23', 'Ave Maria')",
							},
						},
						"required": []string{"type"},
					},
				},
				map[string]interface{}{
					"name":        "read_newspaper",
					"description": "Lê as manchetes do dia dos principais jornais",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"newspaper": map[string]interface{}{
								"type":        "string",
								"description": "Nome do jornal (ex: 'Folha', 'O Globo')",
							},
						},
					},
				},
				map[string]interface{}{
					"name":        "daily_horoscope",
					"description": "Busca e lê o horóscopo do dia para o signo do paciente",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"sign": map[string]interface{}{
								"type":        "string",
								"description": "Signo (ex: 'Capricórnio', 'Leão')",
							},
						},
					},
				},
				map[string]interface{}{
					"name":        "play_trivia_game",
					"description": "Inicia um jogo de perguntas e respostas (Quiz)",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"theme": map[string]interface{}{
								"type":        "string",
								"description": "Tema do quiz (ex: 'história', 'música antiga')",
							},
						},
					},
				},
				map[string]interface{}{
					"name":        "riddles_and_jokes",
					"description": "Conta piadas ou propõe adivinhas/charadas",
					"parameters": map[string]interface{}{
						"type": map[string]interface{}{
							"type":        "string",
							"description": "Se quer piada ou adivinha",
							"enum":        []string{"piada", "adivinha"},
						},
					},
				},
				map[string]interface{}{
					"name":        "voice_diary",
					"description": "Inicia uma sessão de diário por voz onde o paciente pode desabafar ou registrar o dia",
					"parameters": map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
					},
				},
				map[string]interface{}{
					"name":        "learn_new_language",
					"description": "Inicia uma lição básica de idioma (inglês, espanhol, francês) com frases do dia a dia",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"language": map[string]interface{}{
								"type":        "string",
								"description": "Idioma a aprender",
								"enum":        []string{"ingles", "espanhol", "frances", "italiano", "alemao", "japones", "coreano", "chines", "arabe", "hindi", "russo", "portugues", "turco", "holandes", "sueco", "polones", "tcheco", "grego", "hebraico", "tailandes", "vietnamita", "indonesio", "malaio", "swahili", "bengali", "ucraniano", "romeno", "hungaro", "finlandes", "noruegues", "dinamarques"},
							},
							"topic": map[string]interface{}{
								"type":        "string",
								"description": "Tema da lição (saudações, números, comida, família, viagem, saúde)",
							},
						},
						"required": []string{"language"},
					},
				},
				},
		},
		// 1. Gmail - send_email
		map[string]interface{}{
			"name":        "send_email",
			"description": "Envia um email usando Gmail do usuário",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"to": map[string]interface{}{
						"type":        "string",
						"description": "Email do destinatário",
					},
					"subject": map[string]interface{}{
						"type":        "string",
						"description": "Assunto do email",
					},
					"body": map[string]interface{}{
						"type":        "string",
						"description": "Corpo do email",
					},
				},
				"required": []string{"to", "subject", "body"},
			},
		},
		// 2. Drive - save_to_drive
		map[string]interface{}{
			"name":        "save_to_drive",
			"description": "Salva um documento no Google Drive do usuário",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"filename": map[string]interface{}{
						"type":        "string",
						"description": "Nome do arquivo",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Conteúdo do documento",
					},
					"folder": map[string]interface{}{
						"type":        "string",
						"description": "Nome da pasta (opcional)",
					},
				},
				"required": []string{"filename", "content"},
			},
		},
		// 3. Sheets - manage_health_sheet
		map[string]interface{}{
			"name":        "manage_health_sheet",
			"description": "Gerencia planilha de saúde no Google Sheets",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action": map[string]interface{}{
						"type":        "string",
						"description": "Ação: 'create' ou 'append'",
						"enum":        []string{"create", "append"},
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Título da planilha (para create)",
					},
					"data": map[string]interface{}{
						"type":        "object",
						"description": "Dados de saúde (date, time, blood_pressure, glucose, medication, notes)",
					},
				},
				"required": []string{"action"},
			},
		},
		// 4. Docs - create_health_doc
		map[string]interface{}{
			"name":        "create_health_doc",
			"description": "Cria um documento de saúde no Google Docs",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Título do documento",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Conteúdo do documento",
					},
				},
				"required": []string{"title", "content"},
			},
		},
		// 5. Maps - find_nearby_places
		map[string]interface{}{
			"name":        "find_nearby_places",
			"description": "Busca lugares próximos (farmácias, hospitais, restaurantes)",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"place_type": map[string]interface{}{
						"type":        "string",
						"description": "Tipo: pharmacy, hospital, restaurant, etc",
					},
					"location": map[string]interface{}{
						"type":        "string",
						"description": "Localização (lat,lng)",
					},
					"radius": map[string]interface{}{
						"type":        "integer",
						"description": "Raio em metros (padrão: 5000)",
					},
				},
				"required": []string{"place_type", "location"},
			},
		},
		// 6. YouTube - search_videos
		map[string]interface{}{
			"name":        "search_videos",
			"description": "Busca vídeos no YouTube",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Termo de busca",
					},
					"max_results": map[string]interface{}{
						"type":        "integer",
						"description": "Número máximo de resultados (padrão: 5)",
					},
				},
				"required": []string{"query"},
			},
		},
		// 7. Spotify - play_music
		map[string]interface{}{
			"name":        "play_music",
			"description": "Toca musica no spotify",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "musica ou artista",
					},
				},
				"required": []string{"query"},
			},
		},
		// 8. Uber - request_ride
		map[string]interface{}{
			"name":        "request_ride",
			"description": "Solicita corrida uber",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"startLat": map[string]interface{}{
						"type":        "number",
						"description": "startLat",
					},
					"startLng": map[string]interface{}{
						"type":        "number",
						"description": "startLng",
					},
					"endLat": map[string]interface{}{
						"type":        "number",
						"description": "endLat",
					},
					"endLng": map[string]interface{}{
						"type":        "number",
						"description": "endLng",
					},
				},
				"required": []string{"startLat", "startLng", "endLat", "endLng"},
			},
		},
		// 9. Google Fit - get_health_data
		map[string]interface{}{
			"name":        "get_health_data",
			"description": "Recupera dados de saúde do Google Fit",
			"parameters": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		// 10. WhatsApp - send_whatsapp
		map[string]interface{}{
			"name":        "send_whatsapp",
			"description": "Envia mensagem whatsapp",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"to": map[string]interface{}{
						"type":        "string",
						"description": "Numero destino",
					},
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Mensagem",
					},
				},
				"required": []string{"to", "message"},
			},
		},
		// ✅ Google Search Tool (Integrada ao modelo)
		map[string]interface{}{
			"google_search_retrieval": map[string]interface{}{
				"dynamic_retrieval_config": map[string]interface{}{
					"mode":              "MODE_DYNAMIC",
					"dynamic_threshold": 0.5, // Ajuste para equilibrar pesquisa e resposta direta
				},
			},
		},
		// ✅ SQL Select Tool (Database Access)
		map[string]interface{}{
			"function_declarations": []interface{}{
				map[string]interface{}{
					"name":        "run_sql_select",
					"description": "Executa uma consulta SQL SELECT (apenas leitura) no banco de dados para responder perguntas sobre o sistema. Use valid PostgreSQL syntax.",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{
								"type":        "string",
								"description": "A consulta SQL SELECT a ser executada. Ex: 'SELECT count(*) FROM idosos'",
							},
						},
						"required": []string{"query"},
					},
				},
				// ✅ Change Voice Tool (Runtime)
				map[string]interface{}{
					"name":        "change_voice",
					"description": "Altera a voz da assistente (EVA) em tempo real. Vozes disponiveis: Puck, Charon, Kore, Fenrir, Aoede",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"voice_name": map[string]interface{}{
								"type":        "string",
								"description": "Nome da voz desejada (Puck, Charon, Kore, Fenrir, Aoede)",
								"enum":        []string{"Puck", "Charon", "Kore", "Fenrir", "Aoede"},
							},
						},
						"required": []string{"voice_name"},
					},
				},
				// 🔧 ARCHITECT OVERRIDE: Change User Directive
				map[string]interface{}{
					"name":        "change_user_directive",
					"description": "🔧 Altera diretrizes do usuário em tempo real durante a conversa (idioma, modo legacy). Use quando o usuário debug solicitar mudanças de configuração.",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"directive_type": map[string]interface{}{
								"type":        "string",
								"enum":        []string{"language", "legacy_mode"},
								"description": "Tipo de diretiva: 'language' (idioma), 'legacy_mode' (modo legado)",
							},
							"new_value": map[string]interface{}{
								"type":        "string",
								"description": "Novo valor. Exemplos: 'en-US', 'pt-BR', 'true', 'false'",
							},
						},
						"required": []string{"directive_type", "new_value"},
					},
				},
			},
		},
		// ============================================================================
		// 🆕 AGENT CAPABILITIES — Filesystem, Telegram, Web, Self-Coding, Databases
		// ============================================================================
		map[string]interface{}{
			"function_declarations": []interface{}{
				// 📱 Telegram
				map[string]interface{}{
					"name":        "send_telegram",
					"description": "Enviar mensagem via Telegram Bot",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"chat_id": map[string]interface{}{"type": "string", "description": "ID do chat Telegram"},
							"message": map[string]interface{}{"type": "string", "description": "Texto da mensagem"},
						},
						"required": []string{"chat_id", "message"},
					},
				},
				// 📂 Filesystem
				map[string]interface{}{
					"name":        "read_file",
					"description": "Ler conteúdo de um arquivo local",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path": map[string]interface{}{"type": "string", "description": "Caminho do arquivo"},
						},
						"required": []string{"path"},
					},
				},
				map[string]interface{}{
					"name":        "write_file",
					"description": "Escrever conteúdo em arquivo local",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path":    map[string]interface{}{"type": "string", "description": "Caminho do arquivo"},
							"content": map[string]interface{}{"type": "string", "description": "Conteúdo a escrever"},
						},
						"required": []string{"path", "content"},
					},
				},
				map[string]interface{}{
					"name":        "list_files",
					"description": "Listar arquivos em um diretório",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"directory": map[string]interface{}{"type": "string", "description": "Caminho do diretório"},
						},
					},
				},
				map[string]interface{}{
					"name":        "search_files",
					"description": "Buscar arquivos por nome",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{"type": "string", "description": "Padrão de busca"},
						},
						"required": []string{"query"},
					},
				},
			},
		},
		map[string]interface{}{
			"function_declarations": []interface{}{
				// 🌐 Web
				map[string]interface{}{
					"name":        "web_search",
					"description": "Pesquisar na internet e resumir resultados",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{"type": "string", "description": "O que pesquisar"},
						},
						"required": []string{"query"},
					},
				},
				map[string]interface{}{
					"name":        "browse_webpage",
					"description": "Abrir e exibir uma página web",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"url": map[string]interface{}{"type": "string", "description": "URL da página"},
						},
						"required": []string{"url"},
					},
				},
				// 📺 Video & Display
				map[string]interface{}{
					"name":        "play_video",
					"description": "Reproduzir vídeo por URL ou ID do YouTube",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"url":      map[string]interface{}{"type": "string", "description": "URL do vídeo"},
							"video_id": map[string]interface{}{"type": "string", "description": "ID do vídeo no YouTube"},
							"title":    map[string]interface{}{"type": "string", "description": "Título do vídeo"},
						},
					},
				},
				map[string]interface{}{
					"name":        "show_webpage",
					"description": "Mostrar página web embutida no app",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"url":   map[string]interface{}{"type": "string", "description": "URL da página"},
							"title": map[string]interface{}{"type": "string", "description": "Título da página"},
						},
						"required": []string{"url"},
					},
				},
				// 💻 Self-Coding
				map[string]interface{}{
					"name":        "edit_my_code",
					"description": "Editar arquivo do código-fonte da EVA (apenas em branches eva/*)",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"file_path": map[string]interface{}{"type": "string", "description": "Caminho do arquivo"},
							"content":   map[string]interface{}{"type": "string", "description": "Novo conteúdo"},
						},
						"required": []string{"file_path", "content"},
					},
				},
			},
		},
		map[string]interface{}{
			"function_declarations": []interface{}{
				map[string]interface{}{
					"name":        "create_branch",
					"description": "Criar branch git para modificações (prefixo eva/)",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"branch_name": map[string]interface{}{"type": "string", "description": "Nome da branch"},
						},
					},
				},
				map[string]interface{}{
					"name":        "commit_code",
					"description": "Fazer commit das mudanças no código",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"message": map[string]interface{}{"type": "string", "description": "Mensagem do commit"},
						},
						"required": []string{"message"},
					},
				},
				map[string]interface{}{
					"name":        "run_tests",
					"description": "Executar testes do projeto (go test)",
					"parameters": map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
					},
				},
				map[string]interface{}{
					"name":        "get_code_diff",
					"description": "Ver diferenças no código (git diff)",
					"parameters": map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
					},
				},
				// 🗄️ Database Access
				map[string]interface{}{
					"name":        "query_postgresql",
					"description": "Executar query SELECT no PostgreSQL (apenas leitura)",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{"type": "string", "description": "Query SQL SELECT"},
						},
						"required": []string{"query"},
					},
				},
				map[string]interface{}{
					"name":        "query_neo4j",
					"description": "Executar query Cypher no Neo4j (apenas leitura)",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{"type": "string", "description": "Query Cypher MATCH/RETURN"},
						},
						"required": []string{"query"},
					},
				},
				map[string]interface{}{
					"name":        "query_qdrant",
					"description": "Buscar vetores no Qdrant",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"collection": map[string]interface{}{"type": "string", "description": "Nome da coleção"},
							"query":      map[string]interface{}{"type": "string", "description": "Texto para busca semântica"},
							"limit":      map[string]interface{}{"type": "integer", "description": "Máximo de resultados"},
						},
					},
				},
				map[string]interface{}{
					"name":        "query_nietzsche",
					"description": "Consultar NietzscheDB (API local)",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"endpoint": map[string]interface{}{"type": "string", "description": "Endpoint da API (ex: /api/quotes/random)"},
						},
					},
				},
			},
		},

		// ============================================================================
		// 🖥️ SANDBOX + BROWSER + CRON
		// ============================================================================
		map[string]interface{}{
			"function_declarations": []interface{}{
				map[string]interface{}{"name": "execute_code", "description": "Executar código em sandbox (bash, python ou node)", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"language": map[string]interface{}{"type": "string", "description": "bash, python ou node"}, "code": map[string]interface{}{"type": "string", "description": "Código a executar"}, "timeout": map[string]interface{}{"type": "number", "description": "Timeout em segundos"}}, "required": []string{"code"}}},
				map[string]interface{}{"name": "browser_navigate", "description": "Navegar para URL e extrair conteúdo da página", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"url": map[string]interface{}{"type": "string", "description": "URL da página"}}, "required": []string{"url"}}},
				map[string]interface{}{"name": "browser_fill_form", "description": "Preencher e submeter formulário web", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"url": map[string]interface{}{"type": "string", "description": "URL do formulário"}, "fields": map[string]interface{}{"type": "object", "description": "Campos do formulário"}}, "required": []string{"url", "fields"}}},
				map[string]interface{}{"name": "browser_extract", "description": "Extrair dados de uma página web", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"url": map[string]interface{}{"type": "string", "description": "URL"}, "selector": map[string]interface{}{"type": "string", "description": "text, links, title ou termo"}}, "required": []string{"url"}}},
				map[string]interface{}{"name": "create_scheduled_task", "description": "Criar tarefa agendada (cron)", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"description": map[string]interface{}{"type": "string", "description": "Descrição"}, "schedule": map[string]interface{}{"type": "string", "description": "every 5m, daily 08:00, hourly"}, "tool_name": map[string]interface{}{"type": "string", "description": "Tool a executar"}, "tool_args": map[string]interface{}{"type": "object", "description": "Args da tool"}}, "required": []string{"schedule"}}},
			},
		},
		// ============================================================================
		// 🤖 MULTI-LLM + 💬 MESSAGING (Slack, Discord)
		// ============================================================================
		map[string]interface{}{
			"function_declarations": []interface{}{
				map[string]interface{}{"name": "list_scheduled_tasks", "description": "Listar tarefas agendadas", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
				map[string]interface{}{"name": "cancel_scheduled_task", "description": "Cancelar tarefa agendada", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"task_id": map[string]interface{}{"type": "string", "description": "ID da tarefa"}}, "required": []string{"task_id"}}},
				map[string]interface{}{"name": "ask_llm", "description": "Consultar outro modelo de IA (Claude, GPT, DeepSeek)", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"provider": map[string]interface{}{"type": "string", "description": "claude, gpt, deepseek"}, "prompt": map[string]interface{}{"type": "string", "description": "Prompt"}}, "required": []string{"prompt"}}},
				map[string]interface{}{"name": "send_slack", "description": "Enviar mensagem via Slack", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"channel": map[string]interface{}{"type": "string", "description": "#general"}, "message": map[string]interface{}{"type": "string", "description": "Mensagem"}}, "required": []string{"message"}}},
				map[string]interface{}{"name": "send_discord", "description": "Enviar mensagem via Discord", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"channel_id": map[string]interface{}{"type": "string", "description": "ID do canal"}, "message": map[string]interface{}{"type": "string", "description": "Mensagem"}}, "required": []string{"channel_id", "message"}}},
			},
		},
		// ============================================================================
		// 💬 MESSAGING (Teams, Signal) + 🏠 SMART HOME + 🔗 WEBHOOKS
		// ============================================================================
		map[string]interface{}{
			"function_declarations": []interface{}{
				map[string]interface{}{"name": "send_teams", "description": "Enviar mensagem via Microsoft Teams", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"message": map[string]interface{}{"type": "string", "description": "Mensagem"}}, "required": []string{"message"}}},
				map[string]interface{}{"name": "send_signal", "description": "Enviar mensagem via Signal", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"recipient": map[string]interface{}{"type": "string", "description": "Número"}, "message": map[string]interface{}{"type": "string", "description": "Mensagem"}}, "required": []string{"recipient", "message"}}},
				map[string]interface{}{"name": "smart_home_control", "description": "Controlar dispositivo IoT/smart home", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"device_id": map[string]interface{}{"type": "string", "description": "light.sala"}, "action": map[string]interface{}{"type": "string", "description": "on, off, toggle"}, "brightness": map[string]interface{}{"type": "number", "description": "Brilho 0-255"}, "temperature": map[string]interface{}{"type": "number", "description": "Temperatura"}}, "required": []string{"device_id", "action"}}},
				map[string]interface{}{"name": "smart_home_status", "description": "Ver estado de dispositivos smart home", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"device_id": map[string]interface{}{"type": "string", "description": "ID (vazio=listar todos)"}}}},
				map[string]interface{}{"name": "create_webhook", "description": "Criar webhook para notificações", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string", "description": "Nome"}, "url": map[string]interface{}{"type": "string", "description": "URL destino"}, "events": map[string]interface{}{"type": "array", "description": "Eventos", "items": map[string]interface{}{"type": "string"}}}, "required": []string{"name", "url"}}},
			},
		},
		// ============================================================================
		// 🧩 SKILLS (Self-Improving Runtime)
		// ============================================================================
		map[string]interface{}{
			"function_declarations": []interface{}{
				map[string]interface{}{"name": "list_webhooks", "description": "Listar webhooks", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
				map[string]interface{}{"name": "trigger_webhook", "description": "Disparar webhook", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string", "description": "Nome"}, "payload": map[string]interface{}{"type": "object", "description": "Dados"}}, "required": []string{"name"}}},
				map[string]interface{}{"name": "create_skill", "description": "Criar nova skill/capacidade dinâmica", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string", "description": "Nome"}, "description": map[string]interface{}{"type": "string", "description": "Descrição"}, "language": map[string]interface{}{"type": "string", "description": "bash, python, node"}, "code": map[string]interface{}{"type": "string", "description": "Código"}}, "required": []string{"name", "code"}}},
				map[string]interface{}{"name": "list_skills", "description": "Listar skills disponíveis", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
				map[string]interface{}{"name": "execute_skill", "description": "Executar skill existente", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"skill_name": map[string]interface{}{"type": "string", "description": "Nome da skill"}, "args": map[string]interface{}{"type": "object", "description": "Argumentos"}}, "required": []string{"skill_name"}}},
			},
		},
		map[string]interface{}{
			"function_declarations": []interface{}{
				map[string]interface{}{"name": "delete_skill", "description": "Remover skill", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"skill_name": map[string]interface{}{"type": "string", "description": "Nome"}}, "required": []string{"skill_name"}}},
			},
		},
		// ============================================================================
		// 🖥️ UI CONTROL (controlar a interface do browser do usuario)
		// ============================================================================
		map[string]interface{}{
			"function_declarations": []interface{}{
				map[string]interface{}{
					"name":        "control_ui",
					"description": "Controlar a interface do aplicativo no browser do usuario. Pode navegar entre paginas, trocar modo de video/voz/camera, abrir URLs externas, mostrar notificacoes ou entrar em fullscreen.",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"action": map[string]interface{}{
								"type":        "string",
								"description": "Acao a executar na interface",
								"enum":        []string{"navigate", "switch_mode", "open_url", "show_notification", "fullscreen", "scroll_to"},
							},
							"target": map[string]interface{}{
								"type":        "string",
								"description": "Para navigate: rota do app (/dashboard, /detection, /patients, /samples, /microscopy, /tuberculose, /esquistossomose, /doenca-sono, /anemia-falciforme, /epidemio/mapa, /galeria, /reports, /clinics, /settings, /profile, /eva). Para scroll_to: ID do elemento.",
							},
							"mode": map[string]interface{}{
								"type":        "string",
								"description": "Para switch_mode: voice, screen ou camera",
								"enum":        []string{"voice", "screen", "camera"},
							},
							"url": map[string]interface{}{
								"type":        "string",
								"description": "Para open_url: URL externa a abrir em nova aba",
							},
							"message": map[string]interface{}{
								"type":        "string",
								"description": "Para show_notification: texto da notificacao",
							},
						},
						"required": []string{"action"},
					},
				},
			},
		},
	}
}

// CheckUnacknowledgedAlerts verifica alertas não visualizados e escalona se necessário
func CheckUnacknowledgedAlerts(db *sql.DB, pushService *push.FirebaseService) error {
	query := `
		SELECT 
			a.id,
			a.idoso_id,
			a.mensagem,
			a.severidade,
			i.nome,
			c.telefone
		FROM alertas a
		JOIN idosos i ON i.id = a.idoso_id
		LEFT JOIN cuidadores c ON c.idoso_id = i.id AND c.prioridade = 1
		WHERE a.visualizado = false
		  AND a.necessita_escalamento = true
		  AND a.tempo_escalamento <= NOW()
		  AND a.severidade IN ('critica', 'alta')
	`

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query unacknowledged alerts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var alertID, idosoID int64
		var message, severity, elderName string
		var phone sql.NullString

		if err := rows.Scan(&alertID, &idosoID, &message, &severity, &elderName, &phone); err != nil {
			log.Printf("❌ Error scanning alert: %v", err)
			continue
		}

		log.Printf("🚨 ESCALANDO alerta não visualizado - ID: %d, Idoso: %s", alertID, elderName)

		// TODO: Implementar ligação telefônica via Twilio para alertas críticos não visualizados
		// if phone.Valid && phone.String != "" {
		//     twilioService.MakeCall(phone.String, elderName, message)
		// }

		// Marcar que o escalonamento foi tentado
		_, _ = db.Exec(`
			UPDATE alertas 
			SET 
				tentativas_envio = tentativas_envio + 1,
				ultima_tentativa = NOW(),
				tempo_escalamento = NOW() + INTERVAL '10 minutes'
			WHERE id = $1
		`, alertID)
	}

	return nil
}
