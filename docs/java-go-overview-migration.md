# Service App — Spring Reference Review (for Go reproduction)

> This document is a **complete analysis** of the `service-app-spring/` folder, written so that the **same system can be reproduced in Go** (with Go particularities). Every service, file, port, endpoint, entity, queue/topic, config value, infrastructure manifest and inter‑service interaction is documented below.

---

## 1. Project Overview

A microservices platform for managing an exclusive recreational club. It provides:

- **Role-based access** (Manager / Member / Admin) via Keycloak (OAuth2 / OIDC / JWT).
- **Management Dashboard** for managers: CRUD members + edit-only pricing tiers.
- **Public Portal / Landing**: show the 3 membership tiers (Garden Pass = `free`, Club Membership = `half-price`, Patron Membership = `full-price`).
- **AI Recommendation Assistant**: Spring AI (RAG + Google GenAI + Weaviate vector DB) to help users choose a plan from natural language.
- **Member request intake**: prospects submit `{email, serviceType}` which flows through Kafka into the member-service and is cached in Redis.
- **Chat feature** (planned) for authenticated members.

### Microservices (7 components)

| Service | Port | Language role | DB | Messaging |
| :--- | :--- | :--- | :--- | :--- |
| `service-app-registry` | 8761 | Eureka Server (service discovery) | — | — |
| `service-app-gateway` | 8090 | Spring Cloud Gateway (WebFlux) — routing, auth, rate limiting, circuit breaker | Redis (rate limiter) | RabbitMQ (declared) |
| `member-service` | 8081 | Member CRUD + price cache + request consumer | PostgreSQL (Flyway) | RabbitMQ (consume) + Kafka (consume) + Redis |
| `pricing-service` | 8082 | 3 pricing tiers CRUD, publishes price updates | MongoDB | RabbitMQ (publish) |
| `member-request-service` | 8084 | Ingests prospect requests, dedup via Redis, produces Kafka event | Redis | Kafka (produce) |
| `recommendation-service` | 8085 | RAG AI assistant | Weaviate (vector) | — |
| `service-app-infra` | — | Infra: k8s manifests, docker-compose, Keycloak realm, OTel collector | — | — |

### Top-level repo files (Spring)

```
service-app-spring/
├── README.md            # Badges, overview, docker-compose run instructions
├── ServiceApp.md        # Domain description, tools, to-do, in-progress
├── deploy-to-kind.sh    # Creates Kind cluster 'service-app', builds JARs, builds+loads Docker images, applies k8s in order, waits for rollouts
├── cleanup.sh           # Deletes the Kind cluster 'service-app'
├── .gitignore           # Java/Maven/IDE/OS ignores + .env
├── member-service/
├── member-request-service/
├── pricing-service/
├── recommendation-service/
├── service-app-gateway/
├── service-app-registry/
└── service-app-infra/
```

---

## 2. Technology Mapping: Spring -> Go

| Concern | Spring (this repo) | Recommended Go equivalent (already partly used) |
| :--- | :--- | :--- |
| Language / runtime | Java 25, Spring Boot 4.0.2 | Go 1.25 |
| Build | Maven (`pom.xml` per service, no parent aggregator) | `go mod` (`go.mod` per service) |
| Web framework (servlet) | `spring-boot-starter-web` (Tomcat) | **Gin** (`github.com/gin-gonic/gin`) — already used |
| Web framework (reactive) | `spring-cloud-starter-gateway-server-webflux` (Netty) | Gin (or custom reverse proxy for gateway) |
| DI | Spring `@Component`/`@Service`/`@Configuration` + constructor injection | **Manual constructor injection** (`NewXxxService(repo)`) — already used |
| Config | `application.yml` + profiles (`-docker`, `-k8s`) + `${ENV}` | **`.env` + `godotenv` + `os.Getenv`** — already used; profiles -> env-driven conditionals or config structs |
| ORM (SQL) | Spring Data JPA / Hibernate, `JpaRepository` | **GORM** (`gorm.io/gorm` + `gorm.io/driver/postgres`) — already used |
| DB migrations | **Flyway** (`db/migration/V1__...sql`) | `gorm.AutoMigrate` (already used) **or** golang-migrate for versioned SQL |
| Document DB | Spring Data MongoDB (`MongoRepository`) | **go.mongodb.org/mongo-driver** — already used |
| Cache / KV | Spring Data Redis (`RedisTemplate`) | **go-redis** (`github.com/redis/go-redis/v9`) |
| Messaging (AMQP) | Spring AMQP / RabbitMQ (`RabbitTemplate`, `@RabbitListener`) | **rabbitmq/amqp091-go** — already used (consumer) |
| Messaging (Kafka) | Spring Kafka (`KafkaTemplate`, `@KafkaListener`) | **segmentio/kafka-go** or **IBM/sarama** or **twmb/franz-go** |
| Security / JWT | Spring Security + OAuth2 Resource Server (JWK set URI / issuer-uri) | **golang-jwt/jwt/v5** + manual JWKS fetch — already used in `security_config.go` |
| Validation | `jakarta.validation` (`@NotBlank`, `@Email`, `@Past`, `@DecimalMin`) | **go-playground/validator/v10** (comes with Gin via `ShouldBindJSON`) |
| OpenAPI / Swagger | springdoc-openapi (annotations) | **swaggo/swag** (comment annotations) — already used in pricing controller |
| Service discovery | Netflix Eureka (server + client) | **Consul** / **etcd** / Kubernetes-native Services + DNS (no Eureka in Go) |
| Load balancing | `lb://` Eureka-aware | Kubernetes Service DNS / round-robin / Traefik/Envoy |
| API Gateway | Spring Cloud Gateway (routes in YAML) | **Reverse proxy**: `net/http/httputil.ReverseProxy`, or **Traefik**/**Envoy** as sidecar |
| Rate limiting | `RedisRateLimiter` (token bucket, 10 replenish / 20 burst / 1 requested) | **ulule/limiter** + Redis, or **golang.org/x/time/rate** |
| Circuit breaker | Resilience4J reactive (`CircuitBreaker` filter + fallback URI) | **sony/gobreaker** or **failsafe-go** |
| Observability | `spring-boot-starter-opentelemetry` + Micrometer + OTLP export | **go.opentelemetry.io/otel** + OTLP exporter |
| Testing | JUnit 5, Mockito, Testcontainers, WireMock, RestAssured | **testing** (stdlib) + **testcontainers-go** + **stretchr/testify** + **gin tests** |
| Container | Multi-stage Dockerfile (maven build -> `eclipse-temurin:25-jre`) | Multi-stage Dockerfile (`golang:1.25-alpine` build -> `alpine`/`scratch`) — already used |

### Key Go particularities to respect

- **No annotations**: OpenAPI/security config are done via code + comments (swag) and middleware, not annotations.
- **No magic auto-config**: every bean must be explicitly constructed in `main.go` (DI by hand).
- **Profiles**: instead of `application-docker.yml`/`application-k8s.yml`, read host names from env vars (`REDIS_HOST`, `MONGO_HOST`, etc.) and pick defaults based on an `APP_ENV` variable.
- **No Eureka in Go ecosystem**: for local Docker use service DNS names (Compose service names); for k8s use Kubernetes Services + DNS. Drop the registry service or replace with Consul/etcd if you really want a standalone registry.
- **Graceful shutdown**: use `signal.NotifyContext(ctx, SIGINT, SIGTERM)` + `ctx.Done()` (already done in pricing-service `main.go`).
- **Errors as values**: implement custom error types (`EntityNotFoundError`, `DuplicateEmailError`, `AccessDeniedError`) + a Gin `GlobalErrorHandler()` middleware that switches on type — already done.

---

## 3. Spring Project — Full Directory Structure

```
service-app-spring/
├── README.md
├── ServiceApp.md
├── cleanup.sh
├── deploy-to-kind.sh
├── .gitignore
│
├── service-app-registry/
│   ├── pom.xml
│   ├── Dockerfile
│   ├── .gitignore
│   └── src/main/
│       ├── java/com/eslirodrigues/ServiceAppRegistryApplication.java   # @EnableEurekaServer
│       └── resources/application.yml                                    # port 8761, OTLP export
│
├── service-app-gateway/
│   ├── pom.xml
│   ├── Dockerfile
│   ├── .env.example                     # REDIS_PASSWORD
│   ├── .gitattributes / .gitignore
│   └── src/main/
│       ├── java/com/eslirodrigues/service_app_gateway/
│       │   ├── ServiceAppGatewayApplication.java     # @EnableDiscoveryClient
│       │   ├── config/GatewayConfig.java             # KeyResolver (JWT sub), RedisRateLimiter(10,20,1)
│       │   ├── config/SecurityConfig.java            # WebFlux security, CORS, public paths
│       │   ├── config/OpenApiConfig.java             # bearerAuth scheme
│       │   └── controller/FallbackController.java    # /fallback/{service} -> 503
│       └── resources/
│           ├── application.yml          # routes, rate limiter, circuit breaker, eureka, OTLP
│           ├── application-docker.yml   # eureka -> service-app-registry:8761
│           └── application-k8s.yml      # k8s discovery, eureka disabled
│
├── member-service/
│   ├── pom.xml
│   ├── Dockerfile
│   ├── .env.example                     # MEMBER_DB_*, RABBITMQ_*, REDIS_PASSWORD
│   └── src/main/
│       ├── java/com/eslirodrigues/member/
│       │   ├── MemberServiceApplication.java         # @EnableDiscoveryClient
│       │   ├── core/config/RedisConfig.java          # genericRedisTemplate (JSON serializer)
│       │   ├── core/config/SecurityConfig.java       # JWT, realm_access.roles -> ROLE_*
│       │   ├── core/entity/Member.java               # JPA entity
│       │   ├── core/entity/ServiceType.java          # enum FREE/HALF_PRICE/FULL_PRICE
│       │   ├── core/exception/DuplicateEmailException.java
│       │   ├── core/exception/GlobalExceptionHandler.java   # ProblemDetail (RFC7807)
│       │   ├── member/controller/MemberController.java       # /api/v1/members CRUD
│       │   ├── member/dto/CreateMemberRequest.java           # record + validation
│       │   ├── member/dto/UpdateMemberRequest.java           # record + validation
│       │   ├── member/repository/MemberRepository.java       # JpaRepository
│       │   ├── member/service/MemberService.java
│       │   ├── pricing/client/PricingServiceClient.java      # RestClient -> /prices
│       │   ├── pricing/config/RabbitMQConfig.java            # topic exchange + queue + binding
│       │   ├── pricing/config/RestClientConfig.java          # RestClient bean
│       │   ├── pricing/controller/PriceController.java       # /api/v1/members/prices (cache)
│       │   ├── pricing/dto/PriceUpdateEventDTO.java          # record + nested PriceType enum
│       │   ├── pricing/service/PriceCacheService.java        # Redis cache-aside
│       │   ├── pricing/service/PriceUpdateListener.java      # @RabbitListener
│       │   ├── request/config/KafkaConsumerConfig.java       # Kafka consumer factory
│       │   ├── request/controller/MemberRequestController.java  # /api/v1/members/requests
│       │   ├── request/dto/MemberRequestEvent.java           # record(email, serviceType)
│       │   ├── request/service/MemberRequestConsumer.java    # @KafkaListener -> Redis hash
│       │   └── request/service/MemberRequestService.java     # reads Redis hash
│       └── resources/
│           ├── application.yml / -docker.yml / -k8s.yml
│           ├── db/migration/V1__Create_member_table.sql
│           └── http/member.http + http-client.env.json
│
├── pricing-service/
│   ├── pom.xml
│   ├── Dockerfile
│   ├── .env.example                     # PRICING_DB_*, RABBITMQ_*
│   └── src/main/
│       ├── java/com/eslirodrigues/pricing_service/
│       │   ├── PricingServiceApplication.java       # @EnableDiscoveryClient
│       │   ├── config/RabbitMQConfig.java           # TopicExchange + RabbitTemplate (JSON)
│       │   ├── config/SecurityConfig.java           # JWT, GET /api/v1/prices public
│       │   ├── controller/PriceController.java      # GET + PUT /api/v1/prices/{priceType}
│       │   ├── converter/StringToPriceTypeConverter.java
│       │   ├── dto/PriceUpdateDTO.java              # record(value, description) + validation
│       │   ├── entity/Price.java                    # @Document("prices")
│       │   ├── entity/PriceType.java                # enum
│       │   ├── repository/PriceRepository.java      # MongoRepository
│       │   └── service/PriceService.java            # upsert + publish to RabbitMQ
│       └── resources/
│           ├── application.yml / -docker.yml / -k8s.yml
│           └── http/pricing.http + http-client.env.json
│
├── member-request-service/
│   ├── pom.xml
│   ├── Dockerfile
│   ├── .env.example                     # REDIS_PASSWORD
│   └── src/main/
│       ├── java/com/eslirodrigues/member_request_service/
│       │   ├── MemberRequestServiceApplication.java  # @EnableDiscoveryClient
│       │   ├── config/KafkaProducerConfig.java        # KafkaTemplate
│       │   ├── config/SecurityConfig.java             # POST /api/v1/member-requests public
│       │   ├── controller/MemberRequestController.java # POST -> 202 Accepted
│       │   ├── dto/MemberRequestDTO.java              # record(email, serviceType) + nested enum
│       │   ├── service/MemberRequestProducer.java     # kafkaTemplate.send(topic, email, dto)
│       │   └── service/MemberRequestService.java      # Redis SETNX dedup (5 min TTL)
│       └── resources/
│           ├── application.yml / -docker.yml / -k8s.yml
│           └── http/member-request.http + http-client.env.json
│
├── recommendation-service/
│   ├── pom.xml
│   ├── .env.example                     # GOOGLE_API_KEY, GOOGLE_CHAT_MODEL
│   ├── .gitattributes / .gitignore
│   └── src/main/
│       ├── java/com/eslirodrigues/recommendation/
│       │   ├── RecommendationServiceApplication.java
│       │   ├── config/MarkdownReader.java     # reads document/rag-text.md
│       │   ├── controller/RagController.java  # POST /rag/ask
│       │   ├── dto/QuestionRequest.java       # record(question)
│       │   └── service/RagService.java        # VectorStore + ChatClient (Google GenAI)
│       └── resources/
│           ├── application.yaml               # port 8085, spring.ai.google.genai, weaviate
│           ├── document/rag-text.md           # the 3 tiers knowledge base
│           ├── prompts/rag-prompt.md          # {context} {question} template
│           └── http/chat.http + http-client.env.json
│
└── service-app-infra/
    ├── datadog-dashboard-info.md / datadog-dashboard.json
    ├── .gitignore
    ├── k8s/
    │   ├── kustomization.yaml          # namespace service-app, secretGenerator .env.secrets
    │   ├── .env.example
    │   ├── 00-namespace.yaml
    │   ├── 01-keycloak-config.yaml     # ConfigMap with realm-import.json (users/roles/clients)
    │   ├── 02-rbac.yaml                # ServiceAccount + Role + RoleBinding
    │   ├── 10-keycloak-db.yaml         # StatefulSet postgres:17 + Service (headless)
    │   ├── 11-member-service-db.yaml   # StatefulSet postgres:17 + Service
    │   ├── 12-pricing-service-db.yaml  # StatefulSet mongo:7.0 + Service
    │   ├── 13-redis.yaml               # StatefulSet redis:7.4-alpine + Service
    │   ├── 14-rabbitmq.yaml            # StatefulSet rabbitmq:3-management + Service
    │   ├── 15-kafka-zookeeper.yaml     # StatefulSet zookeeper + kafka + Services
    │   ├── 20-keycloak.yaml            # Deployment keycloak:26.3.1 + Service (NodePort 30080)
    │   ├── 30-member-service.yaml      # Deployment + Service
    │   ├── 31-pricing-service.yaml     # Deployment + Service
    │   ├── 32-member-request-service.yaml
    │   ├── 33-service-app-gateway.yaml # Deployment + Service (NodePort 30090)
    │   └── 40-otel-collector.yaml      # ConfigMap + Deployment otel-collector-contrib:0.114.0 + Service
    └── local/
        ├── docker-compose.yml          # all infra + services (build context ../..)
        ├── .env.example
        ├── otel-collector-config.yml   # OTLP receiver -> Datadog exporter
        ├── keycloak/realm-import.json  # realm with 4 clients + 3 users
        └── http/
            ├── auth/auth.http + http-client.env.json
            ├── member-service/member.http + env
            ├── member-request-service/member-request.http + env
            └── pricing-service/pricing.http + env
```

---

## 4. Service-by-Service Detail

### 4.1 service-app-registry (Discovery) — port 8761

- **Purpose**: Netflix Eureka Server. All other services register here (in Docker/local). In k8s, Eureka is **disabled** and Kubernetes Services + `spring-cloud-kubernetes` discovery is used instead.
- **Java**: `ServiceAppRegistryApplication` with `@EnableEurekaServer`.
- **application.yml**:
  - `server.port: 8761`
  - `eureka.client.register-with-eureka: false`, `fetch-registry: false` (it's the server itself).
  - OTLP metrics/traces/logs -> `http://localhost:4318/v1/{metrics,traces,logs}`.
  - Actuator exposes `health,info,metrics`.
- **Dockerfile**: maven build -> `eclipse-temurin:25-jre`, exposes 8761.

**Go note**: There is no direct Eureka equivalent in Go. Options:
1. **Drop the registry** entirely and rely on Docker Compose service names + Kubernetes DNS (simplest, recommended for this project).
2. Use **HashiCorp Consul** + a Go agent, or **etcd**.
3. If you must mimic Eureka, you'd implement an HTTP server holding an in-memory service registry with heartbeats — overkill here.

---

### 4.2 service-app-gateway (API Gateway) — port 8090

- **Purpose**: single entry point. Routes requests to downstream services by path prefix, enforces JWT (OAuth2 resource server), applies **rate limiting** (Redis token bucket) and **circuit breaker** (Resilience4J) with fallback endpoints.
- **Stack**: reactive WebFlux + Spring Cloud Gateway + Spring Security (WebFlux) + Redis + RabbitMQ (declared but only for connectivity check) + Micrometer Prometheus.
- **Routes** (from `application.yml`):

| Route id | URI | Predicate (path) | Filters |
| :--- | :--- | :--- | :--- |
| `member-request-service-api` | `lb://member-request-service` | `/api/v1/member-requests/**` | RequestRateLimiter + CircuitBreaker(fallback `/fallback/member-request-service`) |
| `member-request-service-springdoc` | `lb://member-request-service` | `/member-request-service/v3/api-docs` | StripPrefix=1 |
| `member-service-api` | `lb://member-service` | `/api/v1/members/**` | RequestRateLimiter + CircuitBreaker(fallback `/fallback/member-service`) |
| `member-service-springdoc` | `lb://member-service` | `/member-service/v3/api-docs` | StripPrefix=1 |
| `pricing-service-api` | `lb://pricing-service` | `/api/v1/prices/**` | RequestRateLimiter + CircuitBreaker(fallback `/fallback/pricing-service`) |
| `pricing-service-springdoc` | `lb://pricing-service` | `/pricing-service/v3/api-docs` | StripPrefix=1 |

- **Rate limiter**: `RedisRateLimiter(10, 20, 1)` = **10 tokens/sec replenish, 20 burst, 1 requested tokens per request**. `KeyResolver` = JWT `sub` claim (or `"anonymous"`).
- **Security** (`SecurityConfig.java`):
  - CORS allows origin `http://localhost:4200` (Angular), methods GET/POST/PUT/DELETE/OPTIONS, credentials true.
  - Public: `/fallback/**`, actuator health/info, swagger, `POST /api/v1/member-requests/**`, `GET /api/v1/prices`, OPTIONS `/**`.
  - Everything else: authenticated.
  - OAuth2 resource server JWT (JWK set URI from Keycloak).
- **Fallback** (`FallbackController`): GET/POST `/fallback/{member-request-service|member-service|pricing-service}` -> `503 SERVICE_UNAVAILABLE` with `{error, message, timestamp, service}`.
- **Swagger aggregation**: `springdoc.swagger-ui.urls` lists the 3 downstream api-docs so the gateway UI shows all APIs.
- **Profiles**:
  - default: Eureka `http://localhost:8761/eureka/`.
  - docker: Eureka `http://service-app-registry:8761/eureka/`, Redis `redis`, RabbitMQ `rabbitmq`.
  - k8s: `spring.cloud.kubernetes.discovery.enabled=true`, `eureka.client.enabled=false`, Prometheus + health probes exposed.

**Go reproduction**:
- Implement a reverse proxy using `net/http/httputil.NewSingleHostReverseProxy` per route, or adopt **Traefik** (labels-based routing) / **Envoy**.
- Auth middleware = the same JWKS approach already in `member-service/core/config/security_config.go`.
- Rate limiting: `github.com/ulule/limiter/v3` with Redis store, or `golang.org/x/time/rate` per-IP/per-user.
- Circuit breaker: `github.com/sony/gobreaker` wrapping each proxy round-trip; on open, return the fallback JSON.
- CORS: `github.com/gin-contrib/cors`.

---

### 4.3 member-service — port 8081 (the most complex service)

This service owns **three sub-domains** in one Spring app: `core`, `member`, `pricing` (price cache), `request` (Kafka consumer). In Go you can mirror this with sub-packages.

#### 4.3.1 Dependencies (pom.xml highlights)
springdoc-openapi-webmvc-ui, opentelemetry, spring-cloud-kubernetes-client-all, spring-cloud-netflix-eureka-client, spring-boot-starter-amqp, spring-boot-starter-kafka, actuator, validation, data-jpa, security, oauth2-resource-server, web, data-redis, flyway (+ flyway-database-postgresql), postgresql. Tests: testcontainers (postgresql, rabbitmq), rest-assured, wiremock-standalone, commons-compress/lang3.

#### 4.3.2 Entities

**`Member`** (JPA `@Table(name="member")`):
| Field | Column | Type | Constraints |
| :--- | :--- | :--- | :--- |
| id | id | Long | PK, IDENTITY |
| name | name | String | NOT NULL |
| email | email | String | UNIQUE NOT NULL |
| birthDate | birth_date | LocalDate | — |
| photo | photo | String | — |
| serviceType | service_type | ServiceType enum (STRING) | — |
| managerId | manager_id | String | NOT NULL |

**`ServiceType`** enum: `FREE("free")`, `HALF_PRICE("half-price")`, `FULL_PRICE("full-price")`.
- `@JsonValue` returns the lowercase `value`; `@JsonCreator fromValue` accepts both lowercase (`free`) and uppercase name (`FREE`) -> flexible deserialization.

#### 4.3.3 Flyway migration `V1__Create_member_table.sql`
```sql
CREATE TABLE member (
    id BIGINT GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    birth_date DATE,
    photo VARCHAR(255),
    service_type VARCHAR(100),
    manager_id VARCHAR(255) NOT NULL
);
```

#### 4.3.4 Member CRUD (`/api/v1/members`) — `MemberController`
All endpoints `@PreAuthorize("hasRole('manager') or hasRole('admin')")`. Manager id is taken from JWT `sub` (`@AuthenticationPrincipal Jwt jwt` -> `jwt.getSubject()`).

| Method | Path | Body | Returns |
| :--- | :--- | :--- | :--- |
| GET | `/api/v1/members` | — | `List<Member>` filtered by managerId |
| GET | `/api/v1/members/{memberId}` | — | `Member` (404 if not found, 403 if managerId mismatch) |
| POST | `/api/v1/members` | `CreateMemberRequest` | 201 `Member` (409 if email exists) |
| PUT | `/api/v1/members/{memberId}` | `UpdateMemberRequest` | 200 `Member` (partial update via `Optional.ofNullable`) |
| DELETE | `/api/v1/members/{memberId}` | — | 204 (404 if not found) |

**`CreateMemberRequest`** record: `name(@NotBlank)`, `email(@NotBlank @Email)`, `birthDate(@Past)`, `photo`, `serviceType(@NotNull)`.
**`UpdateMemberRequest`** record: `name`, `email`, `birthDate`, `photo`, `serviceType` (nullable for partial update).

**`MemberRepository`** (JpaRepository): `findAllByManagerId(String)`, `findByEmail(String)`.

**`MemberService`**: create checks duplicate email -> `DuplicateEmailException`; get checks managerId ownership -> `AccessDeniedException`; not found -> `jakarta.persistence.EntityNotFoundException`.

#### 4.3.5 Price cache sub-domain (`/api/v1/members/prices`) — `pricing` package
- `PricingServiceClient` calls pricing-service via `RestClient` GET `/prices` -> `List<PriceUpdateEventDTO>`.
- `RestClientConfig` builds `RestClient` with base url `app.services.pricing.base-url` (`http://localhost:8082/api/v1` locally, `lb://pricing-service/api/v1` in docker, `http://pricing-service:8082/api/v1` in k8s).
- `PriceCacheService` (cache-aside with Redis):
  - Keys: `price-update:{priceType}` (e.g. `price-update:free`).
  - `getAllPrices()` does `multiGet` over the 3 keys; if all present return from cache, else fetch from pricing-service and `set` each.
- `PriceUpdateListener` `@RabbitListener(queues="${app.rabbitmq.queue.price-updated}")` -> calls `cachePriceUpdate` -> updates Redis. So the cache is **push-invalidated** by RabbitMQ events.
- `PriceController` GET `/api/v1/members/prices` -> `priceCacheService.getAllPrices()`.
- `PriceUpdateEventDTO` record: `id, priceType(PriceType enum), value(BigDecimal), description, createdAt, updatedAt`.

**RabbitMQ config** (`RabbitMQConfig.java`):
- Exchange: `pricing.exchange` (topic).
- Queue: `queue.price-updated.member-service`.
- Routing key: `price.updated.key`.
- `JacksonJsonMessageConverter` for JSON.

**Redis config** (`RedisConfig.java`): a `genericRedisTemplate` bean with `StringRedisSerializer` keys + `GenericJacksonJsonRedisSerializer` values.

#### 4.3.6 Member request consumer sub-domain (`request` package)
- **Kafka consumer** `MemberRequestConsumer` `@KafkaListener(topics="${app.kafka.topic.member-requests}", groupId="${spring.kafka.consumer.group-id}")`:
  - Receives `MemberRequestEvent(email, serviceType)`.
  - If `memberRepository.findByEmail(email)` exists -> ignore (warn).
  - Else store in Redis **hash** `member-requests` with field=`email`, value=`serviceType` string.
- `MemberRequestController` GET `/api/v1/members/requests` -> reads the Redis hash `member-requests` and returns `List<MemberRequestEvent>`.
- Kafka config (`KafkaConsumerConfig`): `ErrorHandlingDeserializer` + `JacksonJsonDeserializer` targeting `MemberRequestEvent`, `USE_TYPE_INFO_HEADERS=false`, `TRUSTED_PACKAGES=*`. Group id `member-service-group`, `auto-offset-reset=earliest`.

#### 4.3.7 Security (`core/config/SecurityConfig.java`)
- `@EnableWebSecurity` + `@EnableMethodSecurity` (for `@PreAuthorize`).
- Stateless sessions.
- Public: swagger, actuator health/info. `/api/**` authenticated. Rest denied.
- JWT converter reads `realm_access.roles` from the token and maps each role to `ROLE_<role>` authority (so `@PreAuthorize("hasRole('manager')")` works with Keycloak realm roles).

#### 4.3.8 Configs
**application.yml** (local):
- `server.port: 8081`, `spring.threads.virtual.enabled: true`.
- Postgres `jdbc:postgresql://localhost:5435/${MEMBER_DB_NAME}`, `ddl-auto: validate`, Flyway enabled + baseline-on-migrate.
- Redis `localhost:6379` + `${REDIS_PASSWORD}`.
- OAuth2 `issuer-uri: http://keycloak:8080/realms/service-app-realm`.
- RabbitMQ `localhost:5672`. Kafka `localhost:9092`, group `member-service-group`.
- `app.rabbitmq.*`, `app.kafka.topic.member-requests: member.requests.topic`, `app.services.pricing.base-url: http://localhost:8082/api/v1`.
- Eureka `http://localhost:8761/eureka/`, prefer-ip-address, lease 30s/90s, metadata version/description/health-check-url.
- OTLP -> `localhost:4318`.

**application-docker.yml**: Postgres `member-service-db:5432`, Redis `redis`, RabbitMQ `rabbitmq`, Kafka `kafka:9092`, pricing base-url `lb://pricing-service/api/v1`, Eureka `service-app-registry:8761`.

**application-k8s.yml**: Postgres `member-service-db:5432`, Redis `redis`, RabbitMQ `rabbitmq`, Kafka `kafka:9092`, pricing base-url `http://pricing-service:8082/api/v1`, `spring.cloud.kubernetes.discovery.enabled=true`, `eureka.client.enabled=false`, Prometheus + liveness/readiness probes.

#### 4.3.9 Tests
- `MemberControllerIT`, `MemberRequestControllerIT`, `PriceControllerIT` use `@Testcontainers` (PostgreSQL 17, RabbitMQ 3-management, Redis 7-alpine), mock `JwtDecoder` to issue a dummy JWT with `sub=test-user-id`, RestAssured for HTTP, `@Sql("/cleanup.sql")` truncates between tests. `PriceControllerIT` uses **WireMock** to stub pricing-service `/prices` and verifies cache-miss -> fetch -> cache-hit (wireMock verifies exactly 1 call).

**Go equivalents**:
- `core/entity` already exists (`member.go`, `service_type.go`).
- `member/{controller,dto,repository,service}` already exists using GORM + Gin.
- Add `pricing/{client,service,controller}` for the price cache: `go-redis` v9 for cache-aside, `net/http` client (or `resty`) to call pricing-service `/prices`, AMQP consumer (`amqp091-go`) listening on `queue.price-updated.member-service` to refresh Redis.
- Add `request/{consumer,service,controller}`: Kafka consumer (`segmentio/kafka-go`) on topic `member.requests.topic`, group `member-service-group`, stores into Redis hash `member-requests`; controller GET `/api/v1/members/requests`.
- Validation: Gin's `binding:"required,email"` tags.

---

### 4.4 pricing-service — port 8082

- **DB**: MongoDB, collection `prices`.
- **Entity `Price`** (`@Document(collection="prices")`): `id(String)`, `priceType(PriceType)`, `value(BigDecimal)`, `description(String)`, `createdAt`, `updatedAt`.
- **`PriceType`** enum: same 3 values, same JSON behavior as member-service's `ServiceType`.
- **`PriceRepository`** (MongoRepository): `findByPriceType(PriceType)`.
- **`PriceService`**:
  - `getAllPrices()` -> `findAll()`.
  - `updatePrice(priceType, PriceUpdateDTO)` -> find by priceType or create new (set createdAt); set value + description + updatedAt(now); save; **publish `Price` to RabbitMQ** `pricing.exchange` / `price.updated.key`.
- **`PriceController`** (`/api/v1/prices`):
  - GET `/api/v1/prices` -> public (permitAll).
  - PUT `/api/v1/prices/{priceType}` -> `@PreAuthorize("hasRole('manager') or hasRole('admin')")`, path var converted via `StringToPriceTypeConverter`.
- **`PriceUpdateDTO`** record: `value(@NotNull @DecimalMin("0.0") BigDecimal)`, `description(@NotBlank)`.
- **RabbitMQ config**: declares `TopicExchange("pricing.exchange")`, `RabbitTemplate` with `JacksonJsonMessageConverter`.
- **Security**: like member-service, JWT + realm roles, `GET /api/v1/prices` is public.
- **Mongo connection**: `mongodb://${PRICING_DB_USERNAME}:${PRICING_DB_PASSWORD}@localhost:27017/${PRICING_DB_NAME}?authSource=admin` (local); `pricing-service-db:27017` (docker/k8s).

**Important note for Go**: The existing Go `pricing-service` has a RabbitMQ **consumer** and CRUD by `id`. To match Spring behavior, the Go pricing-service should instead **publish** to `pricing.exchange`/`price.updated.key` after each `updatePrice`, and the **member-service** should be the consumer. Also `PriceType` enum JSON should serialize to lowercase `free/half-price/full-price` and accept both casings on input.

---

### 4.5 member-request-service — port 8084

- **Purpose**: public endpoint to receive prospect requests, dedup via Redis, publish Kafka event.
- **`MemberRequestController`** POST `/api/v1/member-requests` -> 202 Accepted (public, no auth).
- **`MemberRequestDTO`** record: `email(@NotBlank @Email)`, `serviceType(@NotNull ServiceType)` with nested `ServiceType` enum (same 3 values, same JSON behavior).
- **`MemberRequestService.processSubmission`**:
  - Redis `SETNX submission:{email} "processed" EX 300` (5 min TTL).
  - If set succeeded (new) -> `memberRequestProducer.sendMemberRequest(dto)`.
  - Else (duplicate within 5 min) -> warn and ignore.
- **`MemberRequestProducer`**: `kafkaTemplate.send(topic="member.requests.topic", key=email, value=MemberRequestDTO)`. Producer uses `JsonSerializer` with `spring.json.add.type.headers=false`.
- **Security**: POST `/api/v1/member-requests/**` public; everything else authenticated.
- **No DB** — only Redis + Kafka.

**Go reproduction**:
- Gin POST handler -> bind `MemberRequestDTO` (validate email + serviceType).
- `go-redis` `SetNX(ctx, "submission:"+email, "processed", 5*time.Minute)`.
- On success produce to Kafka topic `member.requests.topic` with `segmentio/kafka-go` Writer (JSON value, no type headers).

---

### 4.6 recommendation-service — port 8085

- **Purpose**: RAG chatbot. Loads markdown knowledge base into Weaviate vector store at startup (if empty), then answers user questions using Google GenAI ChatClient with retrieved context.
- **`RagController`** POST `/rag/ask` body `{ "question": "..." }` -> returns plain string answer.
- **`RagService`**:
  - `@PostConstruct init()`: similarity search `"Garden Pass"` topK=1; if empty, load `document/rag-text.md` via `MarkdownReader` and `vectorStore.add(documents)`.
  - `generateAnswer(query)`: `vectorStore.similaritySearch(topK=50)` -> join texts into `context` -> `PromptTemplate(rag-prompt.md)` with `{context},{question}` -> `chatClient.prompt(prompt).call().content()`.
- **`MarkdownReader`**: uses Spring AI `MarkdownDocumentReader` with `horizontalRuleCreateDocument=false`, `includeBlockquote=true`, `includeCodeBlock=true`.
- **`application.yaml`**: `spring.ai.google.genai.api-key=${GOOGLE_API_KEY}`, `chat.options.model=${GOOGLE_CHAT_MODEL}` (default `gemini-2.0-flash-live`), `temperature: 0.5`, embedding api-key same. Weaviate `localhost:8091`, scheme http.
- **`rag-text.md`**: the 3 tiers with value + description (the knowledge base).
- **`rag-prompt.md`**: instructs the model to answer **only** from CONTEXT, else reply "I'm sorry, I don't have information about that."
- **pom**: `spring-ai-starter-model-google-genai`, `spring-ai-starter-model-google-genai-embedding`, `spring-ai-advisors-vector-store`, `spring-ai-starter-vector-store-weaviate`, `spring-ai-markdown-document-reader`. Also copies the OTel javaagent jar to `target/agents`.
- **No Dockerfile / no k8s yet** (marked as in-progress in ServiceApp.md).

**Go reproduction**:
- Use `github.com/google/generative-ai-go` (or `cloud.google.com/go/ai/generativelanguage`) + `google.golang.org/api` for Gemini.
- Vector store: use Weaviate Go client `github.com/weaviate/weaviate-go-client/v4`.
- Embeddings: Gemini embedding API; store docs in Weaviate; similarity search topK=50.
- Markdown chunking: write a simple splitter (by heading/paragraph) since there's no direct Spring AI Markdown reader equivalent.
- Keep the same `rag-text.md` and `rag-prompt.md` resources.

---

## 5. Inter-Service Communication (the big picture)

```
                Prospect (public)
                      │  POST /api/v1/member-requests {email, serviceType}
                      ▼
        ┌──────────────────────────────┐
        │   member-request-service     │  (port 8084, no auth)
        │  Redis SETNX dedup (5 min)   │
        │  Kafka produce ─────────────────────┐
        └──────────────────────────────┘      │ topic: member.requests.topic
                                              ▼
        ┌──────────────────────────────┐  ┌─────────────────────────────┐
        │       member-service         │  │      pricing-service        │
        │  (port 8081, JWT manager)    │  │  (port 8082, GET public)    │
        │                              │  │                             │
        │  Kafka consume ──────────────┘  │  PUT /prices/{type}         │
        │   -> Redis hash member-requests │   -> upsert Mongo           │
        │                              │  │   -> RabbitMQ publish ─────┐
        │  REST GET /prices ◄─────────────── (RestClient or cache)     │
        │  RabbitMQ consume ◄───────────────────────────────────────────┘
        │   -> refresh Redis price cache   exchange: pricing.exchange
        │                                  routing: price.updated.key
        │  PostgreSQL (members)            queue: queue.price-updated.member-service
        └──────────────────────────────┘
                       ▲
                       │ /api/v1/members/**, /api/v1/members/prices, /api/v1/members/requests
                       │
        ┌──────────────────────────────┐
        │     service-app-gateway       │ (port 8090, JWT, rate limit, circuit breaker)
        │  routes by path prefix        │
        └──────────────────────────────┘
                       ▲
                       │ Angular (4200) / Compose client
        Keycloak (8080) provides JWT; Eureka (8761) for local/docker discovery;
        OTel collector (4318) -> Datadog; Redis (6379); RabbitMQ (5672); Kafka (9092).
```

Key flows:
1. **Member request intake**: prospect -> gateway -> member-request-service -> Redis dedup -> Kafka -> member-service consumer -> Redis hash (pending requests the manager can later promote to members).
2. **Price update + cache propagation**: manager -> gateway -> pricing-service PUT -> Mongo upsert -> RabbitMQ -> member-service listener updates Redis cache. Member-service also serves cached prices at `/api/v1/members/prices`, fetching from pricing-service on cache miss.
3. **AI recommendation**: user -> recommendation-service `/rag/ask` -> Weaviate similarity search -> Gemini prompt with context -> answer.

---

## 6. Security Model (Keycloak)

### Realm: `service-app-realm`
- 3 users: `admin` (roles admin+manager), `manager` (role manager), `member` (role member). Passwords from env `REALM_ADMIN_PASSWORD`, `REALM_MANAGER_PASSWORD`, `REALM_MEMBER_PASSWORD`.
- 3 realm roles: `admin`, `manager`, `member`.
- Clients:
  - `service-app-angular` (public, standard flow, redirect `http://localhost:4200/*`, custom `sub` mapper mapping user `id` attribute to `sub` claim — so the JWT `sub` = Keycloak user id, used as `managerId`).
  - `service-app-compose` (public, for the Compose Multiplatform client; only in local realm-import).
  - `service-app-gateway`, `member-service`, `pricing-service` (bearer-only, service accounts).
- Token lifespan: access 300s, sso idle 1800s, sso max 36000s.

### How services validate
- Gateway: OAuth2 resource server, JWK set URI `http://keycloak:8080/realms/service-app-realm/protocol/openid-connect/certs`.
- member-service / pricing-service: `issuer-uri: http://keycloak:8080/realms/service-app-realm` (Spring fetches JWK set from `{issuer}/protocol/openid-connect/certs`). Roles extracted from `realm_access.roles` -> `ROLE_<role>`.
- member-request-service: JWT but POST member-requests is public.

### Login (from `auth.http`)
`POST http://localhost:8080/realms/service-app-realm/protocol/openid-connect/token` with `grant_type=password`, `client_id=service-app-angular`, `username/password`, `scope=openid profile email` -> `access_token`. The token is then sent as `Authorization: Bearer <token>` to the gateway (port 8090).

**Go note**: your `security_config.go` already builds the JWKS URL from the issuer (`{issuer}/protocol/openid-connect/certs`) and parses RSA keys — keep that. Add `ExtractRoles` usage in a `RequireRole("manager","admin")` middleware to replace `@PreAuthorize`. Extract `sub` from claims and set `c.Set("manager_id", sub)` so controllers can read it (your controller already expects `c.GetString("manager_id")` — make sure the middleware sets it).

---

## 7. Infrastructure (`service-app-infra`)

### 7.1 docker-compose (local)
Services (all on network `service-app-network`, restart `no`):
- `keycloak` `quay.io/keycloak/keycloak:26.3.1` `start-dev --import-realm`, port 8080, mounts `./keycloak` realm import, depends on `keycloak-db`.
- `keycloak-db` `postgres:16`, port 5434:5432, volume `keycloak-db-data`.
- `weaviate` `cr.weaviate.io/semitechnologies/weaviate:1.34.1`, ports 8091 + 50051 (gRPC), anonymous auth, persistence `/var/lib/weaviate`.
- `redis` `redis:7-alpine` with `--requirepass ${REDIS_PASSWORD}`, port 6379, volume `redis-data`.
- `rabbitmq` `rabbitmq:3-management`, ports 5672 + 15672, volume `rabbitmq-data`.
- `zookeeper` `confluentinc/cp-zookeeper:7.5.3`, port 2181.
- `kafka` `confluentinc/cp-kafka:7.5.3`, port 9092:29092, advertised `PLAINTEXT://kafka:9092,PLAINTEXT_HOST://localhost:9092`, depends on zookeeper.
- `otel-collector` `otel/opentelemetry-collector-contrib:0.114.0`, ports 4317/4318/8888/13133, `DD_API_KEY` env, mounts `otel-collector-config.yml`.
- `member-service-db` `postgres:16`, port 5435:5432.
- `pricing-service-db` `mongo:7.0`, port 27017.
- App services build with `context: ../..` and the per-service Dockerfile, set `SPRING_PROFILES_ACTIVE=docker` + DB/Redis/RabbitMQ/OTLP envs, healthcheck `curl /actuator/health`.

**Env vars** (`.env.example`): `KC_BOOTSTRAP_ADMIN_*`, `KC_DB_*`, `REALM_*_PASSWORD`, `MEMBER_DB_*`, `PRICING_DB_*`, `RABBITMQ_USERNAME/PW`, `REDIS_PASSWORD`, `DD_API_KEY`, `OTEL_*`, `GOOGLE_API_KEY`, `GOOGLE_CHAT_MODEL`.

### 7.2 OTel collector config
- Receivers: OTLP (http 4318, grpc 4317) + hostmetrics.
- Processors: `memory_limiter` (512Mi), `batch`, `resource` (adds `deployment.environment=local|k8s`), `metricstransform` renaming JVM metrics to Datadog-compatible names (`jvm.heap_memory`, `jvm.gc.duration`, `jvm.thread.count`, `jvm.cpu.recent_utilization`).
- Exporter: `datadog` with `${DD_API_KEY}`, site `datadoghq.com`, histograms as distributions, host metadata tags `env:local`.
- Pipelines: metrics (otlp+hostmetrics -> datadog), traces (otlp -> datadog), logs (otlp -> datadog).
- **For Go**: the JVM-specific `metricstransform` is irrelevant — Go emits `runtime.*` and `process.*` metrics via the OTel SDK; configure `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp`, `otlpmetric/otlpmetrichttp`, `otlplog/otlploghttp` pointing to `otel-collector:4318`.

### 7.3 Kubernetes manifests
- `00-namespace.yaml`: namespace `service-app`.
- `01-keycloak-config.yaml`: ConfigMap `keycloak-realm-config` with inline `realm-import.json` (same realm/users/roles/clients as local).
- `02-rbac.yaml`: ServiceAccount `service-app-sa`, Role `service-app-reader` (get/list/watch on pods/services/endpoints/configmaps), RoleBinding.
- `10-keycloak-db.yaml`, `11-member-service-db.yaml`: StatefulSet `postgres:17` + headless Service, PVC `local-path` 1Gi, liveness `pg_isready`, readiness tcp 5432.
- `12-pricing-service-db.yaml`: StatefulSet `mongo:7.0` + headless Service, readiness/liveness `mongosh ... ping`.
- `13-redis.yaml`: StatefulSet `redis:7.4-alpine` `redis-server --requirepass $REDIS_PASSWORD` + headless Service.
- `14-rabbitmq.yaml`: StatefulSet `rabbitmq:3-management` + Service (amqp 5672 + mgmt 15672), `RABBITMQ_DEFAULT_USER/PASS` from secrets.
- `15-kafka-zookeeper.yaml`: StatefulSet `zookeeper` (2181) + StatefulSet `kafka` `confluentinc/cp-kafka:7.5.3` (9092, advertised `kafka:9092`) + headless Services.
- `20-keycloak.yaml`: Deployment `keycloak:26.3.1` `start-dev --import-realm`, mounts realm ConfigMap, envs from secrets, startup/liveness/readiness probes, Service **NodePort 30080**.
- `30/31/32/33-*-service.yaml`: Deployments using images `member-service:1` etc., `serviceAccountName: service-app-sa`, `SPRING_PROFILES_ACTIVE=k8s`, OTel envs (`OTEL_SERVICE_NAME`, `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318`, `JAVA_TOOL_OPTIONS=-javaagent:/app/agents/opentelemetry-javaagent.jar`), readiness `/actuator/health/readiness`, liveness `/actuator/health/liveness`, resources. Gateway Service is **NodePort 30090** (host port 8090 via Kind extraPortMapping).
- `40-otel-collector.yaml`: ConfigMap with collector config + Deployment `otel-collector-contrib:0.114.0` + Service (4317/4318).
- `kustomization.yaml`: namespace `service-app`, `secretGenerator` from `.env.secrets`, resources list (most are commented out — only base ns/config/rbac applied via `kubectl apply -k`).
- `deploy-to-kind.sh`: creates Kind cluster `service-app` with port mappings 30090->8090 and 30080->8080, installs local-path provisioner, builds JARs, builds+loads Docker images `member-request-service:1`, `member-service:1`, `pricing-service:1`, `service-app-gateway:1`, applies infra in order with rollout waits.

**For Go k8s**: keep the same manifests but change the app Deployments to use your Go images, drop `JAVA_TOOL_OPTIONS`, replace `/actuator/health/*` probes with a Go `/actuator/health` (or `/healthz`) endpoint, and set envs (`APP_ENV=k8s`, DB host = `<svc>-db`, `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318`). Wire the OTel Go SDK in each `main.go`.

---

## 8. Observability

- Every Spring service exposes Actuator `health,info,metrics` (+ `prometheus`, `gateway` in k8s) and exports OTLP metrics/traces/logs to the collector at `localhost:4318` (local) or `otel-collector:4318` (docker/k8s).
- Collector forwards to **Datadog** (`DD_API_KEY`).
- `datadog-dashboard-info.md` describes the dashboard: RED metrics from `http.server.request.duration`, JVM health (heap, GC, threads, CPU), slowest endpoints by `http.route`.
- **For Go**: instrument each service with `go.opentelemetry.io/otel` + `otelhttp` (HTTP middleware) + `otelgrpc`/`otelmux` + `otelgin` (`go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin`). Export via OTLP HTTP to the collector. Replace JVM metrics with Go runtime metrics (`go.opentelemetry.io/contrib/instrumentation/runtime`).

---

## 9. Testing Approach (Spring) and Go equivalents

| Spring test concern | Spring tool | Go equivalent |
| :--- | :--- | :--- |
| Integration test container | `@Testcontainers` (`PostgreSQLContainer`, `RabbitMQContainer`, `MongoDBContainer`, `GenericContainer<redis>`, `ConfluentKafkaContainer`) | `github.com/testcontainers/testcontainers-go` |
| Mock JWT | `@Bean JwtDecoder` mocked with Mockito returning a dummy `Jwt` | build a fake RSA key / unsigned token in tests, or stub the middleware |
| HTTP assertions | RestAssured (`given().spec(...).body(...).post(...).then().statusCode(201)`) | `net/http/httptest` + `testify/assert` |
| Stub external HTTP | WireMock | `github.com/wiremock/go-wiremock` or `httptest.Server` |
| DB cleanup | `@Sql("/cleanup.sql")` `TRUNCATE member RESTART IDENTITY` | `db.Exec("TRUNCATE ...")` or transaction rollback |
| Unit mocking | Mockito | `github.com/stretchr/testify/mock` / `go.uber.org/mock` |

Pattern to copy: each controller gets an `*_test.go` (or `*_integration_test.go`) that spins the needed containers, points the app at them via env, starts `httptest.Server`, and asserts status codes + bodies + cache side-effects.

---

## 10. Recommended Go Project Structure (reproduction target)

```
service-app-go/
├── README.md
├── deploy-to-kind.sh            # adapt: build Go binaries, docker build, kind load
├── cleanup.sh
├── .gitignore
│
├── service-app-gateway/         # NEW (Go)
│   ├── go.mod
│   ├── main.go                  # reverse proxy + auth + rate limit + circuit breaker
│   ├── config/{security.go, ratelimit.go, cors.go, openapi.go}
│   ├── controller/fallback.go
│   ├── proxy/routes.go
│   ├── .env.example
│   └── Dockerfile
│
├── service-app-registry/        # OPTIONAL — recommend skipping (use k8s DNS / Consul)
│   └── (only if you want a standalone registry)
│
├── member-service/              # EXISTS — extend
│   ├── go.mod / main.go / Dockerfile / .env.example
│   ├── core/{config,entity,exception}/
│   ├── member/{controller,dto,repository,service}/
│   ├── pricing/                 # NEW: price cache + RabbitMQ consumer + pricing client
│   │   ├── client/pricing_client.go
│   │   ├── config/rabbitmq_config.go
│   │   ├── controller/price_controller.go
│   │   ├── dto/price_update_event.go
│   │   └── service/{price_cache_service.go, price_update_listener.go}
│   ├── request/                 # NEW: Kafka consumer + requests controller
│   │   ├── config/kafka_consumer_config.go
│   │   ├── controller/member_request_controller.go
│   │   ├── dto/member_request_event.go
│   │   └── service/{member_request_consumer.go, member_request_service.go}
│   └── ...
│
├── pricing-service/             # EXISTS — align to Spring: publish instead of consume
│   ├── go.mod / main.go / Dockerfile / .env.example
│   ├── core/{config,entity,exception}/
│   ├── pricing/{controller,dto,repository,service}/
│   └── pricing/messaging/rabbitmq_publisher.go   # change consumer -> publisher
│
├── member-request-service/      # NEW (Go)
│   ├── go.mod / main.go / Dockerfile / .env.example
│   ├── core/config/security_config.go
│   ├── request/
│   │   ├── controller/member_request_controller.go  # POST /api/v1/member-requests -> 202
│   │   ├── dto/member_request_dto.go
│   │   └── service/{member_request_service.go (Redis SETNX), kafka_producer.go}
│   └── ...
│
├── recommendation-service/      # NEW (Go)
│   ├── go.mod / main.go / Dockerfile / .env.example
│   ├── recommendation/
│   │   ├── controller/rag_controller.go   # POST /rag/ask
│   │   ├── dto/question_request.go
│   │   ├── service/rag_service.go         # Weaviate + Gemini
│   │   └── config/markdown_reader.go
│   └── resources/{document/rag-text.md, prompts/rag-prompt.md}
│
└── service-app-infra/           # EXISTS — keep, adapt app manifests for Go images
    ├── k8s/*.yaml
    └── local/{docker-compose.yml, .env.example, otel-collector-config.yml, keycloak/realm-import.json, http/...}
```

### Suggested Go module path & shared libs
- Each service is its own module: `service-app-go/<service>` (matches existing `go.mod`).
- Consider a shared internal package for common concerns (JWKS auth middleware, error types, OTel setup) — e.g. a `service-app-go/internal/auth` or a tiny `pkg/` — to avoid duplicating `security_config.go` in every service.

---

## 11. Port & Endpoint Reference (must keep identical)

| Service | Port | Base path | Public endpoints | Auth endpoints |
| :--- | :--- | :--- | :--- | :--- |
| gateway | 8090 | — | `/fallback/**`, swagger, `POST /api/v1/member-requests/**`, `GET /api/v1/prices` | everything else |
| member-service | 8081 | `/api/v1` | `/actuator/health`, `/actuator/info`, swagger | `/api/v1/members/**` (manager/admin), `/api/v1/members/prices`, `/api/v1/members/requests` |
| pricing-service | 8082 | `/api/v1` | `GET /api/v1/prices`, actuator, swagger | `PUT /api/v1/prices/{priceType}` (manager/admin) |
| member-request-service | 8084 | `/api/v1` | `POST /api/v1/member-requests/**`, actuator, swagger | other `/api/**` |
| recommendation-service | 8085 | — | `POST /rag/ask` | — |
| registry | 8761 | — | — | — |
| keycloak | 8080 (NodePort 30080 k8s) | — | token endpoint, certs | — |
| redis | 6379 | — | — | — |
| rabbitmq | 5672 (15672 mgmt) | — | — | — |
| kafka | 9092 | — | — | — |
| weaviate | 8091 (+50051 gRPC) | — | — | — |
| otel-collector | 4318 http / 4317 grpc | — | — | — |

> Note: the existing Go `member-service/main.go` runs on `:8090` — **change to `:8081`** to match the Spring service and the gateway/k8s expectations. The existing Go `pricing-service` Dockerfile exposes `8091` — change to `8082`.

---

## 12. Reproduction Checklist (step-by-step)

1. **Infra first**: keep `service-app-infra/` (k8s + docker-compose + keycloak realm + otel config) — it's technology-agnostic. Adjust the app Deployments' images and probes later.
2. **member-service** (exists): fix port to 8081; add `pricing/` (Redis cache-aside + RabbitMQ consumer + REST client to pricing-service) and `request/` (Kafka consumer + Redis hash + GET controller). Add `ExtractRoles`/`RequireRole` middleware and set `manager_id` from `sub`.
3. **pricing-service** (exists): change RabbitMQ from consumer to **publisher** on `pricing.exchange`/`price.updated.key` after `updatePrice`; expose `GET /api/v1/prices` (public) and `PUT /api/v1/prices/{priceType}` (manager/admin); use `PriceType` path param; ensure JSON enum lowercase.
4. **member-request-service** (new): Gin POST `/api/v1/member-requests` -> Redis `SETNX submission:{email}` 5m -> Kafka produce `member.requests.topic`; 202 Accepted.
5. **service-app-gateway** (new): reverse proxy by path prefix to the 3 services; JWT JWKS auth; Redis token-bucket rate limiter (10/s, burst 20); gobreaker circuit breaker with fallback JSON; CORS for localhost:4200; aggregated swagger.
6. **recommendation-service** (new): Gemini + Weaviate RAG; `POST /rag/ask`; load `rag-text.md` into Weaviate at startup.
7. **Observability**: add `otelgin` middleware + OTLP HTTP exporter in every `main.go`.
8. **Dockerfiles**: multi-stage `golang:1.25-alpine` -> `alpine` (with `ca-certificates`); expose correct ports; `CGO_ENABLED=0`.
9. **Tests**: `testcontainers-go` for Postgres/Mongo/RabbitMQ/Kafka/Redis; `httptest` + `testify`; WireMock-equivalent for the pricing client.
10. **Scripts**: adapt `deploy-to-kind.sh` to `go build` + `docker build` + `kind load docker-image` for each Go service; keep the rollout-wait logic.

---

## 13. Existing Go State vs Spring (delta summary)

| Component | Spring | Go now | Gap |
| :--- | :--- | :--- | :--- |
| member-service | full (CRUD + price cache + Kafka consumer) | CRUD only (Gin/GORM/JWT) | add `pricing/` cache + RabbitMQ consumer + REST client; add `request/` Kafka consumer; port 8081; role middleware; set manager_id |
| pricing-service | Mongo + RabbitMQ publish + PUT by type | Mongo + RabbitMQ **consume** + CRUD by id | switch to publish; `GET` public; `PUT /{priceType}`; enum path var; port 8082 |
| member-request-service | Redis dedup + Kafka produce | missing | build new |
| service-app-gateway | WebFlux gateway | missing | build new (reverse proxy) |
| recommendation-service | Spring AI RAG | missing | build new (Gemini + Weaviate) |
| service-app-registry | Eureka | missing | recommend skip (k8s DNS / Consul) |
| service-app-infra | k8s + compose + keycloak + otel | copied as-is | adapt app manifests for Go images/probes; add Go services to docker-compose |

This document contains everything needed to reproduce the system in Go while respecting Go's idioms (explicit DI, middleware, env-based config, no annotations, graceful shutdown, error-as-values).
