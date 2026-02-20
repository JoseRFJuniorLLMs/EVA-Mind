// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package tools

import (
	"context"
	"fmt"
	"log"
	"time"

	"eva-mind/internal/motor/calendar"
	"eva-mind/internal/motor/drive"
	"eva-mind/internal/motor/gmail"
	"eva-mind/internal/motor/maps"
	"eva-mind/internal/motor/spotify"
	"eva-mind/internal/motor/whatsapp"
	"eva-mind/internal/motor/youtube"
)

// ============================================================================
// 📧 GMAIL — Enviar Email via Google API
// ============================================================================

func (h *ToolsHandler) handleSendEmail(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	to, _ := args["to"].(string)
	subject, _ := args["subject"].(string)
	body, _ := args["body"].(string)

	if to == "" {
		return map[string]interface{}{"error": "Informe o destinatário do email"}, nil
	}
	if subject == "" {
		subject = "Mensagem da EVA"
	}
	if body == "" {
		return map[string]interface{}{"error": "Informe o conteúdo do email"}, nil
	}

	accessToken, err := h.GetGoogleAccessToken(idosoID)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Não foi possível acessar o Gmail: %v", err)}, nil
	}

	// Non-blocking: envia em goroutine
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		svc := gmail.NewService(ctx)
		sendErr := svc.SendEmail(accessToken, to, subject, body)

		if h.NotifyFunc != nil {
			if sendErr != nil {
				log.Printf("❌ [GMAIL] Erro ao enviar email: %v", sendErr)
				h.NotifyFunc(idosoID, "email_error", map[string]interface{}{
					"to":    to,
					"error": sendErr.Error(),
				})
			} else {
				log.Printf("✅ [GMAIL] Email enviado para %s", to)
				h.NotifyFunc(idosoID, "email_sent", map[string]interface{}{
					"to":      to,
					"subject": subject,
				})
			}
		}
	}()

	return map[string]interface{}{
		"status":  "enviando",
		"to":      to,
		"subject": subject,
		"message": fmt.Sprintf("Enviando email para %s...", to),
	}, nil
}

// ============================================================================
// 🎥 YOUTUBE — Buscar e Reproduzir Vídeos
// ============================================================================

func (h *ToolsHandler) handleSearchVideos(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return map[string]interface{}{"error": "Informe o que deseja assistir"}, nil
	}

	maxResultsFloat, _ := args["max_results"].(float64)
	maxResults := int64(maxResultsFloat)
	if maxResults <= 0 {
		maxResults = 5
	}

	accessToken, err := h.GetGoogleAccessToken(idosoID)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Não foi possível acessar o YouTube: %v", err)}, nil
	}

	// Non-blocking: busca em goroutine
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		svc := youtube.NewService(ctx)
		videos, searchErr := svc.SearchVideos(accessToken, query, maxResults)

		if h.NotifyFunc != nil {
			if searchErr != nil {
				log.Printf("❌ [YOUTUBE] Erro na busca: %v", searchErr)
				h.NotifyFunc(idosoID, "video_error", map[string]interface{}{
					"query": query,
					"error": searchErr.Error(),
				})
			} else if len(videos) > 0 {
				log.Printf("✅ [YOUTUBE] Encontrados %d vídeos para '%s'", len(videos), query)
				// Envia o primeiro vídeo para reprodução + lista completa
				h.NotifyFunc(idosoID, "play_video", map[string]interface{}{
					"video_id": videos[0]["video_id"],
					"url":      videos[0]["url"],
					"title":    videos[0]["title"],
					"results":  videos,
				})
			}
		}
	}()

	return map[string]interface{}{
		"status":  "buscando",
		"query":   query,
		"message": fmt.Sprintf("Procurando vídeos de '%s'...", query),
	}, nil
}

// ============================================================================
// 🎵 SPOTIFY — Buscar e Tocar Música
// ============================================================================

func (h *ToolsHandler) handlePlayMusic(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	query, _ := args["query"].(string)
	artist, _ := args["artist"].(string)
	genre, _ := args["genre"].(string)

	// Construir query de busca
	searchQuery := query
	if searchQuery == "" && artist != "" {
		searchQuery = artist
	}
	if searchQuery == "" && genre != "" {
		searchQuery = genre
	}
	if searchQuery == "" {
		return map[string]interface{}{"error": "Informe a música, artista ou gênero"}, nil
	}

	// Spotify usa OAuth próprio — por enquanto notifica o app para abrir Spotify
	if h.NotifyFunc != nil {
		h.NotifyFunc(idosoID, "play_music", map[string]interface{}{
			"query":  searchQuery,
			"artist": artist,
			"genre":  genre,
		})
	}

	// Tentar buscar via API se tiver token
	go func() {
		// Buscar token Spotify do usuário (se existir)
		var spotifyToken string
		h.db.Conn.QueryRow("SELECT spotify_access_token FROM idosos WHERE id = $1", idosoID).Scan(&spotifyToken)

		if spotifyToken != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			svc := spotify.NewService(ctx)
			tracks, err := svc.SearchTracks(spotifyToken, searchQuery, 5)
			if err == nil && len(tracks) > 0 {
				log.Printf("✅ [SPOTIFY] Encontradas %d músicas para '%s'", len(tracks), searchQuery)
				if h.NotifyFunc != nil {
					h.NotifyFunc(idosoID, "play_spotify", map[string]interface{}{
						"track_name":   tracks[0]["name"],
						"track_artist": tracks[0]["artist"],
						"track_uri":    tracks[0]["uri"],
						"results":      tracks,
					})
				}

				// Tentar reproduzir no dispositivo ativo
				_ = svc.PlayTrack(spotifyToken, tracks[0]["uri"])
			}
		}
	}()

	return map[string]interface{}{
		"status":  "buscando",
		"query":   searchQuery,
		"message": fmt.Sprintf("Procurando '%s'...", searchQuery),
	}, nil
}

// ============================================================================
// 💬 WHATSAPP — Enviar Mensagem via Meta API
// ============================================================================

func (h *ToolsHandler) handleSendWhatsApp(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	to, _ := args["to"].(string)
	message, _ := args["message"].(string)
	contactName, _ := args["contact_name"].(string)

	if to == "" && contactName != "" {
		// Buscar número pelo nome do contato
		var phone string
		err := h.db.Conn.QueryRow(
			"SELECT telefone FROM cuidadores c JOIN cuidador_idoso ci ON c.id = ci.cuidador_id WHERE ci.idoso_id = $1 AND LOWER(c.nome) LIKE LOWER($2) LIMIT 1",
			idosoID, "%"+contactName+"%",
		).Scan(&phone)
		if err == nil && phone != "" {
			to = phone
		}
	}

	if to == "" {
		return map[string]interface{}{"error": "Informe o número ou nome do contato"}, nil
	}
	if message == "" {
		return map[string]interface{}{"error": "Informe a mensagem a enviar"}, nil
	}

	if h.whatsappToken == "" || h.whatsappPhoneID == "" {
		return map[string]interface{}{"error": "WhatsApp não configurado — peça ao administrador"}, nil
	}

	// Non-blocking
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		svc := whatsapp.NewService(ctx, h.whatsappToken, h.whatsappPhoneID)
		sendErr := svc.SendMessage(to, message)

		if h.NotifyFunc != nil {
			if sendErr != nil {
				log.Printf("❌ [WHATSAPP] Erro: %v", sendErr)
				h.NotifyFunc(idosoID, "whatsapp_error", map[string]interface{}{
					"to":    to,
					"error": sendErr.Error(),
				})
			} else {
				log.Printf("✅ [WHATSAPP] Mensagem enviada para %s", to)
				h.NotifyFunc(idosoID, "whatsapp_sent", map[string]interface{}{
					"to":      to,
					"message": message,
				})
			}
		}
	}()

	return map[string]interface{}{
		"status":  "enviando",
		"to":      to,
		"message": fmt.Sprintf("Enviando mensagem no WhatsApp para %s...", to),
	}, nil
}

// ============================================================================
// 📅 GOOGLE CALENDAR — Gerenciar Eventos
// ============================================================================

func (h *ToolsHandler) handleManageCalendar(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	action, _ := args["action"].(string)

	accessToken, err := h.GetGoogleAccessToken(idosoID)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Não foi possível acessar o Calendar: %v", err)}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	svc := calendar.NewService(ctx)

	switch action {
	case "list", "listar", "":
		events, listErr := svc.ListUpcomingEventsForUser(accessToken)
		if listErr != nil {
			return map[string]interface{}{"error": fmt.Sprintf("Erro ao buscar eventos: %v", listErr)}, nil
		}

		var eventList []map[string]string
		for _, e := range events {
			start := e.Start.DateTime
			if start == "" {
				start = e.Start.Date
			}
			eventList = append(eventList, map[string]string{
				"title": e.Summary,
				"start": start,
				"link":  e.HtmlLink,
			})
		}

		if len(eventList) == 0 {
			return map[string]interface{}{
				"status":  "sucesso",
				"events":  eventList,
				"message": "Você não tem eventos próximos na agenda.",
			}, nil
		}

		return map[string]interface{}{
			"status":  "sucesso",
			"events":  eventList,
			"count":   len(eventList),
			"message": fmt.Sprintf("Encontrei %d eventos na sua agenda.", len(eventList)),
		}, nil

	case "create", "criar":
		summary, _ := args["summary"].(string)
		description, _ := args["description"].(string)
		startTime, _ := args["start_time"].(string)
		endTime, _ := args["end_time"].(string)

		if summary == "" {
			return map[string]interface{}{"error": "Informe o título do evento"}, nil
		}
		if startTime == "" {
			return map[string]interface{}{"error": "Informe a data/hora de início"}, nil
		}
		if endTime == "" {
			// Default: 1 hora depois
			if t, parseErr := time.Parse(time.RFC3339, startTime); parseErr == nil {
				endTime = t.Add(1 * time.Hour).Format(time.RFC3339)
			}
		}

		link, createErr := svc.CreateEventForUser(accessToken, summary, description, startTime, endTime)
		if createErr != nil {
			return map[string]interface{}{"error": fmt.Sprintf("Erro ao criar evento: %v", createErr)}, nil
		}

		if h.NotifyFunc != nil {
			h.NotifyFunc(idosoID, "calendar_event_created", map[string]interface{}{
				"title": summary,
				"start": startTime,
				"link":  link,
			})
		}

		return map[string]interface{}{
			"status":  "sucesso",
			"title":   summary,
			"link":    link,
			"message": fmt.Sprintf("Evento '%s' criado na sua agenda!", summary),
		}, nil

	default:
		return map[string]interface{}{"error": fmt.Sprintf("Ação '%s' não reconhecida. Use 'list' ou 'create'.", action)}, nil
	}
}

// ============================================================================
// 📁 GOOGLE DRIVE — Salvar Arquivos
// ============================================================================

func (h *ToolsHandler) handleSaveToDrive(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	filename, _ := args["filename"].(string)
	content, _ := args["content"].(string)
	folder, _ := args["folder"].(string)

	if filename == "" {
		filename = fmt.Sprintf("EVA_nota_%s.txt", time.Now().Format("2006-01-02_15-04"))
	}
	if content == "" {
		return map[string]interface{}{"error": "Informe o conteúdo a salvar"}, nil
	}
	if folder == "" {
		folder = "EVA-Mind"
	}

	accessToken, err := h.GetGoogleAccessToken(idosoID)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Não foi possível acessar o Drive: %v", err)}, nil
	}

	// Non-blocking
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		svc := drive.NewService(ctx)
		fileID, saveErr := svc.SaveFile(accessToken, filename, content, folder)

		if h.NotifyFunc != nil {
			if saveErr != nil {
				log.Printf("❌ [DRIVE] Erro: %v", saveErr)
				h.NotifyFunc(idosoID, "drive_error", map[string]interface{}{
					"filename": filename,
					"error":    saveErr.Error(),
				})
			} else {
				log.Printf("✅ [DRIVE] Arquivo salvo: %s (ID: %s)", filename, fileID)
				h.NotifyFunc(idosoID, "drive_saved", map[string]interface{}{
					"filename": filename,
					"file_id":  fileID,
					"folder":   folder,
				})
			}
		}
	}()

	return map[string]interface{}{
		"status":   "salvando",
		"filename": filename,
		"folder":   folder,
		"message":  fmt.Sprintf("Salvando '%s' no Google Drive...", filename),
	}, nil
}

// ============================================================================
// 📍 GOOGLE MAPS — Buscar Locais (API Real)
// ============================================================================

func (h *ToolsHandler) handleFindNearbyPlaces(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	placeType, _ := args["type"].(string)
	location, _ := args["location"].(string)
	radiusFloat, _ := args["radius"].(float64)
	radius := int(radiusFloat)

	if placeType == "" {
		placeType = "pharmacy" // Default para idosos
	}
	if radius == 0 {
		radius = 3000
	}

	if h.mapsAPIKey == "" {
		// Fallback: NotifyFunc para o app fazer a busca
		if h.NotifyFunc != nil {
			h.NotifyFunc(idosoID, "search_places", map[string]interface{}{
				"type":     placeType,
				"location": location,
				"radius":   radius,
			})
		}
		return map[string]interface{}{
			"status":  "buscando",
			"message": "Buscando locais próximos...",
		}, nil
	}

	// Usar API real do Google Maps
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		svc := maps.NewService(ctx, h.mapsAPIKey)
		places, searchErr := svc.FindNearbyPlaces(placeType, location, radius)

		if h.NotifyFunc != nil {
			if searchErr != nil {
				h.NotifyFunc(idosoID, "places_error", map[string]interface{}{"error": searchErr.Error()})
			} else {
				h.NotifyFunc(idosoID, "places_found", map[string]interface{}{
					"places": places,
					"count":  len(places),
					"type":   placeType,
				})
			}
		}
	}()

	return map[string]interface{}{
		"status":  "buscando",
		"type":    placeType,
		"message": fmt.Sprintf("Buscando %s próximos...", placeType),
	}, nil
}

// ============================================================================
// 📱 TELEGRAM — Enviar Mensagem
// ============================================================================

func (h *ToolsHandler) handleSendTelegram(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	chatID, _ := args["chat_id"].(string)
	message, _ := args["message"].(string)

	if chatID == "" {
		return map[string]interface{}{"error": "Informe o chat_id do Telegram"}, nil
	}
	if message == "" {
		return map[string]interface{}{"error": "Informe a mensagem"}, nil
	}

	if h.telegramService == nil {
		return map[string]interface{}{"error": "Telegram não configurado"}, nil
	}

	go func() {
		sendErr := h.telegramService.SendMessage(chatID, message)
		if h.NotifyFunc != nil {
			if sendErr != nil {
				log.Printf("❌ [TELEGRAM] Erro: %v", sendErr)
				h.NotifyFunc(idosoID, "telegram_error", map[string]interface{}{
					"chat_id": chatID,
					"error":   sendErr.Error(),
				})
			} else {
				log.Printf("✅ [TELEGRAM] Mensagem enviada para chat %s", chatID)
				h.NotifyFunc(idosoID, "telegram_sent", map[string]interface{}{
					"chat_id": chatID,
					"message": message,
				})
			}
		}
	}()

	return map[string]interface{}{
		"status":  "enviando",
		"chat_id": chatID,
		"message": "Enviando mensagem no Telegram...",
	}, nil
}

// ============================================================================
// 📂 FILESYSTEM — Ler, Escrever, Listar Arquivos
// ============================================================================

func (h *ToolsHandler) handleReadFile(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return map[string]interface{}{"error": "Informe o caminho do arquivo"}, nil
	}

	if h.filesystemService == nil {
		return map[string]interface{}{"error": "Serviço de filesystem não configurado"}, nil
	}

	content, err := h.filesystemService.ReadFile(path)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro ao ler arquivo: %v", err)}, nil
	}

	return map[string]interface{}{
		"status":  "sucesso",
		"path":    path,
		"content": content,
		"message": fmt.Sprintf("Arquivo '%s' lido com sucesso.", path),
	}, nil
}

func (h *ToolsHandler) handleWriteFile(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)

	if path == "" {
		return map[string]interface{}{"error": "Informe o caminho do arquivo"}, nil
	}
	if content == "" {
		return map[string]interface{}{"error": "Informe o conteúdo a escrever"}, nil
	}

	if h.filesystemService == nil {
		return map[string]interface{}{"error": "Serviço de filesystem não configurado"}, nil
	}

	if err := h.filesystemService.WriteFile(path, content); err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro ao escrever arquivo: %v", err)}, nil
	}

	return map[string]interface{}{
		"status":  "sucesso",
		"path":    path,
		"message": fmt.Sprintf("Arquivo '%s' salvo com sucesso.", path),
	}, nil
}

func (h *ToolsHandler) handleListFiles(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	directory, _ := args["directory"].(string)
	if directory == "" {
		directory = "."
	}

	if h.filesystemService == nil {
		return map[string]interface{}{"error": "Serviço de filesystem não configurado"}, nil
	}

	files, err := h.filesystemService.ListDirectory(directory)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro ao listar diretório: %v", err)}, nil
	}

	return map[string]interface{}{
		"status":    "sucesso",
		"directory": directory,
		"files":     files,
		"count":     len(files),
		"message":   fmt.Sprintf("Diretório '%s' contém %d itens.", directory, len(files)),
	}, nil
}

func (h *ToolsHandler) handleSearchFiles(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return map[string]interface{}{"error": "Informe o padrão de busca"}, nil
	}

	if h.filesystemService == nil {
		return map[string]interface{}{"error": "Serviço de filesystem não configurado"}, nil
	}

	results, err := h.filesystemService.SearchFiles(query)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro na busca: %v", err)}, nil
	}

	return map[string]interface{}{
		"status":  "sucesso",
		"query":   query,
		"results": results,
		"count":   len(results),
		"message": fmt.Sprintf("Encontrados %d arquivos com '%s'.", len(results), query),
	}, nil
}

// ============================================================================
// 🌐 WEB SEARCH & BROWSE
// ============================================================================

func (h *ToolsHandler) handleWebSearch(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return map[string]interface{}{"error": "Informe o que deseja pesquisar"}, nil
	}

	if h.autonomousLearner == nil {
		return map[string]interface{}{"error": "Serviço de pesquisa web não disponível"}, nil
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		insights, searchErr := h.autonomousLearner(ctx, query)

		if h.NotifyFunc != nil {
			if searchErr != nil {
				log.Printf("❌ [WEB] Erro na pesquisa: %v", searchErr)
				h.NotifyFunc(idosoID, "web_search_error", map[string]interface{}{
					"query": query,
					"error": searchErr.Error(),
				})
			} else {
				log.Printf("✅ [WEB] Pesquisa concluída: '%s'", query)
				h.NotifyFunc(idosoID, "web_search_result", map[string]interface{}{
					"query":    query,
					"insights": insights,
				})
			}
		}
	}()

	return map[string]interface{}{
		"status":  "pesquisando",
		"query":   query,
		"message": fmt.Sprintf("Pesquisando na web sobre '%s'...", query),
	}, nil
}

func (h *ToolsHandler) handleBrowseWebpage(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	url, _ := args["url"].(string)
	if url == "" {
		return map[string]interface{}{"error": "Informe a URL da página"}, nil
	}

	// Notifica o app para abrir WebView
	if h.NotifyFunc != nil {
		h.NotifyFunc(idosoID, "show_webpage", map[string]interface{}{
			"url": url,
		})
	}

	return map[string]interface{}{
		"status":  "abrindo",
		"url":     url,
		"message": fmt.Sprintf("Abrindo página: %s", url),
	}, nil
}

// ============================================================================
// 📺 VIDEO & WEB DISPLAY
// ============================================================================

func (h *ToolsHandler) handlePlayVideo(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	url, _ := args["url"].(string)
	videoID, _ := args["video_id"].(string)
	title, _ := args["title"].(string)

	if url == "" && videoID != "" {
		url = fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	}
	if url == "" {
		return map[string]interface{}{"error": "Informe a URL ou ID do vídeo"}, nil
	}

	if h.NotifyFunc != nil {
		h.NotifyFunc(idosoID, "play_video", map[string]interface{}{
			"url":      url,
			"video_id": videoID,
			"title":    title,
		})
	}

	return map[string]interface{}{
		"status":  "reproduzindo",
		"url":     url,
		"title":   title,
		"message": fmt.Sprintf("Reproduzindo vídeo: %s", title),
	}, nil
}

func (h *ToolsHandler) handleShowWebpage(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	url, _ := args["url"].(string)
	title, _ := args["title"].(string)

	if url == "" {
		return map[string]interface{}{"error": "Informe a URL da página"}, nil
	}

	if h.NotifyFunc != nil {
		h.NotifyFunc(idosoID, "show_webpage", map[string]interface{}{
			"url":   url,
			"title": title,
		})
	}

	return map[string]interface{}{
		"status":  "abrindo",
		"url":     url,
		"message": fmt.Sprintf("Mostrando página: %s", title),
	}, nil
}

// ============================================================================
// 💻 SELF-CODING — EVA edita seu próprio código
// ============================================================================

func (h *ToolsHandler) handleEditMyCode(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	filePath, _ := args["file_path"].(string)
	content, _ := args["content"].(string)

	if filePath == "" {
		return map[string]interface{}{"error": "Informe o caminho do arquivo"}, nil
	}
	if content == "" {
		return map[string]interface{}{"error": "Informe o novo conteúdo"}, nil
	}

	if h.selfcodeService == nil {
		return map[string]interface{}{"error": "Serviço de auto-programação não configurado"}, nil
	}

	if err := h.selfcodeService.WriteSourceFile(filePath, content); err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro: %v", err)}, nil
	}

	return map[string]interface{}{
		"status":  "sucesso",
		"path":    filePath,
		"message": fmt.Sprintf("Arquivo '%s' editado com sucesso (branch eva/).", filePath),
	}, nil
}

func (h *ToolsHandler) handleCreateBranch(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	branchName, _ := args["branch_name"].(string)
	if branchName == "" {
		branchName = fmt.Sprintf("feature-%s", time.Now().Format("20060102-150405"))
	}

	if h.selfcodeService == nil {
		return map[string]interface{}{"error": "Serviço de auto-programação não configurado"}, nil
	}

	if err := h.selfcodeService.CreateBranch(branchName); err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro: %v", err)}, nil
	}

	return map[string]interface{}{
		"status":  "sucesso",
		"branch":  "eva/" + branchName,
		"message": fmt.Sprintf("Branch 'eva/%s' criada.", branchName),
	}, nil
}

func (h *ToolsHandler) handleCommitCode(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	message, _ := args["message"].(string)
	if message == "" {
		message = "EVA auto-commit"
	}

	if h.selfcodeService == nil {
		return map[string]interface{}{"error": "Serviço de auto-programação não configurado"}, nil
	}

	if err := h.selfcodeService.CommitChanges(message); err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro: %v", err)}, nil
	}

	return map[string]interface{}{
		"status":  "sucesso",
		"message": fmt.Sprintf("Commit realizado: '%s'", message),
	}, nil
}

func (h *ToolsHandler) handleRunTests(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	if h.selfcodeService == nil {
		return map[string]interface{}{"error": "Serviço de auto-programação não configurado"}, nil
	}

	// Blocking — MCP caller precisa do resultado
	output, testErr := h.selfcodeService.RunTests()

	if testErr != nil {
		return map[string]interface{}{
			"status":  "falhou",
			"output":  output,
			"error":   testErr.Error(),
			"message": "Testes falharam.",
		}, nil
	}

	return map[string]interface{}{
		"status":  "sucesso",
		"output":  output,
		"message": "Todos os testes passaram.",
	}, nil
}

func (h *ToolsHandler) handleGetCodeDiff(idosoID int64, args map[string]interface{}) (map[string]interface{}, error) {
	if h.selfcodeService == nil {
		return map[string]interface{}{"error": "Serviço de auto-programação não configurado"}, nil
	}

	diff, err := h.selfcodeService.GetDiff()
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Erro: %v", err)}, nil
	}

	return map[string]interface{}{
		"status":  "sucesso",
		"diff":    diff,
		"message": "Diferenças no código:",
	}, nil
}
