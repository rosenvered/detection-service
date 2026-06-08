# Detection Service

Classifies customer prompts for sensitive topics (healthcare, finance, legal, hr) using GPT-4.1 via the AIM OpenAI proxy.

## Run

```bash
go run ./cmd/server
```

Optional environment variables:

- `ADDR` — listen address (default `:8080`)
- `DB_PATH` — SQLite file path (default `detection.db`)
- `OPENAI_API_KEY` — API key for the AIM proxy (falls back to the take-home demo key)

## Example

```bash
curl -X POST http://localhost:8080/detect \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"How much will my treatment cost and will insurance cover it?","policy_id":"pol_a1b2c3"}'
```

Response:

```json
{"detected_topics":["healthcare","finance"]}
```

A default policy `pol_a1b2c3` with all four topics enabled is seeded on startup. Every `/detect` call is written to the audit log in SQLite.
