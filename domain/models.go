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

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailTaken        = errors.New("email already registered")
	ErrUsernameTaken     = errors.New("username already taken")
	ErrInvalidCredential = errors.New("invalid email or password")
	ErrInvalidToken      = errors.New("invalid or expired token")
)
