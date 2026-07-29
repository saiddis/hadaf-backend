<!--
SPDX-License-Identifier: AGPL-3.0-or-later
Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors
-->

# Observability stack

Monitoring overlay for the Hadaf backend: **metrics** (Prometheus), **logs**
(Loki) and **traces** (Tempo via the OpenTelemetry Collector), visualised in
**Grafana** with cross-correlation between all three signals.

## The mental model (read this first)

Three signals answer three different questions. Keep them separate in your head:

| Signal | Answers | Where it comes from | Cardinality |
|--------|---------|---------------------|-------------|
| **Metric** | *Is something wrong, and how much?* | app `/metrics`, Prometheus **pulls** every 15s | low (bounded labels) |
| **Log** | *What exactly happened on this request?* | app stdout JSON → Promtail → Loki | high |
| **Trace** | *Where in the chain did the time/error go?* | app **pushes** OTLP → Collector → Tempo | high |

```
                          ┌─────────── app (shb) ───────────┐
       Prometheus  ◄──pull── /metrics                       │
                          │   stdout JSON ──►  Promtail ──► Loki
                          │   OTLP spans  ──►  Collector ──► Tempo
                          └─────────────────────────────────┘
                                       ▼ ▼ ▼
                                    Grafana
                     (one trace_id links all three together)
```

The glue is **`trace_id`**: every log line carries it, every span is keyed by
it. Alert fires (metric) → open the log (Loki) → click `trace_id` → see the
exact slow span (Tempo). One incident, three lenses, one click between them.

## Quick start

```bash
# 1. Ensure you have a .env (copy from the example and fill secrets)
cp .env_example .env

# 2. Start everything — app + deps + the full monitoring stack — in one command
docker compose up -d

# 3. Open the UIs
#   Grafana     → http://localhost:3000   (admin / admin by default)
#   Prometheus  → http://localhost:9090
#   Tempo API   → http://localhost:3200
```

The observability services live in the single `docker-compose.yml` (the former
`docker-compose.observability.yml` overlay was merged in). Each service that
ships a shell carries a real healthcheck and they start in dependency order
(Loki → Promtail, Tempo → Collector, all three → Grafana), so `up -d` converges
cleanly without races.

Grafana comes pre-provisioned: the Prometheus, Loki and Tempo datasources and
the **"Hadaf API — RED & Dependencies"** dashboard load automatically. No manual
setup.

Traces are **opt-in**. The app ships with tracing off (a no-op tracer, zero
overhead). To emit spans, set in `.env`:

```bash
OTEL_TRACES_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
```

Then restart the app. Spans flow app → Collector → Tempo; in Grafana → Explore
pick **Tempo** to search traces, or click **trace_id** in a Loki log line to
jump straight to its trace (and back).

To stop a single monitoring service without touching the app (e.g. Grafana):

```bash
docker compose stop grafana
```

## One file, one network

The app's runtime deps (Postgres, Redis, MinIO) **and** the monitoring stack
both live in `docker-compose.yml`. Because everything is one compose project on
the default network, Prometheus reaches the app as `app:8000`, Promtail tails
sibling containers, and the app pushes traces to `otel-collector:4317` — all by
service name, no extra wiring.

All host ports and Grafana credentials are configured via `.env` (see the
`OBSERVABILITY` block in `.env_example`) — nothing is hardcoded.

## What each config is for

| Path | Component | Purpose |
|------|-----------|---------|
| `docker-compose.yml` | stack | App + deps + prometheus/grafana/loki/promtail/tempo/otel-collector, with healthchecks; ports via env |
| `prometheus/prometheus.yml` | Prometheus | Scrape config — pulls `app:8000/metrics` every 15s; loads alert rules |
| `prometheus/alerts.yml` | Prometheus | Alert rules: target down, 5xx rate, p99 latency, panics, external-dep errors, DB pool exhaustion |
| `grafana/provisioning/datasources/datasources.yml` | Grafana | Auto-wires Prometheus + Loki + Tempo datasources (fixed uids), with log↔trace correlation |
| `grafana/provisioning/dashboards/dashboards.yml` | Grafana | Tells Grafana to load dashboards from the mounted folder |
| `grafana/dashboards/api-red.json` | Grafana | The RED + dependencies dashboard (rate, errors, latency, external calls, DB pool, panics) |
| `loki/loki-config.yml` | Loki | Single-binary log store, filesystem backend, 7-day retention |
| `promtail/promtail-config.yml` | Promtail | Discovers Docker containers via the socket and ships their stdout logs to Loki |
| `otel-collector/config.yaml` | OTel Collector | Receives OTLP traces from the app (:4317), redacts PII, exports to Tempo |
| `tempo/tempo.yml` | Tempo | Single-binary trace store, filesystem backend, service-graph generation |

## What the app exposes

The application is already instrumented (`pkg/metrics`). Metrics served at
`GET /metrics`:

| Metric | Type | Meaning |
|--------|------|---------|
| `shb_http_requests_total{method,path,status}` | counter | Request count (RED — Rate/Errors) |
| `shb_http_request_duration_seconds{method,path,status}` | histogram | Request latency (RED — Duration) |
| `shb_external_request_duration_seconds{service,operation,status}` | histogram | Latency/errors of SMS, SMTP, MinIO calls |
| `shb_http_panics_total` | counter | Panics recovered by the middleware |
| `shb_db_pool_*` | gauge | pgx connection-pool stats (total/acquired/idle/max) |
| `go_*`, `process_*` | — | Standard Go runtime & process collectors |

`path` is the **route template** (`/api/v1/needs/:id`), never the raw URL, to
keep cardinality bounded.

Health endpoints (used by orchestrators, not Prometheus):
`GET /healthz` (liveness) and `GET /readyz` (readiness — pings Postgres & Redis).

## Logs

The app logs structured JSON (zerolog) to stdout. Promtail picks up the
container's stdout via the Docker socket and pushes it to Loki, where it is
searchable in Grafana → Explore (filter by `compose_service="app"`). Because the
logs are JSON, Grafana can parse fields (`request_id`, `level`, …) on the fly.

## Tracing (Phase 3)

The application is OpenTelemetry-instrumented end to end:

| Layer | Instrumentation | Spans produced |
|-------|-----------------|----------------|
| HTTP | `otelgin` middleware | one root span per request (route template, status) |
| Database | `otelpgx` query tracer | one child span per SQL statement |
| Cache | `redisotel` | one child span per Redis command |
| SDK | `pkg/tracing` | OTLP/gRPC exporter, batch processor, parent-based sampling |

Health/metrics endpoints are filtered out to keep traces signal-only. Every log
line carries `trace_id`/`span_id` when a span is active, so Grafana pivots
log → trace and trace → log in one click.

The SDK is **disabled by default** (no-op provider, no overhead). The Collector
adds a redaction processor as a PII safety net (report §2.5) — phone/OTP/email
attributes are dropped even if one ever slips into a span.

**Verified live** (not just "should work"): the overlay was booted and a trace
pushed end-to-end (app → Collector → Tempo) and read back; a span carrying
`phone`+`otp` came out of Tempo with those attributes **stripped**. See the
verification table in `docs/observability-report.md` §9 for the honest boundary
(what was run live vs. what is covered by unit tests).

## Security note

`/metrics` is currently served on the app's public port. Before production,
restrict it to the monitoring network (internal bind / firewall) — it reveals
internal route names and load. See `docs/code-review.md` §2.5 and the
observability report risks table.
