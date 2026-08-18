package domain

import (
	"errors"
	"time"
)

// ─── Auth ─────────────────────────────────────────────────────────────────────

type User struct {
	ID           string    `db:"id"`
	Username     string    `db:"username"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// ─── Deck ─────────────────────────────────────────────────────────────────────

type Deck struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	Mode      string    `db:"mode"`      // truth_or_dare | truth_or_truth | talk_more
	IsPublic  bool      `db:"is_public"`
	IsSystem  bool      `db:"is_system"` // true = deck bawaan app, tidak bisa diedit
	CreatedBy *string   `db:"created_by"`
	CardCount int       `db:"card_count"`
	CreatedAt time.Time `db:"created_at"`
}

type Card struct {
	ID      string `db:"id"`
	DeckID  string `db:"deck_id"`
	Type    string `db:"type"`    // truth | dare
	Content string `db:"content"`
}

// ─── Game Session ─────────────────────────────────────────────────────────────

type Session struct {
	ID             string     `db:"id"`
	Code           string     `db:"code"`
	DeckID         string     `db:"deck_id"`
	HostID         string     `db:"host_id"`
	Status         string     `db:"status"`           // waiting | active | finished
	CurrentTurnIdx int        `db:"current_turn_idx"`
	CreatedAt      time.Time  `db:"created_at"`
	FinishedAt     *time.Time `db:"finished_at"`
}

type GamePlayer struct {
	ID          string    `db:"id"`
	SessionID   string    `db:"session_id"`
	UserID      string    `db:"user_id"`
	Username    string    `db:"username"`
	TurnOrder   int       `db:"turn_order"`
	IsConnected bool      `db:"is_connected"`
	JoinedAt    time.Time `db:"joined_at"`
}

type PlayedCard struct {
	ID        string    `db:"id"`
	SessionID string    `db:"session_id"`
	PlayerID  string    `db:"player_id"`
	CardID    string    `db:"card_id"`
	Result    string    `db:"result"` // done | pass
	PlayedAt  time.Time `db:"played_at"`
}

// ─── Game Redis State ─────────────────────────────────────────────────────────

type RedisSession struct {
	SessionID      string   `json:"session_id"`
	DeckID         string   `json:"deck_id"`
	HostID         string   `json:"host_id"`
	Status         string   `json:"status"`
	PlayerOrder    []string `json:"player_order"`    // []userID urut giliran
	CurrentTurnIdx int      `json:"current_turn_idx"`
	PlayedCardIDs  []string `json:"played_card_ids"` // kartu yang sudah keluar
	TotalCards     int      `json:"total_cards"`
}

// ─── WebSocket Events ─────────────────────────────────────────────────────────

const (
	EventPlayerJoined   = "player_joined"
	EventPlayerLeft     = "player_left"
	EventGameStarted    = "game_started"
	EventCardDrawn      = "card_drawn"
	EventTurnResult     = "turn_result"
	EventTurnChanged    = "turn_changed"
	EventGameFinished   = "game_finished"
	EventPlayerRejoined = "player_rejoined"
	EventError          = "error"
	EventRoomState      = "room_state"
)

type IncomingEvent struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

type OutgoingEvent struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// ─── WebSocket Payloads ───────────────────────────────────────────────────────

type RoomStatePayload struct {
	SessionID   string       `json:"session_id"`
	Code        string       `json:"code"`
	Status      string       `json:"status"`
	HostID      string       `json:"host_id"`
	Players     []PlayerInfo `json:"players"`
	CurrentTurn string       `json:"current_turn_user_id"`
}

type PlayerInfo struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	IsConnected bool   `json:"is_connected"`
	TurnOrder   int    `json:"turn_order"`
}

type CardDrawnPayload struct {
	CardID  string `json:"card_id"`
	Type    string `json:"type"`
	Content string `json:"content"`
	DrawnBy string `json:"drawn_by_user_id"`
}

type TurnResultPayload struct {
	PlayerID  string `json:"player_id"`
	CardID    string `json:"card_id"`
	Result    string `json:"result"` // done | pass
	CardsLeft int    `json:"cards_left"`
}

type TurnChangedPayload struct {
	NextPlayerID   string `json:"next_player_id"`
	NextPlayerName string `json:"next_player_name"`
	CurrentTurnIdx int    `json:"current_turn_idx"`
}

type GameFinishedPayload struct {
	Reason string `json:"reason"` // cards_exhausted | host_stopped
}

// ─── Errors ───────────────────────────────────────────────────────────────────

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailTaken        = errors.New("email already registered")
	ErrUsernameTaken     = errors.New("username already taken")
	ErrInvalidCredential = errors.New("invalid email or password")
	ErrInvalidToken      = errors.New("invalid or expired token")
	ErrDeckNotFound      = errors.New("deck not found")
	ErrDeckForbidden     = errors.New("access denied")
	ErrCardNotFound      = errors.New("card not found")
	ErrSessionNotFound   = errors.New("session not found")
	ErrNotInRoom         = errors.New("kamu tidak ada di room ini")
	ErrRoomFull          = errors.New("room sudah penuh")
	ErrGameAlreadyActive = errors.New("game sudah dimulai")
	ErrNotYourTurn       = errors.New("bukan giliran kamu")
	ErrNoActiveCard      = errors.New("tarik kartu dulu sebelum submit")
	ErrCardsExhausted    = errors.New("semua kartu sudah dimainkan")
)

// ErrInvalidInput — validation error dengan pesan custom
type ErrInvalidInput string

func (e ErrInvalidInput) Error() string { return string(e) }
