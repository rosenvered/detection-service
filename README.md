# Detection Service

Classifies customer prompts for sensitive topics (healthcare, finance, legal, hr) using a hybrid keyword + GPT-4.1 classifier via the AIM OpenAI proxy.

## Run

```bash
go run ./cmd/server
```

Optional environment variables:

- `ADDR` — listen address (default `:8080`)
- `DB_PATH` — SQLite file path (default `detection.db`)
- `OPENAI_API_KEY` — API key for the AIM proxy (falls back to the take-home demo key)

## Endpoints

### POST /detect — full classification

Returns all detected topics (keyword ∪ LLM), filtered by policy.

```bash
curl -X POST http://localhost:8080/detect \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"How much will my treatment cost and will insurance cover it?","policy_id":"pol_a1b2c3"}'
```

### POST /protect — fail-fast

Keyword scan first; LLM only if no keyword hit. Returns the first detected topic.

```bash
curl -X POST http://localhost:8080/protect \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"Please summarize this medical record...","policy_id":"pol_a1b2c3"}'
```

### /policies — CRUD

```bash
# List
curl http://localhost:8080/policies

# Create
curl -X POST http://localhost:8080/policies \
  -H 'Content-Type: application/json' \
  -d '{"enabled_topics":["healthcare","finance"]}'

# Get / Update / Delete
curl http://localhost:8080/policies/pol_a1b2c3
curl -X PUT http://localhost:8080/policies/pol_a1b2c3 \
  -H 'Content-Type: application/json' \
  -d '{"enabled_topics":["healthcare"]}'
curl -X DELETE http://localhost:8080/policies/pol_a1b2c3
```

### GET /audit — query call history

Every `/detect` and `/protect` call is logged automatically. Query with optional filters:

```bash
curl "http://localhost:8080/audit?policy_id=pol_a1b2c3&endpoint=protect&limit=10&offset=0"
```

Query params: `policy_id`, `endpoint` (`detect` or `protect`), `from` / `to` (RFC3339), `limit` (default 50, max 100), `offset` (default 0).

A default policy `pol_a1b2c3` with all four topics enabled is seeded on startup.
