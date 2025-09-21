-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_lineitem ADD COLUMN tags varchar(12)[];
ALTER TABLE app_lineitem ADD COLUMN bids int[];
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_lineitem DROP COLUMN tags;
ALTER TABLE app_lineitem DROP COLUMN bids;
-- +goose StatementEnd
