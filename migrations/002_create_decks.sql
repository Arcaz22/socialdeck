-- Decks
CREATE TABLE IF NOT EXISTS decks (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    mode       TEXT        NOT NULL CHECK (mode IN ('truth_or_truth', 'truth_or_dare', 'talk_more')),
    is_public  BOOLEAN     NOT NULL DEFAULT true,
    is_system  BOOLEAN     NOT NULL DEFAULT false,  -- true = deck bawaan app
    created_by UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Cards
CREATE TABLE IF NOT EXISTS cards (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    deck_id    UUID        NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    type       TEXT        NOT NULL CHECK (type IN ('truth', 'dare')),
    content    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cards_deck_id ON cards(deck_id);
CREATE INDEX IF NOT EXISTS idx_decks_created_by ON decks(created_by);
CREATE INDEX IF NOT EXISTS idx_decks_is_public ON decks(is_public);
CREATE UNIQUE INDEX IF NOT EXISTS idx_cards_unique_seed ON cards(deck_id, type, content);
