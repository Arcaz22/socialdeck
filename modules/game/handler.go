package game

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: restrict di production
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Handler struct {
	service *Service
	hub     *Hub
}

func NewHandler(service *Service, hub *Hub) *Handler {
	return &Handler{service: service, hub: hub}
}

// ─── POST /game/rooms ─────────────────────────────────────────────────────────

type CreateRoomRequest struct {
	DeckID string `json:"deck_id" binding:"required"`
}

type GameRoomResponse struct {
	SessionID string `json:"session_id"`
	Code      string `json:"code"`
}

type GameErrorResponse struct {
	Error string `json:"error"`
}

type GameMessageResponse struct {
	Message string `json:"message"`
}

type WebSocketTicketResponse struct {
	Ticket    string `json:"ticket"`
	ExpiresIn int    `json:"expires_in"`
}

// CreateRoom godoc
// @Summary Create game room
// @Description Create a game room from a deck. The authenticated user becomes the host.
// @Tags Game
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateRoomRequest true "Create room request"
// @Success 201 {object} GameRoomResponse
// @Failure 400 {object} GameErrorResponse
// @Failure 401 {object} GameErrorResponse
// @Failure 500 {object} GameErrorResponse
// @Router /game/rooms [post]
func (h *Handler) CreateRoom(c *gin.Context) {
	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	session, err := h.service.CreateRoom(c.Request.Context(), req.DeckID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"session_id": session.ID,
		"code":       session.Code,
	})
}

// ─── POST /game/rooms/join ────────────────────────────────────────────────────

type JoinRoomRequest struct {
	Code string `json:"code" binding:"required,len=6"`
}

// JoinRoom godoc
// @Summary Join game room
// @Description Join an existing waiting game room by room code.
// @Tags Game
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body JoinRoomRequest true "Join room request"
// @Success 200 {object} GameRoomResponse
// @Failure 400 {object} GameErrorResponse
// @Failure 401 {object} GameErrorResponse
// @Failure 500 {object} GameErrorResponse
// @Router /game/rooms/join [post]
func (h *Handler) JoinRoom(c *gin.Context) {
	var req JoinRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	session, err := h.service.JoinRoom(c.Request.Context(), req.Code, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id": session.ID,
		"code":       session.Code,
	})
}

// ─── GET /game/rooms/:id/state ────────────────────────────────────────────────

// GetRoomState godoc
// @Summary Get game room state
// @Description Get current game room state, players, host, and current turn.
// @Tags Game
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Success 200 {object} RoomStatePayload
// @Failure 401 {object} GameErrorResponse
// @Failure 404 {object} GameErrorResponse
// @Failure 500 {object} GameErrorResponse
// @Router /game/rooms/{id}/state [get]
func (h *Handler) GetRoomState(c *gin.Context) {
	sessionID := c.Param("id")
	userID := c.GetString("user_id")

	state, err := h.service.GetRedisState(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "room tidak ditemukan"})
		return
	}

	players, err := h.service.GetPlayers(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Bangun player info list
	playerInfos := make([]PlayerInfo, len(players))
	for i, p := range players {
		playerInfos[i] = PlayerInfo{
			UserID:      p.UserID,
			Username:    p.Username,
			IsConnected: p.IsConnected,
			TurnOrder:   p.TurnOrder,
		}
	}

	var currentTurn string
	if len(state.PlayerOrder) > 0 {
		currentTurn = state.PlayerOrder[state.CurrentTurnIdx%len(state.PlayerOrder)]
	}

	session, _ := h.service.GetSession(c.Request.Context(), sessionID)

	c.JSON(http.StatusOK, RoomStatePayload{
		SessionID:   sessionID,
		Code:        session.Code,
		Status:      state.Status,
		HostID:      state.HostID,
		Players:     playerInfos,
		CurrentTurn: currentTurn,
	})

	_ = userID // bisa dipakai untuk filter info sensitif nanti
}

// ─── POST /game/rooms/:id/leave ───────────────────────────────────────────────

// LeaveRoom godoc
// @Summary Leave game room
// @Description Leave a waiting room. If the host leaves while waiting, the room is finished/cancelled.
// @Tags Game
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Success 200 {object} GameMessageResponse
// @Failure 400 {object} GameErrorResponse
// @Failure 401 {object} GameErrorResponse
// @Failure 403 {object} GameErrorResponse
// @Failure 404 {object} GameErrorResponse
// @Failure 500 {object} GameErrorResponse
// @Router /game/rooms/{id}/leave [post]
func (h *Handler) LeaveRoom(c *gin.Context) {
	sessionID := c.Param("id")
	userID := c.GetString("user_id")

	cancelled, err := h.service.LeaveRoom(c.Request.Context(), sessionID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.hub.Unregister(sessionID, userID)
	if cancelled {
		h.hub.Broadcast(sessionID, OutgoingEvent{
			Type:    EventGameFinished,
			Payload: GameFinishedPayload{Reason: "host_left"},
		})
		c.JSON(http.StatusOK, gin.H{"message": "room cancelled"})
		return
	}

	h.hub.Broadcast(sessionID, OutgoingEvent{
		Type:    EventPlayerLeft,
		Payload: map[string]any{"user_id": userID},
	})
	c.JSON(http.StatusOK, gin.H{"message": "left room"})
}

// ─── GET /game/rooms/:id/ws ───────────────────────────────────────────────────

// CreateWebSocketTicket godoc
// @Summary Create WebSocket ticket
// @Description Create a short-lived one-time ticket for connecting to a game room WebSocket.
// @Tags Game
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Success 200 {object} WebSocketTicketResponse
// @Failure 401 {object} GameErrorResponse
// @Failure 403 {object} GameErrorResponse
// @Failure 500 {object} GameErrorResponse
// @Router /game/rooms/{id}/ws-ticket [post]
func (h *Handler) CreateWebSocketTicket(c *gin.Context) {
	sessionID := c.Param("id")
	userID := c.GetString("user_id")

	ticket, err := h.service.CreateWebSocketTicket(c.Request.Context(), sessionID, userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, WebSocketTicketResponse{
		Ticket:    ticket,
		ExpiresIn: 30,
	})
}

// WebSocket godoc
// @Summary Connect to game room WebSocket
// @Description Upgrade to a WebSocket connection for realtime game events. Get a short-lived ticket from /game/rooms/{id}/ws-ticket first, then pass it as query parameter.
// @Tags Game
// @Produce json
// @Param id path string true "Session ID"
// @Param ticket query string true "Short-lived WebSocket ticket"
// @Success 101 {string} string "Switching Protocols"
// @Failure 401 {object} GameErrorResponse
// @Failure 403 {object} GameErrorResponse
// @Failure 500 {object} GameErrorResponse
// @Router /game/rooms/{id}/ws [get]
func (h *Handler) WebSocket(c *gin.Context) {
	sessionID := c.Param("id")
	if !isWebSocketUpgrade(c.Request) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "websocket upgrade required"})
		return
	}

	userID, err := h.service.ConsumeWebSocketTicket(c.Request.Context(), sessionID, c.Query("ticket"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Validasi user memang ada di room ini
	players, err := h.service.GetPlayers(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	inRoom := false
	for _, p := range players {
		if p.UserID == userID {
			inRoom = true
			break
		}
	}
	if !inRoom {
		c.JSON(http.StatusForbidden, gin.H{"error": "kamu tidak ada di room ini"})
		return
	}

	// Upgrade ke WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// Register client
	client := NewClient(conn, sessionID, userID, h.hub, h.service)
	h.hub.Register(sessionID, userID, client)
	h.service.HandleConnect(c.Request.Context(), sessionID, userID)

	// Broadcast ke room bahwa player (re)join
	h.hub.Broadcast(sessionID, OutgoingEvent{
		Type: EventPlayerJoined,
		Payload: map[string]any{
			"user_id": userID,
		},
	})

	// Kirim room state ke player yang baru connect
	state, _ := h.service.GetRedisState(c.Request.Context(), sessionID)
	session, _ := h.service.GetSession(c.Request.Context(), sessionID)
	playerInfos := make([]PlayerInfo, len(players))
	for i, p := range players {
		playerInfos[i] = PlayerInfo{
			UserID:      p.UserID,
			Username:    p.Username,
			IsConnected: p.IsConnected,
			TurnOrder:   p.TurnOrder,
		}
	}
	var currentTurn string
	if len(state.PlayerOrder) > 0 {
		currentTurn = state.PlayerOrder[state.CurrentTurnIdx%len(state.PlayerOrder)]
	}
	client.send(OutgoingEvent{
		Type: EventRoomState,
		Payload: RoomStatePayload{
			SessionID:   sessionID,
			Code:        session.Code,
			Status:      state.Status,
			HostID:      state.HostID,
			Players:     playerInfos,
			CurrentTurn: currentTurn,
		},
	})

	// Mulai read/write pump
	client.Run(c.Request.Context())
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}
