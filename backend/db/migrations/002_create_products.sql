CREATE TYPE product_status    AS ENUM ('on_sale', 'sold', 'draft');
CREATE TYPE product_condition AS ENUM ('new', 'like_new', 'good', 'fair', 'poor');

CREATE TABLE IF NOT EXISTS products (
    id          BIGSERIAL         PRIMARY KEY,
    seller_id   BIGINT            NOT NULL REFERENCES users(id),
    buyer_id    BIGINT            REFERENCES users(id),
    title       VARCHAR(200)      NOT NULL,
    description TEXT              NOT NULL DEFAULT '',
    price       INTEGER           NOT NULL CHECK (price >= 0),
    status      product_status    NOT NULL DEFAULT 'on_sale',
    condition   product_condition NOT NULL,
    category_id BIGINT,
    image_urls  TEXT              NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_products_seller_id  ON products(seller_id);
CREATE INDEX IF NOT EXISTS idx_products_buyer_id   ON products(buyer_id);
CREATE INDEX IF NOT EXISTS idx_products_category_id ON products(category_id);
CREATE INDEX IF NOT EXISTS idx_products_status     ON products(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_products_deleted_at ON products(deleted_at);
