
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(100) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processing_started_at TIMESTAMPTZ,
    deadline_at TIMESTAMPTZ NOT NULL,
    delivered_at TIMESTAMPTZ
);

-- Add the column to existing databases if it is missing.
ALTER TABLE events
ADD COLUMN IF NOT EXISTS processing_started_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_events_pending
ON events (tenant_id, next_attempt_at, created_at)
WHERE status IN ('pending', 'processing');

CREATE TABLE IF NOT EXISTS delivery_attempts (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status_code INTEGER,
    response_body TEXT,
    error TEXT,
    duration_ms BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_attempts_event
ON delivery_attempts (event_id, attempted_at);
