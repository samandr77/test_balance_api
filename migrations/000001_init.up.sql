CREATE TABLE balances (
    user_id    UUID           PRIMARY KEY,
    amount     NUMERIC(20, 8) NOT NULL CHECK (amount >= 0),
    currency   VARCHAR(10)    NOT NULL DEFAULT 'USDT',
    updated_at TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE withdrawals (
    id              UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID           NOT NULL REFERENCES balances(user_id),
    amount          NUMERIC(20, 8) NOT NULL CHECK (amount > 0),
    currency        VARCHAR(10)    NOT NULL,
    destination     TEXT           NOT NULL,
    status          VARCHAR(20)    NOT NULL DEFAULT 'pending',
    idempotency_key TEXT           NOT NULL,
    payload_hash    TEXT           NOT NULL,
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE ledger_entries (
    id            BIGSERIAL      PRIMARY KEY,
    user_id       UUID           NOT NULL,
    withdrawal_id UUID           REFERENCES withdrawals(id),
    amount        NUMERIC(20, 8) NOT NULL,
    direction     VARCHAR(10)    NOT NULL CHECK (direction IN ('debit', 'credit')),
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);
