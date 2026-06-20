**Description:**

The Go port (`member-service`, `pricing-service`, `service-app-infra`) currently diverges from the Spring reference contracts documented in `java-go-overview-migration.md`. This issue tracks aligning the three existing Go folders to the Spring behavior, using the most-used industry patterns, and **skipping `service-app-registry`** (platform-native DNS instead of Eureka).

**Current state (delta vs Spring):**
- `member-service` runs on `:8090` (Spring = `8081`); `AuthMiddleware` never sets `manager_id` (latent bug — controller reads an empty value); only Member CRUD; missing `pricing` (Redis cache-aside + RabbitMQ consumer + pricing REST client) and `request` (Kafka consumer + Redis hash) sub-domains.
- `pricing-service` runs a RabbitMQ **consumer** (Spring **publishes**); CRUD-by-id endpoints (Spring = `GET /prices` public + `PUT /prices/{priceType}` manager/admin); `PriceType` JSON = uppercase `FREE` (Spring = lowercase `free`); Dockerfile exposes `8091` (Spring = `8082`); `GET /prices` auth-protected (Spring = public); no role guard on `PUT`.
- `service-app-infra` `docker-compose.yml` and k8s app manifests still reference `service-app-registry`, `SPRING_PROFILES_ACTIVE`, and JVM OTLP envs.

**Decisions (see `docs/decisions/go-patterns-decision.md` and `docs/adr/0001-drop-eureka-align-go-services.md`):**
- Service discovery: **platform-native DNS** (Docker Compose service names + Kubernetes Services/DNS) — drop `service-app-registry`.
- Kafka client: **segmentio/kafka-go**; Redis: **redis/go-redis/v9**; AMQP: **rabbitmq/amqp091-go**; JWT: **golang-jwt/jwt/v5** (JWKS).

**Scope:** `member-service`, `pricing-service`, `service-app-infra`. The gateway / member-request-service / recommendation-service are follow-up issues.
