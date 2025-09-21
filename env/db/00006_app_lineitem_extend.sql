-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_lineitem ADD COLUMN price_usd DECIMAL(10, 4);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_lineitem DROP COLUMN price_usd;
-- +goose StatementEnd
