package game

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024
)

type Client struct {
	sessionID string
	userID    string
	conn      *websocket.Conn
	sendCh    chan OutgoingEvent
	hub       *Hub
	svc       *Service
}

func NewClient(conn *websocket.Conn, sessionID, userID string, hub *Hub, svc *Service) *Client {
	return &Client{
		sessionID: sessionID,
		userID:    userID,
		conn:      conn,
		sendCh:    make(chan OutgoingEvent, 32),
		hub:       hub,
		svc:       svc,
	}
}

func (c *Client) send(event OutgoingEvent) {
	select {
	case c.sendCh <- event:
	default:
		log.Printf("client %s send buffer full, dropping event", c.userID)
	}
}

// Run — panggil setelah register ke hub
func (c *Client) Run(ctx context.Context) {
	go c.writePump()
	c.readPump(ctx)
}

// ─── Read (Client → Server) ───────────────────────────────────────────────────

func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.hub.Unregister(c.sessionID, c.userID)
		c.conn.Close()
		c.svc.HandleDisconnect(ctx, c.sessionID, c.userID)

		// Broadcast ke room bahwa player disconnect
		c.hub.Broadcast(c.sessionID, OutgoingEvent{
			Type:    EventPlayerLeft,
			Payload: map[string]any{"user_id": c.userID},
		})
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var event IncomingEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			c.send(OutgoingEvent{Type: EventError, Payload: map[string]any{"error": "invalid message format"}})
			continue
		}

		c.handleEvent(ctx, event)
	}
}

// ─── Write (Server → Client) ──────────────────────────────────────────────────

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case event, ok := <-c.sendCh:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(event); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ─── Event Handler ────────────────────────────────────────────────────────────

func (c *Client) handleEvent(ctx context.Context, event IncomingEvent) {
	switch event.Type {

	case "start_game":
		state, err := c.svc.StartGame(ctx, c.sessionID, c.userID)
		if err != nil {
			c.send(OutgoingEvent{Type: EventError, Payload: map[string]any{"error": err.Error()}})
			return
		}
		players, _ := c.svc.GetPlayers(ctx, c.sessionID)
		nextInfo := c.svc.GetNextPlayerInfo(state, players)

		c.hub.Broadcast(c.sessionID, OutgoingEvent{
			Type:    EventGameStarted,
			Payload: nextInfo,
		})

	case "draw_card":
		card, err := c.svc.DrawCard(ctx, c.sessionID, c.userID)
		if err != nil {
			c.send(OutgoingEvent{Type: EventError, Payload: map[string]any{"error": err.Error()}})
			return
		}
		// Broadcast kartu ke semua (semua lihat kartu yang sama)
		c.hub.Broadcast(c.sessionID, OutgoingEvent{
			Type:    EventCardDrawn,
			Payload: card,
		})

	case "submit_result":
		result, _ := event.Payload["result"].(string)
		turnResult, gameOver, err := c.svc.SubmitResult(ctx, c.sessionID, c.userID, result)
		if err != nil {
			c.send(OutgoingEvent{Type: EventError, Payload: map[string]any{"error": err.Error()}})
			return
		}

		// Broadcast hasil turn
		c.hub.Broadcast(c.sessionID, OutgoingEvent{
			Type:    EventTurnResult,
			Payload: turnResult,
		})

		if gameOver {
			c.hub.Broadcast(c.sessionID, OutgoingEvent{
				Type:    EventGameFinished,
				Payload: GameFinishedPayload{Reason: "cards_exhausted"},
			})
			return
		}

		// Broadcast giliran berikutnya
		state, _ := c.svc.GetRedisState(ctx, c.sessionID)
		players, _ := c.svc.GetPlayers(ctx, c.sessionID)
		nextInfo := c.svc.GetNextPlayerInfo(state, players)

		c.hub.Broadcast(c.sessionID, OutgoingEvent{
			Type:    EventTurnChanged,
			Payload: nextInfo,
		})

	case "stop_game":
		if err := c.svc.StopGame(ctx, c.sessionID, c.userID); err != nil {
			c.send(OutgoingEvent{Type: EventError, Payload: map[string]any{"error": err.Error()}})
			return
		}
		c.hub.Broadcast(c.sessionID, OutgoingEvent{
			Type:    EventGameFinished,
			Payload: GameFinishedPayload{Reason: "host_stopped"},
		})
	}
}
