CREATE TABLE IF NOT EXISTS metrics (
    id TEXT NOT NULL,
    type TEXT NOT NULL,
    total BIGINT,
    value DOUBLE PRECISION,
    PRIMARY KEY (id, type),
    CHECK (type IN ('gauge', 'counter'))
);
