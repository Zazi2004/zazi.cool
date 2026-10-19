CREATE TABLE pastes (
    id         text        PRIMARY KEY,
    content    text        NOT NULL,
    language   text        NOT NULL DEFAULT 'plaintext',
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

-- Reads filter on expires_at and the purge deletes by it.
CREATE INDEX pastes_expires_at_idx ON pastes (expires_at);
