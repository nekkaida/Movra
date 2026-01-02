# Movra - Cross-Border Payment Platform

A full-stack cross-border payment platform built with microservices architecture. Similar to Wise/Remitly, enabling users to send money internationally with real-time exchange rates, multiple funding methods, and various payout options.

## Architecture Overview

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   React     │────▶│ API Gateway │────▶│   Auth      │
│  Frontend   │     │  (Node.js)  │     │  (C#/.NET)  │
└─────────────┘     └─────────────┘     └─────────────┘
                           │
       ┌───────────────────┼───────────────────┐
       ▼                   ▼                   ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Payment    │     │  Exchange   │     │ Settlement  │
│   (Java)    │     │ Rate (Go)   │     │    (Go)     │
└─────────────┘     └─────────────┘     └─────────────┘
       │                   │                   │
       └───────────────────┼───────────────────┘
                           ▼
                    ┌─────────────┐
                    │   Kafka     │
                    └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │Notification │
                    │  (Node.js)  │
                    └─────────────┘
```

## Tech Stack

| Service | Technology |
|---------|------------|
| API Gateway | Node.js + Express |
| Auth Service | C#/.NET 8 |
| Payment Service | Java 21 + Spring Boot 3 |
| Exchange Rate Service | Go + Gin |
| Settlement Service | Go + Gin |
| Notification Service | Node.js + Express |
| Frontend | React + TypeScript |
| Databases | PostgreSQL (per service) |
| Cache | Redis |
| Message Queue | Apache Kafka |
| Observability | Prometheus, Grafana, Loki, Jaeger |

## Features

### Funding Methods
- Bank Transfer
- Card Payment
- Wallet Balance

### Payout Methods
- Bank Deposit
- Mobile Wallet (GCash, GoPay)
- Cash Pickup

### Supported Corridors
- SGD → PHP (Singapore to Philippines)
- SGD → INR (Singapore to India)
- SGD → IDR (Singapore to Indonesia)
- USD → PHP (US to Philippines)
- SGD → USD (Singapore to US)

## Getting Started

### Prerequisites
- Docker & Docker Compose
- Node.js 20+
- Go 1.21+
- Java 21+
- .NET 8 SDK

### Local Development

```bash
# Start infrastructure (PostgreSQL, Redis, Kafka)
docker-compose -f infrastructure/docker/docker-compose.yml up -d

# Start services (in separate terminals or use the provided script)
./scripts/start-all.sh
```

### Running with Kubernetes

```bash
# Create namespace
kubectl create namespace movra

# Install with Helm
helm install movra infrastructure/k8s/helm/movra -n movra
```

## Project Structure

```
movra/
├── docs/                  # Documentation
├── proto/                 # gRPC protocol buffer definitions
├── services/
│   ├── api-gateway/       # Node.js API Gateway
│   ├── auth-service/      # C#/.NET Auth Service
│   ├── payment-service/   # Java/Spring Payment Service
│   ├── exchange-rate-service/  # Go Exchange Rate Service
│   ├── settlement-service/     # Go Settlement Service
│   └── notification-service/   # Node.js Notification Service
├── frontend/              # React + TypeScript
├── infrastructure/
│   ├── docker/           # Docker Compose files
│   └── k8s/              # Kubernetes/Helm charts
└── scripts/              # Utility scripts
```

## Documentation

- [Architecture Design](docs/plans/2026-01-02-movra-architecture-design.md)

## License

MIT
