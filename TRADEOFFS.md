# Tradeoffs & Next Steps

## Hybrid vs pure LLM

**Chosen:** Keyword matcher + LLM fallback.

Keywords are fast and deterministic but miss nuance and can false-positive. LLMs catch context but are slow (200ms–2s) and inconsistent. The split: union both on `/detect` for completeness; keyword-first on `/protect` for latency.

**Alternative considered:** Pure LLM for both endpoints — simpler, but `/protect` would not meet inline pipeline latency needs.

## Go vs Python

**Chosen:** Go with Gin.

Strong latency story for `/protect`, explicit concurrency model, single binary deployment. Python would match the take-home's OpenAI snippet verbatim and iterate faster on prompt engineering, but Go better demonstrates systems thinking for a detection pipeline service.

## SQLite vs Postgres

**Chosen:** SQLite (`modernc.org/sqlite`, pure Go, no CGO).

Zero ops overhead, survives restarts, fine for demo and single-node deployment. Audit history matters for a governance product — in-memory would lose data on restart.

**Next step:** Postgres when multi-instance or high write volume requires it.

## Naive keywords vs ML

**Chosen:** Curated per-topic keyword lists.

Easy to build, explain in an interview, and extend. Production would need managed word lists, customer-specific overrides, and periodic review for false positives.

## No authentication

Acceptable for take-home scope. First production addition would be API keys with per-tenant policy isolation.

## Test strategy

**Chosen:** `StubClassifier` for offline unit and HTTP integration tests. No test hits the real LLM proxy — fast, deterministic CI.

**Gap:** No live LLM integration test. A gated test (`INTEGRATION_LLM=1`) against the proxy would catch prompt/regression issues before deploy.

## Topic ordering

Detected topics are returned in keyword-priority order (healthcare → finance → legal → hr), then LLM order for remaining hits. LLM order is not stable across calls even at `temperature: 0`. Sorting alphabetically or by severity before returning would improve client consistency.

## What I'd do next (priority order)

1. **Auth + tenant scoping** — required before any customer deployment
2. **Sorted, stable response ordering** — cheap win for API consumers
3. **Prometheus metrics** — keyword hit rate, LLM latency p99, error rate
4. **Circuit breaker** on LLM proxy — graceful degradation under outage
5. **PII redaction** in audit log storage — prompts may contain sensitive content
