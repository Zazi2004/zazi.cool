-- Uploaded images, already resized and re-encoded by the server.
CREATE TABLE images (
    id            uuid        PRIMARY KEY DEFAULT uuidv7(),
    content_type  text        NOT NULL CHECK (content_type IN ('image/jpeg', 'image/png')),
    data          bytea       NOT NULL,
    width         integer     NOT NULL CHECK (width > 0),
    height        integer     NOT NULL CHECK (height > 0),
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE pages (
    id               uuid        PRIMARY KEY DEFAULT uuidv7(),
    slug             text        NOT NULL UNIQUE,
    title            text        NOT NULL,
    description      text        NOT NULL DEFAULT '',
    avatar_image_id  uuid        REFERENCES images (id),
    avatar_shape     text,
    theme            text        NOT NULL,
    accent           text        NOT NULL,
    published        boolean     NOT NULL DEFAULT false,
    listed           boolean     NOT NULL DEFAULT true,
    position         integer     NOT NULL,
    version          integer     NOT NULL DEFAULT 1,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CHECK (avatar_image_id IS NULL OR avatar_shape IS NULL)
);

-- Links and groups of a page. A top-level link has no parent; a link inside a
-- group has the group as parent. Groups are top-level only.
CREATE TABLE items (
    id               uuid        PRIMARY KEY,
    page_id          uuid        NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
    parent_id        uuid        REFERENCES items (id) ON DELETE CASCADE,
    kind             text        NOT NULL CHECK (kind IN ('link', 'group')),
    position         integer     NOT NULL,
    title            text        NOT NULL,
    subtitle         text        NOT NULL DEFAULT '',
    url              text,
    image_id         uuid        REFERENCES images (id),
    shape            text,
    open_by_default  boolean     NOT NULL DEFAULT false,
    published        boolean     NOT NULL DEFAULT true,
    CHECK ((kind = 'link') = (url IS NOT NULL)),
    CHECK (kind = 'link' OR parent_id IS NULL),
    CHECK (kind = 'group' OR shape IS NULL),
    CHECK (image_id IS NULL OR shape IS NULL)
);
CREATE INDEX items_page_id_idx ON items (page_id);
-- Lets the orphan clean-up find image references without scanning every item.
CREATE INDEX items_image_id_idx ON items (image_id) WHERE image_id IS NOT NULL;

CREATE TABLE sessions (
    token_hash            bytea       PRIMARY KEY,
    password_fingerprint  bytea       NOT NULL,
    created_at            timestamptz NOT NULL DEFAULT now(),
    expires_at            timestamptz NOT NULL
);
