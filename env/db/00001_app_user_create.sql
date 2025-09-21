-- +goose Up
-- +goose StatementBegin
CREATE TABLE app_user (
    id              SERIAL PRIMARY KEY,
    created_at      TIMESTAMP NOT NULL,
    updated_at      TIMESTAMP NOT NULL,
    username        VARCHAR(128) NOT NULL,
    password        VARCHAR(256) NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE app_user;
-- +goose StatementEnd
