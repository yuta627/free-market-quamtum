CREATE TABLE IF NOT EXISTS likes (
    id         BIGSERIAL   PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users(id),
    product_id BIGINT      NOT NULL REFERENCES products(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_likes_user_product ON likes(user_id, product_id);
CREATE INDEX IF NOT EXISTS idx_likes_product_id ON likes(product_id);
