# Detection Service — Design & Planning

## Overview

Classifies customer prompts for sensitive topics (`healthcare`, `finance`, `legal`, `hr`). Each request references a **policy** defining enabled detections. Built in Go with Gin, SQLite, and GPT-4.1 via the AIM OpenAI proxy.

## Architecture

```
HTTP API (Gin)                    Classification Service
POST /detect, /protect     →      PolicyGate → KeywordMatcher → LLMClassifier
/policies CRUD, GET /audit              ↓              ↓
                                   SQLite          AIM OpenAI Proxy
                              (policies, audit_log)   (gpt-4.1)
```

| Component | Role |
|-----------|------|
| Handlers | JSON validation, routing, error → HTTP status |
| Service | Keyword + LLM orchestration, audit writes |
| KeywordMatcher | Sub-ms deterministic scan per topic |
| LLMClassifier | GPT-4.1 with JSON parse, regex fallback, retry |
| PolicyStore / AuditStore | SQLite CRUD and append-only query |

## Endpoints

**POST /detect** — Returns all topics. Unions keyword hits with LLM results, filtered by policy `enabled_topics`. Audits with `method`: `keyword`, `llm`, or `hybrid`.

**POST /protect** — Fail-fast. Keyword `FirstMatch` first (~0ms); LLM `ClassifyOne` fallback. Returns one topic. Audits every call.

**/policies** — CRUD on `{id, enabled_topics}`. Default `pol_a1b2c3` seeded at startup.

**GET /audit** — Query auto-logged records. Filters: `policy_id`, `endpoint`, `from`/`to`, `limit` (max 100), `offset`.

## Extra capability: hybrid fast-path

`/protect` is inline in customer pipelines where latency matters. Keywords skip LLM round-trips (200ms–2s) for obvious prompts. `/detect` unions both signals for completeness. Tradeoff: keywords can false-positive; LLM catches nuance but is slow.

## LLM reliability

1. `temperature: 0`, strict JSON prompt
2. Extract JSON from prose/markdown wrappers
3. Regex fallback for topic names
4. Single retry with stricter prompt
5. Filter to valid topics ∩ policy enabled set
6. 502 if parsing fails entirely

## Data model

- **Policy:** `id`, `enabled_topics[]`, timestamps
- **Audit:** `id`, `timestamp`, `endpoint`, `prompt`, `policy_id`, `detected_topics[]`, `method`, `latency_ms`

## Implementation status

| # | Delivered |
|---|-----------|
| 1 | `/detect`, LLM classifier, SQLite, auto-audit, seed policy |
| 2 | Keywords, `/protect`, `/policies` CRUD |
| 3 | `GET /audit`, validation, tests (stub LLM, in-memory SQLite) |
| 4 | DESIGN.md, TRADEOFFS.md, README |

## Future work

### Auth & tenant isolation
API keys per customer; policies and audit records scoped by tenant. Required before any real deployment.

### Postgres & horizontal scale
SQLite is single-node. Postgres supports concurrent writers, replication, and connection pooling when multiple service instances run behind a load balancer.

### Redis result cache
Cache classification results by `hash(policy_id + prompt)`, not policies (policies are small and rarely change). Avoids repeated LLM calls for identical or templated prompts — the main source of cost and latency at scale. TTL-based expiry (e.g. 5–60 min); cache only after a successful classification.

### Embedding / fine-tuned classifier
Replace or supplement naive LLM calls with embedding similarity or a per-customer fine-tuned model for higher accuracy and lower per-request cost.

### LLM streaming
Today the service blocks on the full Chat Completion response. Streaming (`stream: true`) delivers tokens incrementally; for `/protect` LLM fallback, return as soon as a parseable topic appears instead of waiting for complete JSON. Latency win on the slow path only — keyword hits are already sub-ms.

### Circuit breaker on LLM proxy
After repeated proxy failures, stop calling the LLM for a cooldown period and fail fast (immediate 503/502). Prevents cascading timeouts when the proxy is down. `/protect` can degrade to keywords-only; `/detect` returns an error immediately. The keyword layer is already a partial degradation strategy; a circuit breaker formalizes it for the LLM dependency.

### Per-topic policy actions
Policies today only list `enabled_topics`. Production governance needs per-topic enforcement, e.g. `healthcare → block`, `finance → redact`, `legal → alert`, `hr → log_only`. Moves the service from "what was detected" to "what to do about it" without pushing that logic to every customer pipeline.

### PII redaction in audit logs
Audit records store full prompts, which may contain PHI or financial data. Redact or hash prompt content at write time; retain enough for compliance queries.

### Observability & determinism
Prometheus metrics: p50/p99 latency, keyword hit rate, LLM error rate, cache hit rate. Sort `detected_topics` before returning — LLM order varies across calls even at `temperature: 0`.
