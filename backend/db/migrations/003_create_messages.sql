CREATE TABLE IF NOT EXISTS messages (
    id         BIGSERIAL   PRIMARY KEY,
    product_id BIGINT      NOT NULL REFERENCES products(id),
    sender_id  BIGINT      NOT NULL REFERENCES users(id),
    body       TEXT        NOT NULL,
    is_read    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_messages_product_id ON messages(product_id);
CREATE INDEX IF NOT EXISTS idx_messages_sender_id  ON messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_deleted_at ON messages(deleted_at);
