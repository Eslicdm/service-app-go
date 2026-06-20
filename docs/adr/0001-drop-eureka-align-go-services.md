# ADR-0001: Drop Eureka registry and align Go services to Spring contracts

## Status

Accepted

## Date

2026-06-20

## Author

Esli Rodrigues

## Context

The Spring `service-app` reference (`service-app-spring/`) defines a microservices platform (gateway, member-service, pricing-service, member-request-service, recommendation-service) with Netflix Eureka for local/Docker discovery. A partial Go port already exists (`member-service`, `pricing-service`, `service-app-infra`) but diverges from the Spring contracts:

- `member-service` runs on port `8090` (Spring uses `8081`), has no role-based middleware, exposes only Member CRUD, and is missing the `pricing` (Redis cache-aside + RabbitMQ consumer + pricing REST client) and `request` (Kafka consumer + Redis hash) sub-domains.
- `pricing-service` runs a RabbitMQ **consumer** (Spring **publishes** price updates), exposes CRUD-by-id endpoints (Spring exposes `GET /prices` public + `PUT /prices/{priceType}` manager/admin), serializes `PriceType` to uppercase `FREE` (Spring serializes to lowercase `free`), and its Dockerfile exposes `8091` (Spring uses `8082`).
- `service-app-infra` was copied verbatim from Spring: `docker-compose.yml` still references `service-app-registry`, `SPRING_PROFILES_ACTIVE`, and JVM OTLP envs; k8s app manifests still use `JAVA_TOOL_OPTIONS` and Spring actuator probe sub-paths.

The user asked to port the system to Go, **skip `service-app-registry`**, and use the most-used industry pattern. A comparative analysis is recorded in `docs/decisions/go-patterns-decision.md`.

## Decision Drivers

- Parity with the Spring reference so the existing Keycloak realm, docker-compose infra, k8s manifests, and frontends work unchanged against the Go services.
- Use the most-adopted Go industry patterns (minimize custom infrastructure).
- Keep the existing Go toolchain/libraries where they already match (Gin, GORM, mongo-driver, amqp091-go, golang-jwt).
- Ports and endpoint contracts must stay identical to Spring (`member-service:8081`, `pricing-service:8082`, `/api/v1/...`).

## Considered Options

### Option 1: Platform-native DNS + align all contracts (chosen)
Drop the registry; use Docker Compose service names + Kubernetes Services/DNS. Rewrite `member-service` and `pricing-service` to match Spring endpoints/ports/messaging direction; adapt infra for Go.
**Pros:** Zero extra infrastructure; matches Spring k8s behavior; smallest operational footprint.
**Cons:** No client-side load balancing in local Docker (acceptable — Compose routes to the single replica).

### Option 2: Introduce Consul as the registry
**Pros:** Real client-side discovery, health-aware.
**Cons:** Adds a distributed system to run; overkill for Compose+k8s; diverges from "most-used pattern for Go on Compose/k8s".

### Option 3: Keep a custom Go Eureka-like registry
**Pros:** 1:1 with Spring local topology.
**Cons:** Reinvents platform DNS; high maintenance; discouraged.

## Decision

**Option 1 — Platform-native DNS + full contract alignment** is selected.

Concretely:
1. **No `service-app-registry`** in Go. Service-to-service calls use env-injected URLs (`PRICING_SERVICE_URL=http://pricing-service:8082/api/v1`) resolved by Compose/k8s DNS.
2. **`member-service`** port → `8081`; add `RequireRole("manager","admin")` middleware + set `manager_id` from JWT `sub`; add `pricing/` (Redis cache-aside `price-update:<type>`, RabbitMQ consumer on `queue.price-updated.member-service` from `pricing.exchange`/`price.updated.key`, REST client `GET /prices`) and `request/` (Kafka consumer on `member.requests.topic` group `member-service-group` → Redis hash `member-requests`; `GET /api/v1/members/requests`).
3. **`pricing-service`** port → `8082`; switch RabbitMQ to a **publisher** on `pricing.exchange`/`price.updated.key` after each `updatePrice`; expose `GET /api/v1/prices` (public) and `PUT /api/v1/prices/{priceType}` (manager/admin); `PriceType` JSON serializes to lowercase `free/half-price/full-price` and accepts both casings on input.
4. **`service-app-infra`**: `docker-compose.yml` rebuilt for Go (drop registry, drop `SPRING_PROFILES_ACTIVE`/JVM OTLP envs, add Go envs, `curl`-based healthchecks against `/actuator/health`); k8s `30/31` app manifests updated for Go images, `/actuator/health` probes, and Go OTel envs; `32/33` left as placeholders for not-yet-built services.
5. **Libraries:** `redis/go-redis/v9`, `segmentio/kafka-go`, `rabbitmq/amqp091-go`, `golang-jwt/jwt/v5` (JWKS), Gin, GORM, mongo-driver.

### Rationale
Platform DNS is the dominant pattern for Go microservices on Compose/Kubernetes and is exactly what the Spring app uses in k8s (Eureka disabled). Aligning endpoints/ports to Spring lets the existing Keycloak realm, frontends, and infra work without changes.

### Why Not Option 2/3
Consul and a custom registry add operational/engineering cost for a single-orchestrator, single-language system that already has DNS-based discovery.

## Migration Plan
1. Create feature branch `feature/align-go-services-to-spring`.
2. Refactor `pricing-service` (publisher, GET public, PUT by type, enum lowercase, port 8082, role guard).
3. Extend `member-service` (port 8081, role guard + manager_id, `pricing/`, `request/`, Redis, RabbitMQ consumer, Kafka consumer, graceful shutdown).
4. Adapt `service-app-infra` (docker-compose for Go, drop registry; k8s 30/31 for Go).
5. `go mod tidy` + `go build`/`go vet` on both services.
6. Commit incrementally (Conventional Commits); push; open PR.

## Consequences
### Positive
- Contract parity with Spring; frontends/infra/Keycloak unchanged.
- One less service to build/deploy/monitor (no registry).
- Idiomatic Go stack with strong community support.
### Negative
- No client-side load balancing in plain Compose (single replica only).
- Member-service becomes more complex (3 sub-domains in one binary).
### Neutral
- Redis/Kafka/RabbitMQ still required as external dependencies.

## Open Items
- Build the remaining services (`service-app-gateway`, `member-request-service`, `recommendation-service`) in follow-up issues.
- Add OpenTelemetry Go SDK instrumentation (`otelgin`) — tracked separately.

## Notes
Comparative analysis: `docs/decisions/go-patterns-decision.md`. Migration blueprint: `java-go-overview-migration.md`.
