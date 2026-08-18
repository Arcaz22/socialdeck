package game

import (
	"sync"
)

// Hub menyimpan semua koneksi WebSocket per session
type Hub struct {
	mu       sync.RWMutex
	sessions map[string]map[string]*Client // sessionID → userID → client
}

func NewHub() *Hub {
	return &Hub{
		sessions: make(map[string]map[string]*Client),
	}
}

func (h *Hub) Register(sessionID, userID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.sessions[sessionID] == nil {
		h.sessions[sessionID] = make(map[string]*Client)
	}
	h.sessions[sessionID][userID] = client
}

func (h *Hub) Unregister(sessionID, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, ok := h.sessions[sessionID]; ok {
		delete(room, userID)
		if len(room) == 0 {
			delete(h.sessions, sessionID)
		}
	}
}

// Broadcast ke semua player dalam satu session
func (h *Hub) Broadcast(sessionID string, event OutgoingEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.sessions[sessionID] {
		client.send(event)
	}
}

// Send ke satu player spesifik
func (h *Hub) SendTo(sessionID, userID string, event OutgoingEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if room, ok := h.sessions[sessionID]; ok {
		if client, ok := room[userID]; ok {
			client.send(event)
		}
	}
}

func (h *Hub) ConnectedCount(sessionID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.sessions[sessionID])
}
