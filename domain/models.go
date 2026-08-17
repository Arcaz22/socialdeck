package domain

import (
	"errors"
	"time"
)

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

// ─── Deck & Card ─────────────────────────────────────────────────────────────

type Deck struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Mode      string    `db:"mode" json:"mode"`
	IsPublic  bool      `db:"is_public" json:"is_public"`
	IsSystem  bool      `db:"is_system" json:"is_system"`
	CreatedBy *string   `db:"created_by" json:"created_by"`
	CardCount int       `db:"card_count" json:"card_count"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type Card struct {
	ID      string `db:"id" json:"id"`
	DeckID  string `db:"deck_id" json:"deck_id"`
	Type    string `db:"type" json:"type"` // truth | dare
	Content string `db:"content" json:"content"`
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
)

// ErrInvalidInput — validation error dengan pesan custom
type ErrInvalidInput string

func (e ErrInvalidInput) Error() string { return string(e) }
