# Decision: Service Discovery & Messaging Clients for the Go Port

This document evaluates the most-used industry patterns for two cross-cutting choices required to port the Spring `service-app` to Go: **service discovery** (replacing Netflix Eureka) and the **Kafka client library**. It compares the realistic options against identical metrics and culminates in an executive summary with a recommendation.

---

## Decision 1 — Service Discovery (replacing Netflix Eureka)

The Spring app uses a standalone `service-app-registry` (Eureka Server) for local/Docker discovery and disables it in Kubernetes (using `spring-cloud-kubernetes` + DNS). The Go port has no Eureka equivalent and the user asked to **skip the registry service** and pick the most-used pattern.

### Option 1: Platform-native DNS (Docker Compose service names + Kubernetes Services/DNS)

No standalone registry process. Each service is addressed by its Compose service name (`http://pricing-service:8082`) locally and by its Kubernetes Service DNS (`http://pricing-service:8082`) in k8s.

* **Where it runs:** Docker Compose (local) and Kubernetes (prod). Both already provide built-in DNS resolution between containers/pods.
* **How it is triggered:** A service simply makes an HTTP call to `http://<service-name>:<port>`. The platform resolves the name. Kubernetes Services add L4 load balancing across pods.
* **Code Explanation:** No client-side discovery code. The pricing base-url is read from env (`PRICING_SERVICE_URL=http://pricing-service:8082/api/v1`).
    ```yaml
    # docker-compose.yml
    services:
      pricing-service:
        image: pricing-service:1
      member-service:
        environment:
          - PRICING_SERVICE_URL=http://pricing-service:8082/api/v1
    ```
* **Impact/Experience:** Zero operational burden. Developers already understand DNS. No extra container, no heartbeat/lease tuning, no Eureka learning curve. Matches how the Spring app already behaves in k8s (Eureka disabled, k8s DNS used).
* **Cost:** Free — uses capabilities already present in Compose and k8s.
* **Speed/Performance:** DNS is resolved by the platform resolver; connection reuse (keep-alive) eliminates per-call lookup. Effectively zero overhead.

### Option 2: HashiCorp Consul + Go agent

Run a Consul cluster (or single dev agent) and have each Go service register itself and resolve others via Consul's DNS or HTTP API.

* **Where it runs:** A Consul agent per host/cluster plus a server cluster (≥1 in dev, ≥3 in prod).
* **How it is triggered:** Services call the Consul HTTP API (`/v1/catalog/service/<name>`) or use Consul DNS (`<service>.service.consul`) on startup and periodically send heartbeats.
* **Code Explanation:** Requires `github.com/hashicorp/consul/api` integration, health checks, and TTL heartbeats in each `main.go`.
    ```go
    client, _ := consul.NewClient(consul.DefaultConfig())
    client.Agent().ServiceRegister(&consul.AgentServiceRegistration{
        Name: "pricing-service", Port: 8082,
        Check: &consul.AgentServiceCheck{HTTP: "http://localhost:8082/actuator/health", Interval: "10s"},
    })
    ```
* **Impact/Experience:** Adds a distributed system to operate. Useful for multi-runtime or hybrid deployments, but overkill for a single-language, single-orchestrator project.
* **Cost:** Free OSS, but real cost is operational complexity (running a quorum, backups, ACLs).
* **Speed/Performance:** Fast, but introduces an extra network hop and heartbeat traffic; client-side caching recommended.

### Option 3: Custom in-process registry (mimic Eureka)

Build a small Go HTTP server holding an in-memory service map with heartbeats, replicating what Eureka does.

* **Where it runs:** A new `service-app-registry` Go service.
* **How it is triggered:** Services POST heartbeats; callers GET the registry.
* **Code Explanation:** Several hundred lines of Go for registration, lease expiry, and a client SDK — re-implementing what the platform already gives for free.
* **Impact/Experience:** High maintenance, reinvents the wheel, no battle-testing. Strongly discouraged.
* **Cost:** Engineering time + ongoing maintenance.
* **Speed/Performance:** Fine, but pointless given platform DNS exists.

---

## Decision 2 — Kafka Client Library

The Spring `member-request-service` produces and the `member-service` consumes Kafka events (`member.requests.topic`, JSON values, no type headers). The Go port needs a producer and a consumer.

### Option 1: segmentio/kafka-go

Idiomatic Go library built on top of `net/http`-style context support, pure Go (no CGO), simple Reader/Writer API.

* **Where it runs:** Inside each Go service process.
* **How it is triggered:** `kafka.NewWriter` / `kafka.NewReader` against the broker address.
* **Code Explanation:**
    ```go
    w := &kafka.Writer{Addr: kafka.TCP("kafka:9092"), Topic: "member.requests.topic",
        Balancer: &kafka.Hash{}, AllowAutoTopicCreation: true}
    w.WriteMessages(ctx, kafka.Message{Key: []byte(email), Value: jsonBytes})
    ```
* **Impact/Experience:** Most popular Go Kafka library by GitHub adoption (12k+ stars), excellent docs, context-native, clean API. Fits Gin/Golang idioms. No CGO.
* **Cost:** Free.
* **Speed/Performance:** Good throughput; pure Go. Slightly lower max throughput than Sarama for some workloads but more than enough here.

### Option 2: IBM/sarama

The original, most mature Go Kafka client. Comprehensive protocol support.

* **Where it runs:** Inside each Go service process.
* **How it is triggered:** `sarama.NewSyncProducer` / `sarama.NewConsumerGroup`.
* **Code Explanation:** Verbose config struct (`sarama.NewConfig()`), manual partition/offset handling, channel-based consumption.
* **Impact/Experience:** Mature and battle-tested, but the API is older/verbose, config surface is large, and it's heavier to learn. Maintained under IBM/sarama now.
* **Cost:** Free.
* **Speed/Performance:** Highest raw throughput in benchmarks; CGO-free.

### Option 3: twmb/franz-go

Modern, fast, pure-Go client with a fluent API and full feature set (transactions, exactly-once).

* **Where it runs:** Inside each Go service process.
* **How it is triggered:** `kgo.NewClient(kgo.SeedBrokers("kafka:9092"))`.
* **Code Explanation:**
    ```go
    cl, _ := kgo.NewClient(kgo.SeedBrokers("kafka:9092"))
    cl.Produce(ctx, &kgo.Record{Topic: "member.requests.topic", Key: []byte(email), Value: jsonBytes}, nil)
    ```
* **Impact/Experience:** Newest of the three. Excellent design and performance, but smaller community/adoption than segmentio/kafka-go and a steeper mental model (client-per-process, sources).
* **Cost:** Free.
* **Speed/Performance:** Excellent (often best-in-class), pure Go.

---

## Core Baseline & Assumptions

* **Environment:** Docker Compose (local) + Kubernetes (prod). Both already provide DNS + L4 load balancing.
* **Traffic:** Internal microservice calls, low-to-moderate volume (club management, not high-frequency trading).
* **Kafka usage:** One topic, JSON values, single partition is acceptable in dev, group `member-service-group`.
* **Constraint:** The user explicitly asked to **skip `service-app-registry`** and use the most-used pattern.

---

## Executive Summary & Recommendation

### Service Discovery

| Feature | 1. Platform-native DNS | 2. Consul | 3. Custom registry |
| :--- | :--- | :--- | :--- |
| **Developer Experience** | Excellent (nothing to learn) | Good | Poor |
| **Operational Burden** | None | High (quorum, ACLs) | High (build + maintain) |
| **Cost** | Free | Free OSS + ops cost | Engineering cost |
| **Setup Complexity** | None | Medium-High | High |
| **Industry adoption (Go)** | Dominant for Compose/k8s | Common in multi-runtime | Rare / discouraged |

**Recommendation:** **Option 1 — Platform-native DNS.** Drop `service-app-registry` entirely. Use Compose service names locally and Kubernetes Services/DNS in prod. This is the de-facto industry pattern for Go microservices on Compose/k8s and matches the Spring app's own k8s behavior (Eureka disabled).

### Kafka Client

| Feature | 1. segmentio/kafka-go | 2. IBM/sarama | 3. twmb/franz-go |
| :--- | :--- | :--- | :--- |
| **Developer Experience** | Excellent (context, simple API) | Fair (verbose) | Good (fluent) |
| **Maturity / Adoption** | High (most stars) | Highest (original) | Growing (newest) |
| **CGO-free** | Yes | Yes | Yes |
| **Setup Complexity** | Low | Medium | Low-Medium |
| **Fit for this project** | Excellent | Adequate | Excellent |

**Recommendation:** **Option 1 — segmentio/kafka-go.** It is the most-used idiomatic Go Kafka library, context-native (fits Gin's `c.Request.Context()` and graceful shutdown), pure Go, and simple enough for a single-topic producer + group consumer.

**Final Decision (per user instruction to use the most-used pattern and skip the registry):**
- Service discovery: **Platform-native DNS** (no registry service).
- Kafka client: **segmentio/kafka-go**.
- Redis client: **redis/go-redis/v9** (undisputed industry standard, not compared here).
- AMQP client: **rabbitmq/amqp091-go** (already in use, official RabbitMQ client).
