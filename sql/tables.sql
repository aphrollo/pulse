create extension timescaledb;

CREATE TABLE agents (
    time TIMESTAMPTZ DEFAULT now(),
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT,             -- Type of agent (e.g., "bot", "monitor", etc.)
    info JSONB             -- Additional metadata (e.g., agent config)
);

-- Add indexes for faster lookups (if necessary)
CREATE INDEX idx_agents_name ON agents(name);
CREATE INDEX idx_agents_last_heartbeat ON agents(time);

CREATE TYPE agent_state AS ENUM (
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

CREATE TABLE agent_updates (
    time TIMESTAMPTZ DEFAULT now(),
    agent_id UUID REFERENCES agents(id) ON DELETE CASCADE,
    status agent_state NOT NULL,
    message TEXT
);

-- Convert this table into a TimescaleDB hypertable for time-series data
SELECT create_hypertable('agent_updates', 'time');

-- Create the agent heartbeats table
CREATE TABLE agent_heartbeats (
    time TIMESTAMPTZ DEFAULT now(),
    agent_id UUID REFERENCES agents(id) ON DELETE CASCADE,
    status agent_state NOT NULL
);

-- Convert it into a hypertable for time-series data (by heartbeat_time)
SELECT create_hypertable('agent_heartbeats', 'time');
