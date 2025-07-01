create extension timescaledb;

CREATE TABLE workers (
    time TIMESTAMPTZ DEFAULT now(),
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT,             -- Type of worker (e.g., "bot", "monitor", etc.)
    info JSONB             -- Additional metadata (e.g., worker config)
);

-- Add indexes for faster lookups (if necessary)
CREATE INDEX idx_workers_name ON workers(name);
CREATE INDEX idx_workers_last_heartbeat ON workers(time);

CREATE TYPE worker_status AS ENUM (
    'starting',
    'healthy',
    'working',
    'idle',
    'error',
    'unreachable',
    'crashed',
    'stopped',
    'disabled'
    );

CREATE TABLE worker_updates (
    time TIMESTAMPTZ DEFAULT now(),
    worker_id UUID REFERENCES workers(id) ON DELETE CASCADE,
    status worker_status NOT NULL,
    message TEXT
);

-- Convert this table into a TimescaleDB hypertable for time-series data
SELECT create_hypertable('worker_updates', 'time');

-- Create the worker heartbeats table
CREATE TABLE worker_heartbeats (
    time TIMESTAMPTZ DEFAULT now(),
    worker_id UUID REFERENCES workers(id) ON DELETE CASCADE,
    status worker_status NOT NULL
);

-- Convert it into a hypertable for time-series data (by heartbeat_time)
SELECT create_hypertable('worker_heartbeats', 'time');
