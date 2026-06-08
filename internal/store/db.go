package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS policies (
    id             TEXT PRIMARY KEY,
    enabled_topics TEXT NOT NULL,
    created_at     TEXT NOT NULL,
    updated_at     TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS audit_log (
    id              TEXT PRIMARY KEY,
    timestamp       TEXT NOT NULL,
    endpoint        TEXT NOT NULL,
    prompt          TEXT NOT NULL,
    policy_id       TEXT NOT NULL,
    detected_topics TEXT NOT NULL,
    method          TEXT NOT NULL,
    latency_ms      INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_policy    ON audit_log(policy_id);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_endpoint  ON audit_log(endpoint);
`

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	return db, nil
}
