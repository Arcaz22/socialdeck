package game

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// ─── Session ──────────────────────────────────────────────────────────────────

func (r *Repository) CreateSession(ctx context.Context, code, deckID, hostID string) (*Session, error) {
	var s Session
	err := r.db.GetContext(ctx, &s, `
		INSERT INTO game_sessions (code, deck_id, host_id)
		VALUES ($1, $2, $3)
		RETURNING id, code, deck_id, host_id, status, current_turn_idx, created_at
	`, code, deckID, hostID)
	return &s, err
}

func (r *Repository) FindSessionByCode(ctx context.Context, code string) (*Session, error) {
	var s Session
	err := r.db.GetContext(ctx, &s, `
		SELECT id, code, deck_id, host_id, status, current_turn_idx, created_at, finished_at
		FROM game_sessions WHERE code = $1
	`, code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &s, err
}

func (r *Repository) FindSessionByID(ctx context.Context, id string) (*Session, error) {
	var s Session
	err := r.db.GetContext(ctx, &s, `
		SELECT id, code, deck_id, host_id, status, current_turn_idx, created_at, finished_at
		FROM game_sessions WHERE id = $1
	`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &s, err
}

func (r *Repository) UpdateSessionStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE game_sessions SET status = $1,
		finished_at = CASE WHEN $1 = 'finished' THEN NOW() ELSE NULL END
		WHERE id = $2
	`, status, id)
	return err
}

func (r *Repository) UpdateTurnIdx(ctx context.Context, id string, idx int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE game_sessions SET current_turn_idx = $1 WHERE id = $2`, idx, id)
	return err
}

func (r *Repository) ExpireWaitingSessions(ctx context.Context, olderThan time.Duration) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE game_sessions
		SET status = 'finished', finished_at = NOW()
		WHERE status = 'waiting'
		  AND created_at < NOW() - make_interval(secs => $1)
	`, int(olderThan.Seconds()))
	return err
}

// ─── Players ──────────────────────────────────────────────────────────────────

func (r *Repository) AddPlayer(ctx context.Context, sessionID, userID string, turnOrder int) (*Player, error) {
	var p Player
	err := r.db.GetContext(ctx, &p, `
		INSERT INTO game_players (session_id, user_id, turn_order)
		VALUES ($1, $2, $3)
		ON CONFLICT (session_id, user_id) DO UPDATE SET turn_order = EXCLUDED.turn_order
		RETURNING id, session_id, user_id, turn_order, is_connected, joined_at
	`, sessionID, userID, turnOrder)
	return &p, err
}

func (r *Repository) GetPlayers(ctx context.Context, sessionID string) ([]Player, error) {
	var players []Player
	err := r.db.SelectContext(ctx, &players, `
		SELECT gp.id, gp.session_id, gp.user_id, u.username,
		       gp.turn_order, gp.is_connected, gp.joined_at
		FROM game_players gp
		JOIN users u ON u.id = gp.user_id
		WHERE gp.session_id = $1
		ORDER BY gp.turn_order ASC
	`, sessionID)
	return players, err
}

func (r *Repository) SetPlayerConnected(ctx context.Context, sessionID, userID string, connected bool) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE game_players SET is_connected = $1
		WHERE session_id = $2 AND user_id = $3
	`, connected, sessionID, userID)
	return err
}

func (r *Repository) CountPlayers(ctx context.Context, sessionID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM game_players WHERE session_id = $1`, sessionID,
	).Scan(&count)
	return count, err
}

func (r *Repository) PlayerExists(ctx context.Context, sessionID, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM game_players
			WHERE session_id = $1 AND user_id = $2
		)
	`, sessionID, userID).Scan(&exists)
	return exists, err
}

func (r *Repository) RemovePlayer(ctx context.Context, sessionID, userID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var turnOrder int
	err = tx.QueryRowContext(ctx, `
		DELETE FROM game_players
		WHERE session_id = $1 AND user_id = $2
		RETURNING turn_order
	`, sessionID, userID).Scan(&turnOrder)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE game_players
		SET turn_order = -turn_order - 1
		WHERE session_id = $1 AND turn_order > $2
	`, sessionID, turnOrder); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE game_players
		SET turn_order = -turn_order - 2
		WHERE session_id = $1 AND turn_order < 0
	`, sessionID); err != nil {
		return err
	}

	return tx.Commit()
}

// ─── Cards ────────────────────────────────────────────────────────────────────

func (r *Repository) CountDeckCards(ctx context.Context, deckID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM cards WHERE deck_id = $1`, deckID,
	).Scan(&count)
	return count, err
}

func (r *Repository) DrawCard(ctx context.Context, deckID string, playedIDs []string) (*struct {
	ID      string `db:"id"`
	Type    string `db:"type"`
	Content string `db:"content"`
}, error) {
	card := &struct {
		ID      string `db:"id"`
		Type    string `db:"type"`
		Content string `db:"content"`
	}{}

	err := r.db.GetContext(ctx, card, `
		SELECT id, type, content FROM cards
		WHERE deck_id = $1
		  AND NOT (id = ANY($2::uuid[]))
		ORDER BY RANDOM()
		LIMIT 1
	`, deckID, playedIDs)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // semua kartu sudah keluar
	}
	return card, err
}

// ─── Played Cards ─────────────────────────────────────────────────────────────

func (r *Repository) RecordPlayedCard(ctx context.Context, sessionID, playerID, cardID, result string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO played_cards (session_id, player_id, card_id, result)
		VALUES ($1, $2, $3, $4)
	`, sessionID, playerID, cardID, result)
	return err
}
