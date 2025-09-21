-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_user ADD COLUMN birthdate DATE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_user DROP COLUMN birthdate;
-- +goose StatementEnd
