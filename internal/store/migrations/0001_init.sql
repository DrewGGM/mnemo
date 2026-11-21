-- vector needs a superuser: deploy/initdb/01-roles.sql creates it, and here
-- it only fails if that step was skipped. pg_trgm is trusted.
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE tenants (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL UNIQUE,
    fts_config regconfig NOT NULL DEFAULT 'english',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subjects (
    tenant_id  uuid NOT NULL REFERENCES tenants (id),
    id         text NOT NULL,
    policy     jsonb NOT NULL DEFAULT '{}',
    purged_at  timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id)
);

CREATE TABLE episodes (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL,
    subject_id  text NOT NULL,
    session_id  text NOT NULL,
    seq         int NOT NULL,
    role        text NOT NULL CHECK (role IN ('user', 'assistant', 'tool', 'system', 'summary')),
    content     text NOT NULL CHECK (octet_length(content) <= 32768),
    occurred_at timestamptz NOT NULL,
    recorded_at timestamptz NOT NULL DEFAULT now(),
    trust       text NOT NULL CHECK (trust IN ('user', 'assistant', 'untrusted')),
    embedding   vector(1024),
    fts         tsvector,
    token_count int,
    api_key_id  uuid,
    UNIQUE (tenant_id, subject_id, session_id, seq),
    FOREIGN KEY (tenant_id, subject_id) REFERENCES subjects (tenant_id, id)
);

CREATE INDEX episodes_subject_occurred ON episodes (tenant_id, subject_id, occurred_at);
CREATE INDEX episodes_fts ON episodes USING gin (fts);

-- Table by table, never ALL: TRUNCATE skips row level security and TRIGGER
-- would let the service role attach code to a table (ADR-0007, decision 2).
-- The service reads tenants.fts_config to build and query tsvector values.
GRANT SELECT ON schema_migrations, tenants TO mnemo_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON subjects, episodes TO mnemo_app;

-- Development tenant used until authentication exists (T-012).
INSERT INTO tenants (id, name) VALUES ('00000000-0000-0000-0000-000000000001', 'dev');
