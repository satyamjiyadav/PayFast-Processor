CREATE TYPE subscription_status AS ENUM ('active', 'past_due', 'canceled', 'unpaid');
CREATE TYPE invoice_status AS ENUM ('draft', 'open', 'paid', 'uncollectible', 'void');

CREATE TABLE subscriptions (
    id VARCHAR(32) PRIMARY KEY,
    merchant_id VARCHAR(32) NOT NULL REFERENCES merchants(id),
    customer_id VARCHAR(255) NOT NULL, -- Assuming string ID for customer or mapping to a customers table if we had one
    payment_method_id VARCHAR(32) NOT NULL REFERENCES payment_methods(id),
    plan_id VARCHAR(255) NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL,
    interval VARCHAR(20) NOT NULL DEFAULT 'month', -- month, year
    status subscription_status NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_subs_merchant_status ON subscriptions(merchant_id, status);
CREATE INDEX idx_subs_period_end ON subscriptions(current_period_end) WHERE status = 'active';

CREATE TABLE invoices (
    id VARCHAR(32) PRIMARY KEY,
    subscription_id VARCHAR(32) NOT NULL REFERENCES subscriptions(id),
    merchant_id VARCHAR(32) NOT NULL REFERENCES merchants(id),
    amount BIGINT NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL,
    status invoice_status NOT NULL DEFAULT 'draft',
    payment_id VARCHAR(32) REFERENCES payments(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
