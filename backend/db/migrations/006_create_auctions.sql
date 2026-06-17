CREATE TABLE IF NOT EXISTS auctions (
    id              BIGSERIAL       PRIMARY KEY,
    product_id      BIGINT          NOT NULL REFERENCES products(id),
    starting_price  INTEGER         NOT NULL CHECK (starting_price >= 0),
    current_price   INTEGER         NOT NULL CHECK (current_price >= 0),
    bid_count       INTEGER         NOT NULL DEFAULT 0,
    ends_at         TIMESTAMPTZ     NOT NULL,
    status          VARCHAR(20)     NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auctions_product_id ON auctions(product_id);
CREATE INDEX IF NOT EXISTS idx_auctions_status      ON auctions(status);
CREATE INDEX IF NOT EXISTS idx_auctions_ends_at     ON auctions(ends_at);

CREATE TABLE IF NOT EXISTS bids (
    id          BIGSERIAL   PRIMARY KEY,
    auction_id  BIGINT      NOT NULL REFERENCES auctions(id),
    bidder_id   BIGINT      NOT NULL REFERENCES users(id),
    amount      INTEGER     NOT NULL CHECK (amount > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bids_auction_id ON bids(auction_id);
CREATE INDEX IF NOT EXISTS idx_bids_bidder_id  ON bids(bidder_id);
