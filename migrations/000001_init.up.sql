CREATE TYPE merchant_status AS ENUM ('active', 'suspended', 'pending_verification');
CREATE TYPE payout_schedule AS ENUM ('daily', 'weekly', 'biweekly', 'monthly');

CREATE TABLE merchants (
    id                  VARCHAR(32) PRIMARY KEY,
    name                VARCHAR(255) NOT NULL,
    email               VARCHAR(255) NOT NULL UNIQUE,
    api_key_hash        VARCHAR(255) NOT NULL UNIQUE,
    api_key_prefix      VARCHAR(8)   NOT NULL,
    status              merchant_status NOT NULL DEFAULT 'pending_verification',
    payout_schedule     payout_schedule NOT NULL DEFAULT 'daily',
    settlement_delay_days INT NOT NULL DEFAULT 2,
    bank_account_number VARCHAR(255),
    bank_ifsc_code      VARCHAR(11),
    bank_account_name   VARCHAR(255),
    platform_fee_bps    INT NOT NULL DEFAULT 200,
    webhook_url         TEXT,
    webhook_secret_hash VARCHAR(255),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_merchants_api_key_prefix ON merchants(api_key_prefix);
CREATE INDEX idx_merchants_status ON merchants(status);

CREATE TYPE pm_type AS ENUM ('card', 'upi', 'netbanking');
CREATE TYPE card_brand AS ENUM ('visa', 'mastercard', 'rupay', 'amex', 'unknown');

CREATE TABLE payment_methods (
    id                  VARCHAR(32) PRIMARY KEY,
    customer_id         VARCHAR(32),
    merchant_id         VARCHAR(32) NOT NULL REFERENCES merchants(id),
    type                pm_type NOT NULL,
    card_last_four      VARCHAR(4),
    card_brand          card_brand,
    card_exp_month      SMALLINT,
    card_exp_year       SMALLINT,
    card_fingerprint    VARCHAR(64),
    encrypted_pan       BYTEA,
    encryption_key_id   VARCHAR(64),
    upi_vpa             VARCHAR(255),
    bank_code           VARCHAR(20),
    is_reusable         BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pm_merchant ON payment_methods(merchant_id);
CREATE INDEX idx_pm_customer ON payment_methods(customer_id);
CREATE INDEX idx_pm_fingerprint ON payment_methods(card_fingerprint);

CREATE TYPE payment_status AS ENUM ('created', 'processing', 'requires_action', 'authorized', 'captured', 'failed', 'canceled');
CREATE TYPE payment_method_type AS ENUM ('card', 'upi', 'netbanking');

CREATE TABLE payments (
    id                  VARCHAR(32) PRIMARY KEY,
    merchant_id         VARCHAR(32) NOT NULL REFERENCES merchants(id),
    payment_method_id   VARCHAR(32) REFERENCES payment_methods(id),
    payment_method_type payment_method_type NOT NULL,
    amount              BIGINT NOT NULL CHECK (amount > 0),
    currency            VARCHAR(3) NOT NULL DEFAULT 'INR',
    platform_fee        BIGINT NOT NULL DEFAULT 0,
    net_amount          BIGINT NOT NULL DEFAULT 0,
    status              payment_status NOT NULL DEFAULT 'created',
    failure_reason      TEXT,
    idempotency_key     VARCHAR(255) NOT NULL,
    psp_transaction_id  VARCHAR(255),
    psp_provider        VARCHAR(50),
    description         TEXT,
    metadata            JSONB DEFAULT '{}',
    risk_score          SMALLINT,
    authorized_at       TIMESTAMPTZ,
    captured_at         TIMESTAMPTZ,
    failed_at           TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_idempotency UNIQUE (merchant_id, idempotency_key)
);

CREATE INDEX idx_payments_merchant_status ON payments(merchant_id, status);
CREATE INDEX idx_payments_merchant_created ON payments(merchant_id, created_at DESC);
CREATE INDEX idx_payments_status_captured ON payments(status, captured_at) WHERE status = 'captured';
CREATE INDEX idx_payments_psp_txn ON payments(psp_transaction_id);

CREATE TYPE entry_type AS ENUM ('debit', 'credit');
CREATE TYPE account_type AS ENUM ('merchant_payable', 'platform_revenue', 'bank_transit', 'merchant_settlement', 'refund_payable');

CREATE TABLE accounts (
    id                  VARCHAR(32) PRIMARY KEY,
    merchant_id         VARCHAR(32) REFERENCES merchants(id),
    type                account_type NOT NULL,
    currency            VARCHAR(3) NOT NULL DEFAULT 'INR',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_merchant_account UNIQUE (merchant_id, type, currency)
);

CREATE TABLE ledger_entries (
    id                  VARCHAR(32) PRIMARY KEY,
    transaction_group_id VARCHAR(32) NOT NULL,
    account_id          VARCHAR(32) NOT NULL REFERENCES accounts(id),
    payment_id          VARCHAR(32) REFERENCES payments(id),
    entry_type          entry_type NOT NULL,
    amount              BIGINT NOT NULL CHECK (amount > 0),
    currency            VARCHAR(3) NOT NULL DEFAULT 'INR',
    description         TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ledger_account ON ledger_entries(account_id, created_at DESC);
CREATE INDEX idx_ledger_txn_group ON ledger_entries(transaction_group_id);
CREATE INDEX idx_ledger_payment ON ledger_entries(payment_id);

CREATE TABLE idempotency_keys (
    key                 VARCHAR(255) NOT NULL,
    merchant_id         VARCHAR(32) NOT NULL,
    request_path        VARCHAR(255) NOT NULL,
    request_method      VARCHAR(10) NOT NULL,
    response_code       SMALLINT,
    response_body       JSONB,
    status              VARCHAR(20) NOT NULL DEFAULT 'in_progress',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '24 hours',
    PRIMARY KEY (merchant_id, key)
);
