package deck

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/socialdeck/backend/domain"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// ─── Deck ─────────────────────────────────────────────────────────────────────

// ListPublic — semua deck public (sistem + milik user lain yang public)
func (r *Repository) ListPublic(ctx context.Context) ([]domain.Deck, error) {
	var decks []domain.Deck
	err := r.db.SelectContext(ctx, &decks, `
		SELECT d.id, d.name, d.mode, d.is_public, d.is_system,
		       d.created_by, d.created_at,
		       COUNT(c.id) AS card_count
		FROM decks d
		LEFT JOIN cards c ON c.deck_id = d.id
		WHERE d.is_public = true
		GROUP BY d.id
		ORDER BY d.is_system DESC, d.created_at DESC
	`)
	return decks, err
}

// ListByUser — deck milik user (public + private)
func (r *Repository) ListByUser(ctx context.Context, userID string) ([]domain.Deck, error) {
	var decks []domain.Deck
	err := r.db.SelectContext(ctx, &decks, `
		SELECT d.id, d.name, d.mode, d.is_public, d.is_system,
		       d.created_by, d.created_at,
		       COUNT(c.id) AS card_count
		FROM decks d
		LEFT JOIN cards c ON c.deck_id = d.id
		WHERE d.created_by = $1
		GROUP BY d.id
		ORDER BY d.created_at DESC
	`, userID)
	return decks, err
}

// FindByID — detail deck + cards
func (r *Repository) FindByID(ctx context.Context, id string) (*domain.Deck, error) {
	var deck domain.Deck
	err := r.db.GetContext(ctx, &deck, `
		SELECT d.id, d.name, d.mode, d.is_public, d.is_system,
		       d.created_by, d.created_at,
		       COUNT(c.id) AS card_count
		FROM decks d
		LEFT JOIN cards c ON c.deck_id = d.id
		WHERE d.id = $1
		GROUP BY d.id
	`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &deck, err
}

// Create — buat deck baru oleh user
func (r *Repository) Create(ctx context.Context, name, mode string, isPublic bool, userID string) (*domain.Deck, error) {
	var deck domain.Deck
	err := r.db.GetContext(ctx, &deck, `
		INSERT INTO decks (name, mode, is_public, is_system, created_by)
		VALUES ($1, $2, $3, false, $4)
		RETURNING id, name, mode, is_public, is_system, created_by, created_at
	`, name, mode, isPublic, userID)
	if err != nil {
		return nil, err
	}
	deck.CardCount = 0
	return &deck, nil
}

// Update — edit nama/mode/visibility deck
func (r *Repository) Update(ctx context.Context, id, name, mode string, isPublic bool) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE decks
		SET name = $1, mode = $2, is_public = $3, updated_at = NOW()
		WHERE id = $4
	`, name, mode, isPublic, id)
	return err
}

// Delete — hapus deck beserta cards-nya (cascade di DB)
func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM decks WHERE id = $1`, id)
	return err
}

// ─── Card ─────────────────────────────────────────────────────────────────────

func (r *Repository) GetCards(ctx context.Context, deckID string) ([]domain.Card, error) {
	var cards []domain.Card
	err := r.db.SelectContext(ctx, &cards, `
		SELECT id, deck_id, type, content
		FROM cards
		WHERE deck_id = $1
		ORDER BY created_at ASC
	`, deckID)
	return cards, err
}

func (r *Repository) AddCard(ctx context.Context, deckID, cardType, content string) (*domain.Card, error) {
	var card domain.Card
	err := r.db.GetContext(ctx, &card, `
		INSERT INTO cards (deck_id, type, content)
		VALUES ($1, $2, $3)
		RETURNING id, deck_id, type, content
	`, deckID, cardType, content)
	return &card, err
}

func (r *Repository) DeleteCard(ctx context.Context, deckID, cardID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM cards WHERE id = $1 AND deck_id = $2`, cardID, deckID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrCardNotFound
	}

	return nil
}
