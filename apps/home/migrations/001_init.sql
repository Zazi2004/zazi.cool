-- Uploaded banners, already resized and re-encoded by the server.
CREATE TABLE images (
    id            uuid        PRIMARY KEY DEFAULT uuidv7(),
    content_type  text        NOT NULL CHECK (content_type IN ('image/jpeg', 'image/png')),
    data          bytea       NOT NULL,
    width         integer     NOT NULL CHECK (width > 0),
    height        integer     NOT NULL CHECK (height > 0),
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- The homepage content: a single row. banner_choice is 'none', 'upload' or the id
-- of a built-in illustration (validated in Go). The last uploaded image is kept in
-- upload_image_id even when another banner is chosen.
CREATE TABLE content (
    id               smallint    PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    title            text        NOT NULL,
    body             text        NOT NULL DEFAULT '',
    banner_choice    text        NOT NULL DEFAULT 'none',
    upload_image_id  uuid        REFERENCES images (id),
    banner_alt       text        NOT NULL DEFAULT '',
    background       text        NOT NULL DEFAULT 'aurore',
    version          integer     NOT NULL DEFAULT 1,
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CHECK (banner_choice <> 'upload' OR upload_image_id IS NOT NULL)
);

INSERT INTO content (title, body) VALUES (
    'zazi.cool',
    'Bienvenue ! Ici vivent mes petits services web faits maison : un jeu à jouer entre amis, un pastebin et mes pages de liens.'
);

CREATE TABLE sessions (
    token_hash            bytea       PRIMARY KEY,
    password_fingerprint  bytea       NOT NULL,
    created_at            timestamptz NOT NULL DEFAULT now(),
    expires_at            timestamptz NOT NULL
);
