// Copyright (C) 2025-2026 Jose R F Junior <web2ajax@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var videoUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // Permitir conexões sem Origin (ex: apps mobile, curl)
		}
		allowedOrigins := []string{
			"https://eva-ia.org",
			"https://www.eva-ia.org",
			"https://app.eva-ia.org",
			"http://localhost:3000",
			"http://localhost:8080",
		}
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				return true
			}
		}
		return false
	},
}

// VideoSession represents a video call session
type VideoSession struct {
	SessionID     string
	MobileConn    *websocket.Conn
	AttendantConn *websocket.Conn
	SDPOffer      string // ✅ Store Offer for late-joining attendants
	PatientData   map[string]interface{}
	mu            sync.RWMutex
}

// AttendantConnection stores metadata about a connected attendant
type AttendantConnection struct {
	Conn     *websocket.Conn
	UserType string
	UserID   string
	UserName string
	writeMu  sync.Mutex // per-connection write mutex (gorilla/websocket nao e thread-safe para writes)
}

// AttendantPool manages web attendants waiting for calls
type AttendantPool struct {
	attendants map[*websocket.Conn]*AttendantConnection
	mu         sync.RWMutex
}

func NewAttendantPool() *AttendantPool {
	return &AttendantPool{
		attendants: make(map[*websocket.Conn]*AttendantConnection),
	}
}

func (ap *AttendantPool) Add(conn *websocket.Conn, userType, userID, userName string) error {
	// Validate that only admin users can join the pool
	if userType != "admin" {
		log.Printf("⛔ Non-admin user attempted to join pool: %s (type: %s)", userID, userType)
		return fmt.Errorf("apenas usuários admin podem receber notificações de chamada")
	}

	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.attendants[conn] = &AttendantConnection{
		Conn:     conn,
		UserType: userType,
		UserID:   userID,
		UserName: userName,
	}

	log.Printf("👨‍⚕️ Admin attendant added: %s (%s). Total: %d", userName, userID, len(ap.attendants))
	return nil
}

func (ap *AttendantPool) Remove(conn *websocket.Conn) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if attendant, exists := ap.attendants[conn]; exists {
		log.Printf("👋 Admin attendant removed: %s. Total: %d", attendant.UserName, len(ap.attendants)-1)
		delete(ap.attendants, conn)
	}
}

func (ap *AttendantPool) Broadcast(message map[string]interface{}) {
	ap.mu.RLock()
	attendants := make([]*AttendantConnection, 0, len(ap.attendants))
	for _, a := range ap.attendants {
		attendants = append(attendants, a)
	}
	ap.mu.RUnlock()

	log.Printf("📢 Broadcasting to %d admin attendants", len(attendants))
	for _, attendant := range attendants {
		attendant.writeMu.Lock()
		err := attendant.Conn.WriteJSON(message)
		attendant.writeMu.Unlock()
		if err != nil {
			log.Printf("⚠️ Failed to broadcast to %s: %v", attendant.UserName, err)
		} else {
			log.Printf("✅ Notification sent to admin: %s", attendant.UserName)
		}
	}
}

// VideoSessionManager manages active video sessions
type VideoSessionManager struct {
	sessions      map[string]*VideoSession
	attendantPool *AttendantPool
	mu            sync.RWMutex
}

func NewVideoSessionManager() *VideoSessionManager {
	return &VideoSessionManager{
		sessions:      make(map[string]*VideoSession),
		attendantPool: NewAttendantPool(),
	}
}

// CreateSession initializes a session with an SDP offer
func (vsm *VideoSessionManager) CreateSession(sessionID, sdpOffer string) {
	vsm.mu.Lock()
	defer vsm.mu.Unlock()

	vsm.sessions[sessionID] = &VideoSession{
		SessionID: sessionID,
		SDPOffer:  sdpOffer,
	}
	log.Printf("✅ Video Session created with SDP Offer: %s", sessionID)
}

// GetPendingSessions returns a list of active sessions waiting for an attendant
func (vsm *VideoSessionManager) GetPendingSessions() []map[string]interface{} {
	vsm.mu.RLock()
	defer vsm.mu.RUnlock()

	var pending []map[string]interface{}
	for _, session := range vsm.sessions {
		// If no attendant is connected, it's pending
		if session.AttendantConn == nil {
			pd := session.PatientData
			if pd == nil {
				pd = map[string]interface{}{"nome": "Paciente (dados indisponíveis)"}
			}
			pending = append(pending, map[string]interface{}{
				"session_id":   session.SessionID,
				"patient_data": pd,
				"started_at":   time.Now().UTC().Format(time.RFC3339),
			})
		}
	}
	return pending
}

// RegisterClient registers a WebSocket connection to a session
func (vsm *VideoSessionManager) RegisterClient(sessionID string, conn *websocket.Conn, clientType string, userType string, userID string, userName string) error {
	vsm.mu.Lock()
	defer vsm.mu.Unlock()

	// Handle attendant pool registration (not tied to specific session)
	if clientType == "attendant_pool" {
		err := vsm.attendantPool.Add(conn, userType, userID, userName)
		if err != nil {
			return err
		}
		return nil
	}

	session, exists := vsm.sessions[sessionID]
	if !exists {
		session = &VideoSession{
			SessionID: sessionID,
		}
		vsm.sessions[sessionID] = session
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if clientType == "web_attendant" {
		session.AttendantConn = conn
		log.Printf("✅ Web attendant registered for session: %s", sessionID)

		// ✅ FIX: Send pending SDP Offer if available
		if session.SDPOffer != "" {
			log.Printf("📩 Forwarding pending SDP Offer to web attendant")
			offerMsg := map[string]interface{}{
				"type": "webrtc_signal",
				"payload": map[string]interface{}{
					"type": "offer",
					"sdp":  session.SDPOffer,
				},
			}
			if err := conn.WriteJSON(offerMsg); err != nil {
				log.Printf("❌ Erro ao enviar SDP Offer para attendant: %v", err)
				return fmt.Errorf("falha ao enviar SDP Offer: %w", err)
			}
		}
	} else {
		session.MobileConn = conn
		log.Printf("✅ Mobile client registered for session: %s", sessionID)

		// Broadcast incoming call to all attendants
		vsm.notifyIncomingCall(sessionID, session.PatientData)
	}

	return nil
}

// notifyIncomingCall broadcasts an incoming call notification to all attendants
func (vsm *VideoSessionManager) notifyIncomingCall(sessionID string, patientData map[string]interface{}) {
	if patientData == nil {
		patientData = map[string]interface{}{
			"nome": "Paciente (dados indisponíveis)",
		}
	}

	notification := map[string]interface{}{
		"type":         "incoming_call",
		"session_id":   sessionID,
		"patient_data": patientData,
	}

	vsm.attendantPool.Broadcast(notification)
	log.Printf("📞 Incoming call notification sent for session: %s", sessionID)
}

// notifyEmergencyCall broadcasts a CRITICAL emergency alert to all attendants
func (vsm *VideoSessionManager) notifyEmergencyCall(sessionID string, alertData map[string]interface{}) {
	notification := map[string]interface{}{
		"type":       "incoming_call",
		"priority":   "CRITICAL",
		"session_id": sessionID,
		"alert_data": alertData,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	vsm.attendantPool.Broadcast(notification)
	log.Printf("🚨 EMERGENCY CALL notification sent for session: %s", sessionID)
}

// RouteSignal routes WebRTC signals between mobile and web attendant
func (vsm *VideoSessionManager) RouteSignal(sessionID string, senderConn *websocket.Conn, payload map[string]interface{}) error {
	// Hold manager read lock while acquiring session lock to prevent
	// the session from being deleted between the two lock acquisitions.
	vsm.mu.RLock()
	session, exists := vsm.sessions[sessionID]
	if !exists {
		vsm.mu.RUnlock()
		log.Printf("⚠️ Session not found: %s", sessionID)
		return nil
	}
	session.mu.RLock()
	vsm.mu.RUnlock()
	defer session.mu.RUnlock()

	// Determine target connection (opposite of sender)
	var targetConn *websocket.Conn
	if senderConn == session.MobileConn {
		targetConn = session.AttendantConn
	} else {
		targetConn = session.MobileConn
	}

	if targetConn == nil {
		log.Printf("⚠️ Target client not connected for session: %s", sessionID)
		return nil
	}

	// Forward the signal
	message := map[string]interface{}{
		"type":    "webrtc_signal",
		"payload": payload,
	}

	log.Printf("➡️ Routing %s to target for session: %s", payload["type"], sessionID)
	return targetConn.WriteJSON(message)
}

// UnregisterClient removes a client from a session or attendant pool
func (vsm *VideoSessionManager) UnregisterClient(sessionID string, conn *websocket.Conn, clientType string) {
	// Handle attendant pool removal
	if clientType == "attendant_pool" {
		vsm.attendantPool.Remove(conn)
		return
	}

	vsm.mu.Lock()
	defer vsm.mu.Unlock()

	session, exists := vsm.sessions[sessionID]
	if !exists {
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.MobileConn == conn {
		session.MobileConn = nil
		log.Printf("📱 Mobile client disconnected from session: %s", sessionID)
	} else if session.AttendantConn == conn {
		session.AttendantConn = nil
		log.Printf("💻 Web attendant disconnected from session: %s", sessionID)
	}

	// Clean up empty sessions
	if session.MobileConn == nil && session.AttendantConn == nil {
		delete(vsm.sessions, sessionID)
		log.Printf("🗑️ Session cleaned up: %s", sessionID)
	}
}

// HandleVideoWebSocket handles WebSocket connections for video calls
func HandleVideoWebSocket(vsm *VideoSessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := videoUpgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("❌ Video WebSocket upgrade error: %v", err)
			return
		}
		var sessionID string
		var clientType string

		defer func() {
			if sessionID != "" || clientType == "attendant_pool" {
				vsm.UnregisterClient(sessionID, conn, clientType)
			}
			conn.Close()
		}()

		for {
			var msg map[string]interface{}
			err := conn.ReadJSON(&msg)
			if err != nil {
				// Ignore normal close codes (1001, 1005) or EOF to prevent log spam
				if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNoStatusReceived, websocket.CloseNormalClosure) {
					// Silent close
				} else {
					log.Printf("❌ WebSocket read error: %v", err)
				}
				break
			}

			msgType, _ := msg["type"].(string)

			switch msgType {
			case "register":
				sessionID, _ = msg["session_id"].(string)
				clientType, _ = msg["client_type"].(string)

				// Attendant pool registration with user validation
				if clientType == "attendant_pool" {
					userType, _ := msg["user_type"].(string)
					userID, _ := msg["user_id"].(string)
					userName, _ := msg["user_name"].(string)

					err := vsm.RegisterClient("", conn, clientType, userType, userID, userName)
					if err != nil {
						// Send error response for non-admin users
						conn.WriteJSON(map[string]interface{}{
							"type":    "error",
							"message": err.Error(),
						})
						log.Printf("❌ Registration rejected: %v", err)
						continue
					}

					conn.WriteJSON(map[string]interface{}{
						"type":   "registered",
						"status": "admin_attendant",
					})
					continue
				}

				if sessionID == "" {
					log.Printf("⚠️ Registration missing session_id")
					continue
				}

				// Store patient data sent by mobile client
				if patientData, ok := msg["patient_data"].(map[string]interface{}); ok && clientType != "web_attendant" {
					vsm.mu.Lock()
					if sess, exists := vsm.sessions[sessionID]; exists {
						sess.PatientData = patientData
					} else {
						vsm.sessions[sessionID] = &VideoSession{
							SessionID:   sessionID,
							PatientData: patientData,
						}
					}
					vsm.mu.Unlock()
				}

				if err := vsm.RegisterClient(sessionID, conn, clientType, "", "", ""); err != nil {
					conn.WriteJSON(map[string]interface{}{
						"type":    "error",
						"message": "falha ao registrar: " + err.Error(),
					})
					log.Printf("❌ Registration error: %v", err)
					continue
				}

				// Send confirmation
				if err := conn.WriteJSON(map[string]interface{}{
					"type":    "registered",
					"session": sessionID,
				}); err != nil {
					log.Printf("❌ Erro ao enviar confirmacao de registro: %v", err)
				}

			case "webrtc_signal":
				if sessionID == "" {
					log.Printf("⚠️ Signal received before registration")
					continue
				}

				payload, ok := msg["payload"].(map[string]interface{})
				if !ok {
					log.Printf("⚠️ Invalid signal payload")
					continue
				}

				// Route signal to the other peer
				err := vsm.RouteSignal(sessionID, conn, payload)
				if err != nil {
					log.Printf("❌ Error routing signal: %v", err)
				}

			default:
				log.Printf("⚠️ Unknown message type: %s", msgType)
			}
		}
	}
}
