CREATE TABLE IF NOT EXISTS files (
    file_id text PRIMARY KEY,
    owner_id text NOT NULL,
    filename text NOT NULL,
    extension text NOT NULL,
    content_type text NOT NULL,
    bucket text NOT NULL,
    storage_path text NOT NULL,
    size_bytes bigint NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS processed_events (
    event_id text PRIMARY KEY,
    event_type text NOT NULL,
    source_service text NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id text NOT NULL,
    aggregate_version bigint NOT NULL,
    processed_at timestamptz NOT NULL DEFAULT now()
);
