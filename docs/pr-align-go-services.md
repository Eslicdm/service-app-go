## What type of PR is this? (check all applicable)

- [x] Refactor
- [x] Feature
- [ ] Bug Fix
- [ ] Optimization
- [ ] Documentation Update

## Description

Aligns the existing Go `member-service`, `pricing-service`, and `service-app-infra` to the Spring reference contracts documented in `java-go-overview-migration.md`, using the most-used industry patterns and **dropping `service-app-registry`** in favor of platform-native DNS (Docker Compose service names + Kubernetes Services/DNS).

Key decisions are recorded in `docs/decisions/go-patterns-decision.md` (service discovery + Kafka client comparison) and `docs/adr/0001-drop-eureka-align-go-services.md` (ADR-0001). The implementation plan is in `.junie/issue-plans/align-go-services-to-spring-issue-plan.md`.

## Technical Changes

#### 1. pricing-service (`pricing-service/`)
- Switched RabbitMQ from a **consumer** to a **publisher** on `pricing.exchange` / `price.updated.key` (new `pricing/messaging/rabbitmq_publisher.go`); the publisher is called after each successful upsert.
- Reduced endpoints to Spring's surface: `GET /api/v1/prices` (public) + `PUT /api/v1/prices/{priceType}` (manager/admin). Removed `Create/GetByID/Delete` and `CreatePriceDTO`.
- `PriceType` JSON now serializes to lowercase `free/half-price/full-price` (Spring `@JsonValue`); `UnmarshalJSON`/`PriceTypeFromString` accept both casings.
- `PriceService.UpdatePriceByType` now **upserts by type** (creates if missing, sets `createdAt`, generates a UUID for new docs) and **allows `value=0`** for the Free tier (Spring `@DecimalMin("0.0")`); publishes after save.
- Added `RequireRole("manager","admin")` middleware; `AuthMiddleware` sets `manager_id` + `roles` from the JWT; `GET /api/v1/prices` is public via method-aware `isPublicEndpoint`.
- Port → `:8082`; Dockerfile `EXPOSE 8082` + `curl`; `.env.example` corrected to MongoDB with `MONGO_HOST`/`RABBITMQ_HOST`/`KEYCLOAK_REALM_URL`/`PORT`.
- Rewrote unit tests for the new service surface (upsert + create, `value=0` allowed).

#### 2. member-service (`member-service/`)
- Port → `:8081`; `AuthMiddleware` now sets `manager_id` (JWT `sub`) + `roles` (fixes the empty-`manager_id` bug); added `RequireRole("manager","admin")` applied to `/api/v1/members` CRUD.
- Added `ServiceType` `oneof` validation + `ParseServiceType` helper (accepts both casings).
- New `pricing/` sub-domain: `PricingServiceClient` (`GET {PRICING_SERVICE_URL}/prices`), `PriceCacheService` (Redis cache-aside `MGet` over `price-update:<type>` with pricing-service fallback), `PriceUpdateListener` (RabbitMQ consumer on `queue.price-updated.member-service`), `PriceController` (`GET /api/v1/members/prices`).
- New `request/` sub-domain: `MemberRequestConsumer` (`segmentio/kafka-go` reader, group `member-service-group`, topic `member.requests.topic` → skips existing members, else `HSet member-requests`), `MemberRequestService` + `MemberRequestController` (`GET /api/v1/members/requests`).
- `main.go` wires Postgres/GORM, `go-redis/v9`, RabbitMQ listener and Kafka consumer goroutines, pricing client, controllers, with `signal.NotifyContext` graceful shutdown.
- Dockerfile `EXPOSE 8081` + `curl`; `.env.example` extended with `REDIS/RABBITMQ/KAFKA/PRICING_SERVICE_URL/KEYCLOAK_REALM_URL/PORT` envs; added `MemberRepository.EmailExists`.

#### 3. service-app-infra (`service-app-infra/`)
- `local/docker-compose.yml`: dropped `service-app-registry`, `service-app-gateway`, `member-request-service`, `weaviate` (not built yet); rebuilt `member-service` & `pricing-service` with Go envs and `curl /actuator/health` healthchecks.
- k8s `30-member-service.yaml` & `31-pricing-service.yaml`: Go images, `/actuator/health` probes, Go OTel envs, dropped `JAVA_TOOL_OPTIONS`/`SPRING_PROFILES_ACTIVE`.
- k8s `.env.example`: OTEL defaults set to `otlp`; `local/.env.example` cleaned.
- Added Go-adapted `deploy-to-kind.sh` + `cleanup.sh` (build Go images, load into Kind).

## Related Tickets & Documents

- Related Issue: (created alongside this PR)
- Closes: (issue number)
- ADR: `docs/adr/0001-drop-eureka-align-go-services.md`
- Decision: `docs/decisions/go-patterns-decision.md`
- Plan: `.junie/issue-plans/align-go-services-to-spring-issue-plan.md`

## Added/updated tests?

- [x] Yes
  - `pricing-service/pricing/service/price_service_test.go` rewritten for the new upsert-by-type surface (float64 value, create-when-not-found, `value=0` allowed, negative/empty rejected).
  - Both services pass `go build ./...`, `go vet ./...`, and `go test ./...`.
- [ ] No, and this is why:
- [ ] I need help with writing tests
