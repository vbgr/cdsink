-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_lineitem ADD COLUMN metadata JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_lineitem DROP COLUMN metadata;
-- +goose StatementEnd
