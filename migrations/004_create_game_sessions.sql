CREATE TABLE IF NOT EXISTS game_sessions (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    code             CHAR(6)     NOT NULL UNIQUE,
    deck_id          UUID        NOT NULL REFERENCES decks(id),
    host_id          UUID        NOT NULL REFERENCES users(id),
    status           TEXT        NOT NULL DEFAULT 'waiting' CHECK (status IN ('waiting','active','finished')),
    current_turn_idx INT         NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at      TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS game_players (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   UUID        NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    user_id      UUID        NOT NULL REFERENCES users(id),
    turn_order   INT         NOT NULL,
    is_connected BOOLEAN     NOT NULL DEFAULT false,
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(session_id, user_id),
    UNIQUE(session_id, turn_order)
);

CREATE TABLE IF NOT EXISTS played_cards (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID        NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    player_id  UUID        NOT NULL REFERENCES users(id),
    card_id    UUID        NOT NULL REFERENCES cards(id),
    result     TEXT        NOT NULL CHECK (result IN ('done','pass')),
    played_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_game_sessions_code       ON game_sessions(code);
CREATE INDEX IF NOT EXISTS idx_game_sessions_host       ON game_sessions(host_id);
CREATE INDEX IF NOT EXISTS idx_game_players_session     ON game_players(session_id);
CREATE INDEX IF NOT EXISTS idx_played_cards_session     ON played_cards(session_id);
