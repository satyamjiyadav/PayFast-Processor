# Implementation Plan: Stripe-like Payment Processor

> Reference: [payment_system_design.md](file:///Users/satyam/Desktop/Payment_Processor/payment_system_design.md)

---

## 1. Tech Stack

| Layer | Technology | Why |
|---|---|---|
| **Language** | Go 1.22+ | High concurrency, low latency, strong typing — ideal for financial systems |
| **HTTP Framework** | `net/http` + `chi` router | Lightweight, stdlib-compatible, middleware-friendly |
| **OLTP Database** | PostgreSQL 16 | ACID compliance, JSONB support, partitioning for scale |
| **Migrations** | `golang-migrate/migrate` | Version-controlled, reversible schema changes |
| **Message Broker** | Apache Kafka (KRaft, open-source) | Durable async events, consumer groups, exactly-once semantics |
| **Kafka Client** | `segmentio/kafka-go` | Pure Go, no CGO dependency |
| **Cache / Idempotency** | Redis 7 | Sub-ms latency for idempotency key lookups and rate limiting |
| **Encryption** | AES-256-GCM via Go `crypto/aes` | Authenticated encryption for card vault |
| **Observability** | Prometheus + Grafana + structured logging (`slog`) | Metrics, dashboards, and queryable logs |
| **Containerization** | Docker + Docker Compose | Local dev parity and reproducible environments |
| **Testing** | Go `testing` + `testcontainers-go` | Unit + integration tests with real Postgres/Kafka/Redis |

---

## 2. Project Structure (Monorepo — Scalable Microservices)

We use a **monorepo with clear service boundaries**. Each service under `services/` is independently deployable. Shared code lives in `internal/` and `pkg/`. Adding a new service in the future is as simple as creating a new folder under `services/`.

```
Payment_Processor/
├── payment_system_design.md        # System design document
├── implementation.md               # This file
├── docker-compose.yml              # Postgres + Redis + Kafka + Zookeeper
├── Makefile                        # Build, test, migrate, run commands
│
├── migrations/                     # PostgreSQL migration files (shared DB)
│   ├── 000001_create_merchants.up.sql
│   ├── 000001_create_merchants.down.sql
│   ├── 000002_create_payment_methods.up.sql
│   ├── ...
│
├── internal/                       # Shared internal packages (not importable outside)
│   ├── config/                     # Config loading (env vars, YAML)
│   │   └── config.go
│   ├── database/                   # PostgreSQL connection pool & helpers
│   │   └── postgres.go
│   ├── kafka/                      # Kafka producer & consumer wrappers
│   │   ├── producer.go
│   │   └── consumer.go
│   ├── redis/                      # Redis client wrapper
│   │   └── redis.go
│   ├── middleware/                  # Auth, rate-limit, idempotency, logging
│   │   ├── auth.go
│   │   ├── idempotency.go
│   │   ├── ratelimit.go
│   │   └── logging.go
│   └── models/                     # Shared domain models & enums
│       ├── payment.go
│       ├── merchant.go
│       ├── ledger.go
│       └── enums.go
│
├── pkg/                            # Public utility packages (importable)
│   ├── money/                      # Money type — stores amount in smallest unit (paise/cents)
│   │   └── money.go
│   ├── uid/                        # UUID / ULID generator for IDs
│   │   └── uid.go
│   └── crypto/                     # AES-256-GCM encrypt/decrypt helpers
│       └── vault_crypto.go
│
├── services/                       # Each service = independently deployable binary
│   ├── gateway/                    # Payment Gateway API (the orchestrator)
│   │   ├── cmd/
│   │   │   └── main.go
│   │   ├── handler/
│   │   │   ├── payment_handler.go
│   │   │   ├── refund_handler.go
│   │   │   └── health_handler.go
│   │   ├── service/
│   │   │   └── payment_service.go  # Core orchestration logic
│   │   └── router.go
│   │
│   ├── vault/                      # Tokenization / Card Vault Service (CDE)
│   │   ├── cmd/
│   │   │   └── main.go
│   │   ├── handler/
│   │   │   └── token_handler.go
│   │   ├── service/
│   │   │   └── vault_service.go
│   │   └── router.go
│   │
│   ├── ledger/                     # Ledger / Accounting Service
│   │   ├── cmd/
│   │   │   └── main.go
│   │   ├── handler/
│   │   │   └── ledger_handler.go
│   │   ├── service/
│   │   │   └── ledger_service.go
│   │   └── router.go
│   │
│   ├── psp/                        # PSP / Acquirer Integration (Strategy Pattern)
│   │   ├── cmd/
│   │   │   └── main.go
│   │   ├── provider/
│   │   │   ├── provider.go         # Interface (Strategy)
│   │   │   ├── card_provider.go    # Card Network integration
│   │   │   ├── upi_provider.go     # UPI / NPCI integration
│   │   │   └── netbanking_provider.go
│   │   └── router.go
│   │
│   ├── webhook/                    # Webhook Dispatcher (Kafka Consumer)
│   │   ├── cmd/
│   │   │   └── main.go
│   │   └── dispatcher/
│   │       └── dispatcher.go
│   │
│   ├── settlement/                 # Payout & Settlement Cron Service
│   │   ├── cmd/
│   │   │   └── main.go
│   │   └── worker/
│   │       └── settlement_worker.go
│   │
│   └── risk/                       # Fraud & Risk Scoring Service
│       ├── cmd/
│       │   └── main.go
│       ├── handler/
│       │   └── risk_handler.go
│       ├── engine/
│       │   └── rules.go            # Rule-based fraud checks
│       └── router.go
│
└── tests/                          # Integration & E2E tests
    ├── payment_flow_test.go
    └── settlement_flow_test.go
```

> **Scalability by Design:** Need to add Wallets, EMI, or Pay Later in the future? Just add a new file under `services/psp/provider/` implementing the `PaymentProvider` interface. No existing code needs to change. This is the **Open/Closed Principle** in action.

---

## 3. Database Schema (Production-Grade)

All monetary amounts are stored in **smallest currency unit** (paise for INR, cents for USD) as `BIGINT` to eliminate floating-point errors. All IDs use `VARCHAR(26)` for ULID compatibility (sortable, unique, URL-safe).

### 3.1 Merchants

```sql
-- 000001_create_merchants.up.sql
CREATE TYPE merchant_status AS ENUM ('active', 'suspended', 'pending_verification');
CREATE TYPE payout_schedule AS ENUM ('daily', 'weekly', 'biweekly', 'monthly');

CREATE TABLE merchants (
    id                  VARCHAR(26) PRIMARY KEY,          -- ULID
    name                VARCHAR(255) NOT NULL,
    email               VARCHAR(255) NOT NULL UNIQUE,
    api_key_hash        VARCHAR(255) NOT NULL UNIQUE,     -- bcrypt hash of API key
    api_key_prefix      VARCHAR(8)   NOT NULL,            -- First 8 chars for identification (e.g., "sk_live_")
    status              merchant_status NOT NULL DEFAULT 'pending_verification',
    
    -- Settlement Configuration
    payout_schedule     payout_schedule NOT NULL DEFAULT 'daily',
    settlement_delay_days INT NOT NULL DEFAULT 2,         -- T+N days
    bank_account_number VARCHAR(255),                     -- Encrypted
    bank_ifsc_code      VARCHAR(11),
    bank_account_name   VARCHAR(255),
    
    -- Fees
    platform_fee_bps    INT NOT NULL DEFAULT 200,         -- 200 bps = 2.00% fee
    
    -- Webhook
    webhook_url         TEXT,
    webhook_secret_hash VARCHAR(255),                     -- For signing webhook payloads
    
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_merchants_api_key_prefix ON merchants(api_key_prefix);
CREATE INDEX idx_merchants_status ON merchants(status);
```

### 3.2 Payment Methods (Tokenized)

```sql
-- 000002_create_payment_methods.up.sql
CREATE TYPE pm_type AS ENUM ('card', 'upi', 'netbanking');
CREATE TYPE card_brand AS ENUM ('visa', 'mastercard', 'rupay', 'amex', 'unknown');

CREATE TABLE payment_methods (
    id                  VARCHAR(26) PRIMARY KEY,          -- Token ID (e.g., "pm_01HX...")
    customer_id         VARCHAR(26),                      -- Optional, for saved cards
    merchant_id         VARCHAR(26) NOT NULL REFERENCES merchants(id),
    type                pm_type NOT NULL,
    
    -- Card-specific fields (stored ONLY in vault, encrypted)
    card_last_four      VARCHAR(4),
    card_brand          card_brand,
    card_exp_month      SMALLINT,
    card_exp_year       SMALLINT,
    card_fingerprint    VARCHAR(64),                      -- Hash of full PAN for dedup
    encrypted_pan       BYTEA,                            -- AES-256-GCM encrypted PAN
    encryption_key_id   VARCHAR(64),                      -- Which KMS key version was used
    
    -- UPI-specific
    upi_vpa             VARCHAR(255),                     -- e.g., "user@upi"
    
    -- Netbanking-specific
    bank_code           VARCHAR(20),                      -- e.g., "HDFC", "SBI"
    
    is_reusable         BOOLEAN NOT NULL DEFAULT FALSE,   -- Can be charged again?
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pm_merchant ON payment_methods(merchant_id);
CREATE INDEX idx_pm_customer ON payment_methods(customer_id);
CREATE INDEX idx_pm_fingerprint ON payment_methods(card_fingerprint);
```

### 3.3 Payments (Core Transaction Table)

```sql
-- 000003_create_payments.up.sql
CREATE TYPE payment_status AS ENUM (
    'created',            -- Payment intent created
    'processing',         -- Sent to bank / PSP
    'requires_action',    -- Needs OTP / 3DS / UPI approve
    'authorized',         -- Bank approved, funds held
    'captured',           -- Money deducted from customer
    'failed',             -- Declined or error
    'canceled'            -- Merchant or system canceled
);

CREATE TYPE payment_method_type AS ENUM ('card', 'upi', 'netbanking');

CREATE TABLE payments (
    id                  VARCHAR(26) PRIMARY KEY,          -- "pay_01HX..."
    merchant_id         VARCHAR(26) NOT NULL REFERENCES merchants(id),
    payment_method_id   VARCHAR(26) REFERENCES payment_methods(id),
    payment_method_type payment_method_type NOT NULL,
    
    -- Money
    amount              BIGINT NOT NULL CHECK (amount > 0),   -- In paise/cents
    currency            VARCHAR(3) NOT NULL DEFAULT 'INR',
    platform_fee        BIGINT NOT NULL DEFAULT 0,            -- Fee charged (in paise)
    net_amount          BIGINT NOT NULL DEFAULT 0,            -- amount - platform_fee
    
    -- Status
    status              payment_status NOT NULL DEFAULT 'created',
    failure_reason      TEXT,
    
    -- Idempotency
    idempotency_key     VARCHAR(255) NOT NULL,
    
    -- PSP / Bank Response
    psp_transaction_id  VARCHAR(255),                     -- External bank reference
    psp_provider        VARCHAR(50),                      -- "visa", "npci_upi", "hdfc_nb"
    
    -- Metadata
    description         TEXT,
    metadata            JSONB DEFAULT '{}',               -- Merchant-defined key-value pairs
    
    -- Risk
    risk_score          SMALLINT,                         -- 0-100 fraud score
    
    -- Timestamps
    authorized_at       TIMESTAMPTZ,
    captured_at         TIMESTAMPTZ,
    failed_at           TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT uq_idempotency UNIQUE (merchant_id, idempotency_key)
);

-- Performance indexes
CREATE INDEX idx_payments_merchant_status ON payments(merchant_id, status);
CREATE INDEX idx_payments_merchant_created ON payments(merchant_id, created_at DESC);
CREATE INDEX idx_payments_status_captured ON payments(status, captured_at)
    WHERE status = 'captured';  -- Partial index for settlement queries
CREATE INDEX idx_payments_psp_txn ON payments(psp_transaction_id);
```

### 3.4 Ledger (Double-Entry Bookkeeping — Append-Only)

```sql
-- 000004_create_ledger.up.sql
CREATE TYPE entry_type AS ENUM ('debit', 'credit');
CREATE TYPE account_type AS ENUM (
    'merchant_payable',       -- Money owed TO the merchant
    'platform_revenue',       -- Our fee income
    'bank_transit',           -- Funds in-flight from/to bank
    'merchant_settlement',    -- Funds sent to merchant bank
    'refund_payable'          -- Refunds owed back to customer
);

CREATE TABLE accounts (
    id                  VARCHAR(26) PRIMARY KEY,
    merchant_id         VARCHAR(26) REFERENCES merchants(id),
    type                account_type NOT NULL,
    currency            VARCHAR(3) NOT NULL DEFAULT 'INR',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT uq_merchant_account UNIQUE (merchant_id, type, currency)
);

CREATE TABLE ledger_entries (
    id                  VARCHAR(26) PRIMARY KEY,
    transaction_group_id VARCHAR(26) NOT NULL,            -- Groups debit+credit pair
    account_id          VARCHAR(26) NOT NULL REFERENCES accounts(id),
    payment_id          VARCHAR(26) REFERENCES payments(id),
    
    entry_type          entry_type NOT NULL,
    amount              BIGINT NOT NULL CHECK (amount > 0),  -- Always positive
    currency            VARCHAR(3) NOT NULL DEFAULT 'INR',
    
    description         TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ledger is append-only: NO UPDATE or DELETE should ever happen
-- Balance = SUM(credit) - SUM(debit) for an account

CREATE INDEX idx_ledger_account ON ledger_entries(account_id, created_at DESC);
CREATE INDEX idx_ledger_txn_group ON ledger_entries(transaction_group_id);
CREATE INDEX idx_ledger_payment ON ledger_entries(payment_id);

-- Materialized view for fast balance lookups
CREATE MATERIALIZED VIEW account_balances AS
SELECT
    account_id,
    SUM(CASE WHEN entry_type = 'credit' THEN amount ELSE 0 END) AS total_credits,
    SUM(CASE WHEN entry_type = 'debit'  THEN amount ELSE 0 END) AS total_debits,
    SUM(CASE WHEN entry_type = 'credit' THEN amount ELSE -amount END) AS balance
FROM ledger_entries
GROUP BY account_id;

CREATE UNIQUE INDEX idx_ab_account ON account_balances(account_id);
```

### 3.5 Refunds

```sql
-- 000005_create_refunds.up.sql
CREATE TYPE refund_status AS ENUM ('pending', 'processing', 'succeeded', 'failed');

CREATE TABLE refunds (
    id                  VARCHAR(26) PRIMARY KEY,
    payment_id          VARCHAR(26) NOT NULL REFERENCES payments(id),
    merchant_id         VARCHAR(26) NOT NULL REFERENCES merchants(id),
    
    amount              BIGINT NOT NULL CHECK (amount > 0),
    currency            VARCHAR(3) NOT NULL DEFAULT 'INR',
    reason              TEXT,
    status              refund_status NOT NULL DEFAULT 'pending',
    
    idempotency_key     VARCHAR(255) NOT NULL,
    psp_refund_id       VARCHAR(255),
    
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT uq_refund_idempotency UNIQUE (merchant_id, idempotency_key)
);

CREATE INDEX idx_refunds_payment ON refunds(payment_id);
CREATE INDEX idx_refunds_merchant ON refunds(merchant_id, created_at DESC);
```

### 3.6 Settlements / Payouts

```sql
-- 000006_create_settlements.up.sql
CREATE TYPE settlement_status AS ENUM (
    'pending',          -- Aggregated, waiting for bank transfer
    'processing',       -- Bank API called
    'paid',             -- Bank confirmed transfer
    'failed',           -- Bank rejected
    'reversed'          -- Returned after being paid
);

CREATE TABLE settlements (
    id                  VARCHAR(26) PRIMARY KEY,
    merchant_id         VARCHAR(26) NOT NULL REFERENCES merchants(id),
    
    -- Money
    gross_amount        BIGINT NOT NULL,       -- Total payments in this settlement window
    total_fees          BIGINT NOT NULL,       -- Platform fees deducted
    total_refunds       BIGINT NOT NULL,       -- Refunds deducted
    net_amount          BIGINT NOT NULL,       -- What merchant actually receives
    currency            VARCHAR(3) NOT NULL DEFAULT 'INR',
    
    -- Window
    period_start        TIMESTAMPTZ NOT NULL,  -- Settlement window start
    period_end          TIMESTAMPTZ NOT NULL,  -- Settlement window end
    
    -- Status
    status              settlement_status NOT NULL DEFAULT 'pending',
    bank_reference_id   VARCHAR(255),          -- UTR / bank transaction ref
    failure_reason      TEXT,
    
    paid_at             TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Links individual payments to a settlement batch
CREATE TABLE settlement_items (
    id                  VARCHAR(26) PRIMARY KEY,
    settlement_id       VARCHAR(26) NOT NULL REFERENCES settlements(id),
    payment_id          VARCHAR(26) NOT NULL REFERENCES payments(id),
    amount              BIGINT NOT NULL,
    fee                 BIGINT NOT NULL,
    net_amount          BIGINT NOT NULL,
    
    CONSTRAINT uq_settlement_payment UNIQUE (payment_id)  -- A payment can only be in one settlement
);

CREATE INDEX idx_settlements_merchant ON settlements(merchant_id, created_at DESC);
CREATE INDEX idx_settlements_status ON settlements(status) WHERE status = 'pending';
CREATE INDEX idx_settlement_items_settlement ON settlement_items(settlement_id);
```

### 3.7 Webhook Delivery Log

```sql
-- 000007_create_webhook_deliveries.up.sql
CREATE TYPE delivery_status AS ENUM ('pending', 'delivered', 'failed', 'exhausted');

CREATE TABLE webhook_deliveries (
    id                  VARCHAR(26) PRIMARY KEY,
    merchant_id         VARCHAR(26) NOT NULL REFERENCES merchants(id),
    
    event_type          VARCHAR(100) NOT NULL,             -- "payment_intent.succeeded"
    payload             JSONB NOT NULL,
    
    endpoint_url        TEXT NOT NULL,
    
    status              delivery_status NOT NULL DEFAULT 'pending',
    attempts            SMALLINT NOT NULL DEFAULT 0,
    max_attempts        SMALLINT NOT NULL DEFAULT 5,
    last_response_code  SMALLINT,                          -- HTTP status from merchant
    last_error          TEXT,
    next_retry_at       TIMESTAMPTZ,
    
    delivered_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_retry ON webhook_deliveries(status, next_retry_at)
    WHERE status = 'pending' OR status = 'failed';
CREATE INDEX idx_webhook_merchant ON webhook_deliveries(merchant_id, created_at DESC);
```

### 3.8 Idempotency Keys (Redis-backed, with DB fallback)

```sql
-- 000008_create_idempotency_keys.up.sql
CREATE TABLE idempotency_keys (
    key                 VARCHAR(255) NOT NULL,
    merchant_id         VARCHAR(26) NOT NULL,
    
    request_path        VARCHAR(255) NOT NULL,             -- "/v1/payments"
    request_method      VARCHAR(10) NOT NULL,              -- "POST"
    
    response_code       SMALLINT,
    response_body       JSONB,
    
    status              VARCHAR(20) NOT NULL DEFAULT 'in_progress',  -- 'in_progress', 'complete'
    
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '24 hours',
    
    PRIMARY KEY (merchant_id, key)
);
```

---

## 4. Kafka Topics & Event Schema

All events follow a consistent envelope:

```json
{
  "event_id": "evt_01HX...",
  "event_type": "payment_intent.succeeded",
  "created_at": "2026-09-01T10:00:00Z",
  "data": { ... }
}
```

| Topic | Producer | Consumer(s) | Events |
|---|---|---|---|
| `payment.events` | Gateway Service | Webhook, Reporting, Risk | `payment_intent.created`, `payment_intent.processing`, `payment_intent.requires_action`, `payment_intent.succeeded`, `payment_intent.failed`, `payment_intent.canceled` |
| `refund.events` | Gateway Service | Webhook, Reporting, Ledger | `charge.refunded`, `charge.refund_failed` |
| `settlement.events` | Settlement Worker | Webhook, Reporting | `payout.created`, `payout.paid`, `payout.failed` |
| `webhook.dlq` | Webhook Dispatcher | Alert Service | Dead-letter queue — failed deliveries after max retries |
| `risk.events` | Risk Service | Reporting, Analytics | `risk.flagged`, `risk.blocked` |

---

## 5. Core Interfaces (Strategy Pattern for Extensibility)

```go
// provider.go — Any new payment method implements this interface

type PaymentProvider interface {
    // Authorize checks if the payment can proceed (funds available, card valid, etc.)
    Authorize(ctx context.Context, req AuthorizeRequest) (*AuthorizeResponse, error)

    // Capture actually deducts the money after authorization
    Capture(ctx context.Context, authID string, amount int64) (*CaptureResponse, error)

    // Refund returns money to the customer
    Refund(ctx context.Context, req RefundRequest) (*RefundResponse, error)

    // MethodType returns the payment method type this provider handles
    MethodType() PaymentMethodType
}
```

> **Adding Wallets, EMI, or BNPL in the future?** Just create `wallet_provider.go` implementing `PaymentProvider`. Register it in the routing engine. Zero changes to existing providers.

---

## 6. Implementation Phases

### Phase 1 — Foundation & Core Payment Flow *(Week 1–2)*
- [ ] Project scaffolding: Go modules, Docker Compose (Postgres, Redis, Kafka), Makefile
- [ ] Database migrations (all tables above)
- [ ] `internal/` packages: config, database pool, Redis client, Kafka producer
- [ ] `pkg/money`, `pkg/uid`, `pkg/crypto` utility packages
- [ ] **Vault Service**: Tokenize card → return `pm_xxx` token
- [ ] **Gateway Service**: Create Payment → Idempotency check → Call Vault → Store in DB
- [ ] Middleware: API key auth, idempotency, request logging

### Phase 2 — PSP Integration & Ledger *(Week 3–4)*
- [ ] **PSP Service**: Strategy pattern with `CardProvider`, `UPIProvider`, `NetbankingProvider`
- [ ] Mock bank simulator for local development (always approve / configurable decline)
- [ ] **Ledger Service**: Double-entry append for every payment capture
- [ ] Payment state machine: `created → processing → authorized → captured`
- [ ] Refunds: Create refund → reverse ledger entries → call PSP refund

### Phase 3 — Async Events & Webhooks *(Week 5)*
- [ ] Kafka producer integration in Gateway (publish on every status change)
- [ ] **Webhook Dispatcher**: Kafka consumer → HTTP POST with HMAC signature
- [ ] Retry logic with exponential backoff (1s, 4s, 16s, 64s, 256s)
- [ ] Dead-letter queue (`webhook.dlq`) for exhausted retries
- [ ] Webhook delivery logging in `webhook_deliveries` table

### Phase 4 — Settlements & Payouts *(Week 6)*
- [ ] **Settlement Worker**: Cron-based aggregation of captured payments past T+N window
- [ ] Settlement batch creation → link payments via `settlement_items`
- [ ] Ledger entries for payout (debit merchant_payable, credit bank_transit)
- [ ] Mock bank payout API
- [ ] Settlement status updates → Kafka events

### Phase 5 — Risk, Reporting & Dashboards *(Week 7–8)*
- [ ] **Risk Service**: Rule-based scoring (velocity checks, amount thresholds, geo checks)
- [ ] Kafka consumer for CQRS → write to Elasticsearch / ClickHouse
- [ ] REST APIs for Merchant Dashboard (list payments, search, export CSV)
- [ ] REST APIs for Settlement Dashboard (list settlements, status tracking)

---

## 7. Verification Plan

### Automated Tests
```bash
# Unit tests (every package)
make test-unit

# Integration tests (spin up Postgres + Redis + Kafka via testcontainers)
make test-integration

# Full E2E: Create merchant → Tokenize card → Make payment → Verify webhook → Run settlement
make test-e2e
```

### Manual Verification
- Simple HTML checkout page to test card tokenization flow visually
- Trigger settlement cron manually and verify ledger balances sum to zero
- Simulate webhook failures and verify retry + DLQ behavior in Kafka UI

---

## 8. Docker Compose (Local Development)

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:16-alpine
    ports: ["5432:5432"]
    environment:
      POSTGRES_DB: payment_processor
      POSTGRES_USER: pp_user
      POSTGRES_PASSWORD: pp_secret

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  kafka:
    image: bitnami/kafka:3.7
    ports: ["9092:9092"]
    environment:
      KAFKA_CFG_NODE_ID: 1
      KAFKA_CFG_PROCESS_ROLES: broker,controller
      KAFKA_CFG_CONTROLLER_QUORUM_VOTERS: 1@kafka:9093
      KAFKA_CFG_LISTENERS: PLAINTEXT://:9092,CONTROLLER://:9093
      KAFKA_CFG_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_CFG_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP: CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT

  kafka-ui:
    image: provectuslabs/kafka-ui:latest
    ports: ["8080:8080"]
    environment:
      KAFKA_CLUSTERS_0_NAME: local
      KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS: kafka:9092
```

```bash
# Start everything
docker-compose up -d

# Run migrations
make migrate-up

# Start gateway service
make run-gateway
```
