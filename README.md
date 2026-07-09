# Club Membership Management Platform

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://go.dev/)
[![Gin](https://img.shields.io/badge/Gin-1.12-00ADD8.svg)](https://gin-gonic.com/)
[![Angular](https://img.shields.io/badge/Angular-Material-red.svg)](https://angular.io/)
[![Compose Multiplatform](https://img.shields.io/badge/Compose-Multiplatform-blue.svg)](https://www.jetbrains.com/lp/compose-multiplatform/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-Ready-blue.svg)](https://kubernetes.io/)

## Overview

This application is a comprehensive platform designed for managing an exclusive recreational club. It facilitates dynamic pricing for membership tiers (Garden, Club, Patron), member administration, and provides a public-facing portal where prospective members can view benefits and receive AI-driven recommendations. The system features role-based access control, directing Managers to the administration dashboard and Members to social features like the exclusive chat.

## Key Features

- **Role-Based Access:** Secure routing based on user roles—Managers access the Dashboard, while Members access the Chat and Portal.
- **Management Dashboard:** A centralized hub for managers to administer club members (CRUD) and adjust membership pricing tiers.
- **Public Portal:** A user-friendly landing page displaying membership choices (Garden Pass, Club Membership, Patron Membership).
- **AI Recommendation Assistant:** Uses **Google Gemini (RAG + Weaviate)** to help users select the best plan based on natural language input.
- **Dynamic Pricing:** Real-time updates to service descriptions and values via RabbitMQ event propagation.
- **Secure Authentication:** Robust auth flow using Keycloak, OAuth2, OIDC, and JWT with JWKS-based validation.
- **Event-Driven Architecture:** RabbitMQ for price updates, Kafka for member request intake with deduplication.
- **Resilient Gateway:** Custom reverse proxy with per-route circuit breakers and per-user rate limiting.
- **Full Observability:** OpenTelemetry traces and metrics exported to Datadog.

## Architecture & Microservices

The system is built on a Microservices architecture using **Go** with platform-native DNS for service discovery.

| Service Name | Description |
| :--- | :--- |
| **`service-app-gateway`** | Entry point handling routing, JWT authentication, rate limiting, circuit breakers, and load balancing. |
| **`member-service`** | Handles member data persistence, CRUD operations, price caching via Redis, and Kafka consumer for requests. |
| **`pricing-service`** | Manages the 3 pricing tiers (Garden, Club, Patron) with MongoDB and publishes updates via RabbitMQ. |
| **`member-request-service`** | Ingests member requests with Redis deduplication and forwards them to the member service via Kafka. |
| **`recommendation-service`** | **AI-powered** RAG service using Google Gemini and Weaviate for plan recommendations. |

## Tech Stack

### Backend
- **Core:** Go 1.25+, Gin (HTTP framework), GORM (ORM for PostgreSQL)
- **Databases:** PostgreSQL (member-service), MongoDB (pricing-service), Redis (caching, rate limiting, dedup)
- **Messaging:** RabbitMQ (price update events), Kafka (member request events)
- **Security:** Keycloak, OAuth2, OIDC, JWT (JWKS-based RSA validation), role-based access control
- **AI:** Google Gemini (chat + embeddings), Weaviate (vector database), RAG pipeline
- **Observability:** OpenTelemetry (traces + metrics), Datadog
- **Testing:** Go `testing` package, testify (assert + mock), httptest

### [Frontend (Web)](https://github.com/Eslicdm/service-app-angular)
- **Framework:** Angular
- **UI:** Angular Material, Tailwind
- **Testing:** Jasmine, Karma, Angular Testing Library, Cypress

### [Frontend (Multiplatform - Mobile/Web/Desktop)](https://github.com/Eslicdm/ServiceAppCompose)
- **Framework:** Jetpack Compose Multiplatform
- **Libraries:** Ktor Client, SQLDelight, Koin

### DevOps & Infrastructure
- **Containerization:** Docker (multi-stage builds), Kubernetes (Kind for local)
- **CI/CD:** GitHub Actions
- **Observability:** OpenTelemetry, Datadog

## How to Run with Docker Compose

This project uses Docker Compose to orchestrate the microservices, databases, and infrastructure components (Keycloak, Kafka, Redis, RabbitMQ, Weaviate, etc.).

### 1. Prerequisites

- **Docker Desktop** installed and running.

### 2. Configuration Setup

#### A. Environment Variables (`.env`)
The `docker-compose.yml` relies on environment variables. Create a file named `.env` in `service-app-infra/local/` and populate it with the values from `.env.example`:

```bash
cp service-app-infra/local/.env.example service-app-infra/local/.env
```

Each service also has its own `.env.example` for local development without Docker.

#### B. Hosts File Configuration
To ensure Keycloak operates correctly with the frontend and other services, you must map the hostname `keycloak` to your local machine.

- **Windows (PowerShell as Admin):**
    ```powershell
    Add-Content -Path C:\Windows\System32\drivers\etc\hosts -Value "`n127.0.0.1 keycloak"
    ```
- **macOS / Linux:**
    ```bash
    sudo echo "127.0.0.1 keycloak" >> /etc/hosts
    ```

### 3. Startup

Run the following command from the `service-app-infra/local/` directory to build the images and start all containers:

```bash
cd service-app-infra/local
docker-compose up -d --build
```

### 4. Verify

Once all containers are healthy, the services are available at:

| Service | URL |
| :--- | :--- |
| **Gateway** | http://localhost:8090 |
| **Keycloak** | http://localhost:8080 |
| **Member Service** | http://localhost:8081 |
| **Pricing Service** | http://localhost:8082 |
| **Member Request Service** | http://localhost:8084 |
| **Recommendation Service** | http://localhost:8085 |
| **RabbitMQ Dashboard** | http://localhost:15672 |
| **OTel Collector** | http://localhost:8888 |

## Running on Kubernetes (Kind)

For local Kubernetes deployment using Kind:

```bash
# Deploy the full stack
./deploy-to-kind.sh

# Clean up when done
./cleanup.sh
```

## Inter-Service Communication

```
                    Prospect (public)
                          |
                          | POST /api/v1/member-requests
                          v
        +------------------------------------------+
        |   member-request-service (:8084)         |
        |  Redis SETNX dedup (5 min TTL)           |
        |  Kafka produce -----------------------------+
        +------------------------------------------+   |
                                                     |  topic: member.requests.topic
                                                     v
        +------------------------------------------+  +---------------------------+
        |       member-service (:8081)             |  |    pricing-service (:8082) |
        |  Kafka consume ------------------------->|  |                           |
        |   -> Redis hash member-requests          |  | PUT /prices/{type}        |
        |                                         |  |   -> upsert MongoDB       |
        |  REST GET /prices <---------------------------  -> RabbitMQ publish ----+
        |  RabbitMQ consume <--------------------------------------------+
        |   -> refresh Redis price cache           |  exchange: pricing.exchange
        |                                         |  routing:  price.updated.key
        |  PostgreSQL (members)                    |  queue:    queue.price-updated.member-service
        +------------------------------------------+
                           ^
                           |
        +------------------------------------------+
        |     service-app-gateway (:8090)          |
        |  routes by path prefix                   |
        |  JWT auth + rate limit + circuit breaker |
        +------------------------------------------+
                           ^
                           | Angular (4200) / Compose client

        Keycloak (8080) provides JWT
        OTel collector (4318) -> Datadog
        Weaviate (8091) for RAG vector search
```

## Testing

Run unit tests for any service:

```bash
# From the service directory
cd member-service && go test ./...
cd pricing-service && go test ./...
cd member-request-service && go test ./...
cd service-app-gateway && go test ./...
cd recommendation-service && go test ./...
```
