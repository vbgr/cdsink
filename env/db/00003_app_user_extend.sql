-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_user ADD COLUMN last_login timestamp;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_user DROP COLUMN last_login;
-- +goose StatementEnd
