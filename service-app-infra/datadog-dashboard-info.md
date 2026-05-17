# OTel Datadog Java Dashboard

---

## 1. Application Performance (RED Metrics)

These metrics are derived from OpenTelemetry's standard HTTP server metrics (`http.server.request.duration`).

* **Request Rate (Throughput):** Uses the count of request duration samples: `sum:http.server.request.duration.count`.
* **Error Rate (%):** Calculated as a formula using 5xx status codes (A=5xx count, B=total count): `(A / B) * 100`.
* **Latency (Duration):** `p50`, `p95`, and `p99` percentiles of `http.server.request.duration`.

---

## 2. JVM Health (Collector Transformed)

These metrics use the Datadog-friendly names created by the **`metricstransform`** processor in the OpenTelemetry Collector, which ensures compatibility with Datadog's built-in JVM dashboards.

* **Heap Memory Usage:** Used (`avg:jvm.heap_memory`) vs. Max (`avg:jvm.heap_memory_max`).
* **Garbage Collection Pauses:** Total count of pauses: `sum:jvm.gc.duration.as_count()`.
* **Thread Count:** The current number of threads: `avg:jvm.thread.count`.
* **CPU Utilization:** `avg:jvm.cpu.recent_utilization` (Multiplied by 100 in the widget formula for percentage display).

---

## 3. Business Intelligence & Latency Breakdown

* **Top Slowest Endpoints:** `p95:http.server.request.duration` grouped by `{http.route}` (the OpenTelemetry span attribute for the endpoint path).
* **Service Health Overview:** A single table widget summarizing Request count, Error count, calculated Error %, and p95 latency by service.