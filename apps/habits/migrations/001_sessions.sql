-- Sessions of the single user. Only the SHA-256 of the cookie token is stored.
-- password_hash_fp identifies the password hash in use when the session was opened:
-- changing ADMIN_PASSWORD_HASH invalidates every session.
CREATE TABLE sessions (
    token_hash       bytea       PRIMARY KEY,
    password_hash_fp bytea       NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    expires_at       timestamptz NOT NULL
);

CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);
