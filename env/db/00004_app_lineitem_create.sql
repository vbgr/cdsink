-- +goose Up
-- +goose StatementBegin
CREATE TABLE app_lineitem (
    id              SERIAL PRIMARY KEY,
    created_at      TIMESTAMP NOT NULL,
    updated_at      TIMESTAMP NOT NULL,
    code            VARCHAR(16) NOT NULL,
    price           INTEGER NOT NULL,
    description     TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE app_lineitem;
-- +goose StatementEnd
