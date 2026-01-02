# Movra - Cross-Border Payment Platform

## Architecture Design Document

**Date:** 2026-01-02
**Status:** Approved
**Author:** Design Session

---

## 1. Overview

Movra is a cross-border payment platform similar to Wise/Remitly. Users send money from one country/currency to another, with real-time exchange rates, multiple funding methods, and multiple payout options.

### 1.1 Core User Flow

```
1. User signs up → completes KYC verification
2. User enters: "Send $500 SGD to Philippines"
3. System shows: Exchange rate, fees, recipient gets ₱19,850
4. User adds recipient (bank account, mobile wallet, or cash pickup)
5. User funds transfer (bank transfer, card, or wallet balance)
6. User confirms → System processes
7. System converts currency, executes payout
8. User receives notification: "Transfer complete"
```

### 1.2 Business Goals

- Support multiple funding methods (bank transfer, card, wallet)
- Support multiple payout methods (bank deposit, mobile wallet, cash pickup)
- Currency-agnostic design supporting any corridor
- Real-time exchange rates with rate locking
- Regulatory compliance and audit trails

---

## 2. Tech Stack Summary

| Component | Technology | Justification |
|-----------|------------|---------------|
| API Gateway | Node.js + Express | Async I/O, high concurrency for routing |
| Auth Service | C#/.NET 8 | Enterprise auth patterns, strong typing for security |
| Payment Service | Java 21 + Spring Boot 3 | Complex business logic, mature transaction support |
| Exchange Rate Service | Go + Gin | High-throughput, low-latency, concurrent operations |
| Settlement Service | Go + Gin | Batch processing, parallel payout execution |
| Notification Service | Node.js + Express | I/O bound, event-driven |
| Frontend | React + TypeScript | Industry standard, SG market expectation |
| Databases | PostgreSQL (per service) | ACID compliance for financial data |
| Cache | Redis | Rate caching, distributed locks |
| Message Queue | Apache Kafka | Event-driven async communication |
| Observability | Prometheus, Grafana, Loki, Jaeger | Metrics, dashboards, logs, tracing |
| Container Orchestration | Kubernetes (prod), Docker Compose (dev) | Industry standard |

---

## 3. Service Architecture

### 3.1 Service Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         React Frontend                               │
│                    (User Dashboard + Admin)                          │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ REST/HTTPS
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     API Gateway (Node.js)                            │
│              Rate Limiting, Routing, JWT Verification                │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ gRPC + mTLS
        ┌───────────────────────┼───────────────────────┐
        ▼                       ▼                       ▼
┌───────────────┐       ┌───────────────┐       ┌───────────────┐
│ Auth Service  │       │Payment Service│       │ Exchange Rate │
│   (C#/.NET)   │       │(Java/Spring)  │       │ Service (Go)  │
│               │       │               │       │               │
│ - Registration│       │ - Transfers   │       │ - FX Rates    │
│ - Login/JWT   │       │ - State Mgmt  │       │ - Rate Lock   │
│ - KYC Levels  │       │ - Idempotency │       │ - Caching     │
│ - MFA         │       │ - Fraud Check │       │               │
└───────┬───────┘       └───────┬───────┘       └───────┬───────┘
        │                       │                       │
        │ PostgreSQL            │ PostgreSQL            │ Redis
        ▼                       ▼                       ▼
   [Auth DB]              [Payment DB]            [Rate Cache]
                                │
                                │ Kafka Events
                                ▼
                    ┌───────────────────────┐
                    │       Kafka           │
                    │   (Event Bus)         │
                    └───────────┬───────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        ▼                       ▼                       ▼
┌───────────────┐       ┌───────────────┐       ┌───────────────┐
│  Settlement   │       │ Notification  │       │ Audit Consumer│
│ Service (Go)  │       │Service (Node) │       │   (Kafka)     │
│               │       │               │       │               │
│ - Bank Payout │       │ - Email       │       │ - Event Store │
│ - Wallet Pay  │       │ - SMS         │       │ - Compliance  │
│ - Cash Pickup │       │ - Webhooks    │       │               │
└───────┬───────┘       └───────┬───────┘       └───────┬───────┘
        │                       │                       │
        │ PostgreSQL            │ PostgreSQL + Redis    │ PostgreSQL
        ▼                       ▼                       ▼
  [Settlement DB]         [Notification DB]        [Audit DB]
```

### 3.2 Service Details

#### API Gateway (Node.js + Express)
**Responsibility:** Single entry point for all client requests

- Request routing to backend services
- Rate limiting (per user, per IP)
- JWT verification (delegates to Auth Service)
- Request/response logging
- CORS handling

**Database:** None (stateless)

---

#### Auth Service (C#/.NET 8)
**Responsibility:** Identity and access management

- User registration and login
- JWT token issuance and refresh
- KYC level management (affects transfer limits)
- Multi-factor authentication
- Session management
- Password reset flows

**Database:** PostgreSQL
- Users table
- KYC records
- Sessions
- MFA secrets

---

#### Payment Service (Java 21 + Spring Boot 3)
**Responsibility:** Core transaction logic

- Transfer creation with idempotency keys
- Transaction state machine management
- Fraud checks (velocity, amount limits)
- Rate locking coordination
- Fee calculation
- Funding method handling (bank, card, wallet)

**Database:** PostgreSQL
- Transfers table
- Transfer state history
- Idempotency keys
- Fee configurations

**State Machine:**
```
CREATED → AWAITING_FUNDS → FUNDS_RECEIVED → CONVERTING →
CONVERTED → PAYOUT_PENDING → PAYOUT_PROCESSING →
COMPLETED | FAILED | REFUNDED
```

---

#### Exchange Rate Service (Go + Gin)
**Responsibility:** FX rate management

- Real-time rate fetching (simulated + external API option)
- Rate caching with TTL
- Rate locking (hold rate for N seconds during user confirmation)
- Margin/spread calculation
- Multi-corridor support

**Database:** Redis (primary), PostgreSQL (rate history)
- Cached rates with TTL
- Locked rates with expiry
- Historical rates for audit

---

#### Settlement Service (Go + Gin)
**Responsibility:** Payout execution

- Bank deposit processing
- Mobile wallet integration (GCash, GoPay simulation)
- Cash pickup code generation
- Batch processing for efficiency
- Retry logic for failed payouts
- Reconciliation

**Database:** PostgreSQL
- Payout records
- Batch jobs
- Reconciliation logs

---

#### Notification Service (Node.js + Express)
**Responsibility:** User communication

- Email notifications (SendGrid/simulation)
- SMS notifications (Twilio/simulation)
- Push notifications
- Webhook delivery to external systems
- Delivery tracking and retry

**Database:** PostgreSQL + Redis
- Notification logs
- Delivery status
- Redis for job queue

---

## 4. Data Architecture

### 4.1 Database per Service

Each service owns its database. No direct database access between services.

| Service | Database | Purpose |
|---------|----------|---------|
| Auth Service | auth_db (PostgreSQL) | Users, KYC, sessions |
| Payment Service | payment_db (PostgreSQL) | Transfers, state, idempotency |
| Exchange Rate Service | Redis + rate_db (PostgreSQL) | Cached rates, rate history |
| Settlement Service | settlement_db (PostgreSQL) | Payouts, batches |
| Notification Service | notification_db (PostgreSQL) | Delivery logs |
| Audit Consumer | audit_db (PostgreSQL) | All events for compliance |

### 4.2 Key Data Models

#### User (Auth Service)
```
users
├── id (UUID)
├── email
├── phone
├── password_hash
├── kyc_level (NONE, BASIC, VERIFIED, PREMIUM)
├── created_at
└── updated_at
```

#### Transfer (Payment Service)
```
transfers
├── id (UUID)
├── idempotency_key
├── user_id
├── status (enum)
├── source_currency
├── source_amount
├── target_currency
├── target_amount
├── exchange_rate
├── rate_lock_id
├── fee_amount
├── funding_method (BANK_TRANSFER, CARD, WALLET)
├── recipient_id
├── created_at
└── updated_at
```

#### Recipient (Payment Service)
```
recipients
├── id (UUID)
├── user_id
├── type (BANK_ACCOUNT, MOBILE_WALLET, CASH_PICKUP)
├── country
├── currency
├── details (JSONB - account number, wallet ID, etc.)
├── created_at
└── updated_at
```

---

## 5. Communication Patterns

### 5.1 Synchronous (gRPC)

Used when immediate response is required.

| Caller | Callee | Method | Purpose |
|--------|--------|--------|---------|
| Gateway | Auth | VerifyToken | Validate JWT |
| Gateway | Auth | RefreshToken | Token refresh |
| Gateway | Payment | CreateTransfer | New transfer |
| Gateway | Payment | GetTransfer | Transfer status |
| Gateway | Payment | ConfirmTransfer | User confirmation |
| Gateway | ExchangeRate | GetRate | Display rate |
| Payment | ExchangeRate | LockRate | Lock rate for confirmation |
| Payment | ExchangeRate | GetLockedRate | Retrieve locked rate |
| Payment | Auth | GetUserKYCLevel | Check transfer limits |

### 5.2 Asynchronous (Kafka Events)

Used for decoupled, eventually consistent operations.

| Event | Publisher | Consumers |
|-------|-----------|-----------|
| TransferInitiated | Payment | Notification, Audit |
| FundsReceived | Payment | Settlement, Notification, Audit |
| RateLocked | ExchangeRate | Payment, Audit |
| RateExpired | ExchangeRate | Payment |
| PayoutProcessing | Settlement | Notification, Audit |
| PayoutCompleted | Settlement | Payment, Notification, Audit |
| PayoutFailed | Settlement | Payment, Notification, Audit |
| UserVerified | Auth | Payment, Audit |
| FraudFlagged | Payment | Notification, Audit |

### 5.3 Kafka Topics

```
movra.transfers.initiated
movra.transfers.funds-received
movra.transfers.completed
movra.transfers.failed
movra.rates.locked
movra.rates.expired
movra.payouts.processing
movra.payouts.completed
movra.payouts.failed
movra.users.verified
movra.fraud.flagged
```

---

## 6. Security

### 6.1 External Communication
- HTTPS/TLS for all client-facing endpoints
- JWT tokens for user authentication
- Rate limiting at API Gateway

### 6.2 Internal Communication
- mTLS between all services
- Each service has its own certificate
- Certificates signed by internal CA
- cert-manager for automatic rotation in K8s

### 6.3 User Context Propagation
```
gRPC Metadata:
- x-user-id: User identifier
- x-correlation-id: Request trace ID
- x-kyc-level: User's verification level
```

---

## 7. Observability

### 7.1 Stack

| Component | Tool | Purpose |
|-----------|------|---------|
| Metrics | Prometheus | Service metrics, latency, error rates |
| Dashboards | Grafana | Visualization |
| Logs | Loki | Centralized logging |
| Tracing | Jaeger | Distributed request tracing |

### 7.2 Key Metrics

- Request latency (p50, p95, p99)
- Error rates by service and endpoint
- Transfer success/failure rates
- Kafka consumer lag
- Database connection pool usage
- Rate lock expiry rates

### 7.3 Tracing

All requests carry a correlation ID from Gateway through all services. Jaeger collects spans for end-to-end visibility.

---

## 8. Deployment

### 8.1 Local Development
- Docker Compose
- All services + infrastructure in containers
- Hot reload for development

### 8.2 Production (Kubernetes)
- Helm charts per service
- Horizontal Pod Autoscaler
- Resource limits and requests
- Liveness and readiness probes
- cert-manager for TLS

### 8.3 Infrastructure Components

```
Kubernetes Cluster
├── Namespace: movra
│   ├── Deployments
│   │   ├── api-gateway
│   │   ├── auth-service
│   │   ├── payment-service
│   │   ├── exchange-rate-service
│   │   ├── settlement-service
│   │   └── notification-service
│   ├── StatefulSets
│   │   ├── postgresql-auth
│   │   ├── postgresql-payment
│   │   ├── postgresql-settlement
│   │   ├── postgresql-notification
│   │   ├── postgresql-audit
│   │   ├── redis
│   │   └── kafka
│   └── Services, ConfigMaps, Secrets
└── Namespace: observability
    ├── prometheus
    ├── grafana
    ├── loki
    └── jaeger
```

---

## 9. Currencies and Corridors

### 9.1 Design Principle

Currency-agnostic system. Corridors configured via database/config, not code.

### 9.2 Initial Corridors

| Corridor | Source | Destination | Notes |
|----------|--------|-------------|-------|
| SGD → PHP | Singapore | Philippines | Primary corridor |
| SGD → INR | Singapore | India | High remittance volume |
| SGD → IDR | Singapore | Indonesia | Regional |
| USD → PHP | United States | Philippines | Alternative |
| SGD → USD | Singapore | United States | |

### 9.3 Corridor Configuration

```yaml
corridors:
  - source: SGD
    target: PHP
    enabled: true
    fee_percentage: 0.5
    fee_minimum: 3.00
    margin_percentage: 0.3
    payout_methods:
      - BANK_ACCOUNT
      - MOBILE_WALLET
      - CASH_PICKUP
```

---

## 10. Funding Methods

### 10.1 Bank Transfer
- User sends money to Movra's bank account
- Reference number for matching
- 1-3 day processing
- Lowest fees

### 10.2 Card Payment
- Instant funding
- Higher fees (2-3%)
- 3D Secure verification
- Simulated via adapter pattern

### 10.3 Wallet Balance
- Pre-funded account
- Instant transfer initiation
- Requires top-up flow

---

## 11. Payout Methods

### 11.1 Bank Deposit
- Direct to recipient's bank account
- Account validation required
- 1-2 day processing

### 11.2 Mobile Wallet
- GCash (Philippines)
- GoPay (Indonesia)
- Instant delivery
- Mobile number as identifier

### 11.3 Cash Pickup
- Generate pickup code
- Recipient visits partner location
- Shows ID + code
- Instant availability

---

## 12. Error Handling and Resilience

### 12.1 Retry Strategy
- Exponential backoff for transient failures
- Dead letter queue for permanent failures
- Manual review queue for edge cases

### 12.2 Circuit Breakers
- Between all service-to-service calls
- Fail fast when downstream is unhealthy
- Graceful degradation

### 12.3 Idempotency
- All transfer operations use idempotency keys
- Duplicate requests return same result
- Keys stored for 24 hours minimum

---

## 13. Testing Strategy

### 13.1 Unit Tests
- Per service, mock external dependencies
- Focus on business logic

### 13.2 Integration Tests
- Test service with real database
- Test Kafka producers/consumers

### 13.3 Contract Tests
- gRPC contract verification
- Proto file compatibility

### 13.4 End-to-End Tests
- Full flow in Docker Compose environment
- Simulated user journeys

---

## 14. Implementation Order

### Phase 1: Foundation
1. Project structure and build setup
2. Docker Compose with infrastructure (PostgreSQL, Redis, Kafka)
3. Proto files and code generation
4. Basic API Gateway with routing

### Phase 2: Core Services
5. Auth Service (registration, login, JWT)
6. Exchange Rate Service (rates, caching)
7. Payment Service (create transfer, state machine)

### Phase 3: Completion
8. Settlement Service (payouts)
9. Notification Service
10. Kafka event flows

### Phase 4: Frontend
11. React app with transfer flow
12. Admin dashboard

### Phase 5: Production Readiness
13. Kubernetes deployment
14. Observability stack
15. mTLS setup

---

## 15. Repository Structure

```
movra/
├── docs/
│   └── plans/
│       └── 2026-01-02-movra-architecture-design.md
├── proto/
│   ├── auth.proto
│   ├── payment.proto
│   ├── exchange.proto
│   └── settlement.proto
├── services/
│   ├── api-gateway/          (Node.js)
│   ├── auth-service/         (C#/.NET)
│   ├── payment-service/      (Java/Spring)
│   ├── exchange-rate-service/ (Go)
│   ├── settlement-service/   (Go)
│   └── notification-service/ (Node.js)
├── frontend/                 (React + TypeScript)
├── infrastructure/
│   ├── docker/
│   │   └── docker-compose.yml
│   └── k8s/
│       └── helm/
│           └── movra/
├── scripts/
├── .gitignore
└── README.md
```

---

## Appendix A: Interview Talking Points

After building this project, you can answer:

1. **"Why microservices?"** — I experienced the coordination complexity firsthand. For a payment system, independent deployability matters because...

2. **"Why Kafka vs direct calls?"** — User-facing operations are sync for immediate feedback. Settlement happens async because users don't stare at the screen waiting for bank transfers.

3. **"How do you handle duplicate payments?"** — Idempotency keys. Every transfer creation includes a client-generated key. I store it and return the same response for duplicates.

4. **"What happens if the exchange rate expires mid-transaction?"** — Rate locks have a TTL. If expired, we re-fetch, show user the new rate, require re-confirmation.

5. **"How would you scale this?"** — Payment Service is stateless, horizontal scaling. Database read replicas. Exchange Rate Service caches in Redis, can handle 10x load without DB hits.

6. **"Why separate databases?"** — Failure isolation. If settlement DB has issues, payments can still be created. Services degrade gracefully.
