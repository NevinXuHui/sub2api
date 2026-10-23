CREATE TABLE IF NOT EXISTS account_daily_discount_receipts (
    request_id TEXT NOT NULL,
    api_key_id BIGINT NOT NULL,
    account_id BIGINT NOT NULL,
    billed_at TIMESTAMPTZ NOT NULL,
    factor NUMERIC(12,8) NOT NULL CHECK (factor > 0 AND factor <= 1),
    actual_cost NUMERIC(20,8) NOT NULL CHECK (actual_cost >= 0),
    PRIMARY KEY (request_id, api_key_id)
);
CREATE INDEX IF NOT EXISTS idx_account_daily_discount_receipts_account_time
    ON account_daily_discount_receipts (account_id, billed_at);
