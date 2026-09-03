ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'initiated';
ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'processed';
ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'settled';
ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'refunded';

ALTER TYPE payout_schedule ADD VALUE IF NOT EXISTS 'instant';
ALTER TYPE payout_schedule ADD VALUE IF NOT EXISTS '1_hour';
ALTER TYPE payout_schedule ADD VALUE IF NOT EXISTS '12_hours';
ALTER TYPE payout_schedule ADD VALUE IF NOT EXISTS '24_hours';

CREATE TABLE IF NOT EXISTS refunds (
    id          VARCHAR(50) PRIMARY KEY,
    payment_id  VARCHAR(50) NOT NULL REFERENCES payments(id),
    merchant_id VARCHAR(50) NOT NULL REFERENCES merchants(id),
    amount      BIGINT NOT NULL,
    currency    VARCHAR(3) NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'processed',
    created_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

