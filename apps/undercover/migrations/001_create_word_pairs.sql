CREATE TABLE word_pairs (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    word_a      text        NOT NULL,
    word_b      text        NOT NULL,
    -- Normalized words, in canonical order: the UNIQUE constraint then rejects
    -- "chat / tigre" and "tigre / chat" alike, even under concurrent submissions.
    norm_a      text        NOT NULL,
    norm_b      text        NOT NULL,
    extreme     boolean     NOT NULL DEFAULT false,
    hidden      boolean     NOT NULL DEFAULT false,
    source      text        NOT NULL CHECK (source IN ('seed', 'suggestion')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    CHECK (norm_a < norm_b),
    UNIQUE (norm_a, norm_b)
);

CREATE TABLE pair_reports (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    pair_id        bigint      NOT NULL REFERENCES word_pairs (id) ON DELETE CASCADE,
    reason         text        NOT NULL CHECK (reason IN ('extreme', 'remove')),
    reporter_hash  text        NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (pair_id, reason, reporter_hash)
);
