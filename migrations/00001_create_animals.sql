-- +goose Up
CREATE TABLE animals (
    id         BIGSERIAL   PRIMARY KEY,
    data       JSONB       NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX animals_data_gin ON animals USING GIN (data);

-- +goose Down
DROP INDEX IF EXISTS animals_data_gin;
DROP TABLE IF EXISTS animals;
