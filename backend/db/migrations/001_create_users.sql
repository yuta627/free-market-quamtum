CREATE TABLE IF NOT EXISTS users (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(100)  NOT NULL,
    email           VARCHAR(255)  NOT NULL,
    hashed_password VARCHAR(255)  NOT NULL,
    avatar_url      TEXT          NOT NULL DEFAULT '',
    bio             TEXT          NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE        INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
