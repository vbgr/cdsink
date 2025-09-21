package pg

import (
	"context"
	"log/slog"
	"time"

	"randb/pkg/driver/pg/query"
	"randb/pkg/randb"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TODO: use schema
func apply(ctx context.Context, conn *pgxpool.Conn, cs randb.Changeset) error {
	slog.Debug("applying changeset", "table", cs.Table, "changes", cs)

	var t time.Time

	t = time.Now()
	num, err := insert(ctx, conn, cs.Table, cs.Insert)
	if err != nil {
		slog.Error("failed to do insert", "table", cs.Table, "error", err)
	} else {
		slog.Info("inserted rows", "table", cs.Table, "count", num, "in", time.Now().Sub(t))
	}

	t = time.Now()
	if err := update(ctx, conn, cs.Table, cs.Update); err != nil {
		slog.Error("failed to do update", "table", cs.Table, "error", err)
	} else {
		slog.Info("updated rows", "table", cs.Table, "count", len(cs.Update.Values), "in", time.Now().Sub(t))
	}

	t = time.Now()
	if err := delete(ctx, conn, cs.Table, cs.Delete); err != nil {
		slog.Error("failed to do delete", "table", cs.Table, "error", err)
	} else {
		slog.Info("deleted rows", "table", cs.Table, "count", len(cs.Delete.Values), "in", time.Now().Sub(t))
	}

	// TODO: return errors
	return nil
}

// TODO: use schema
func insert(ctx context.Context, conn *pgxpool.Conn, table string, cs randb.Insert) (int64, error) {
	return conn.CopyFrom(ctx, pgx.Identifier{table}, cs.Columns, pgx.CopyFromRows(cs.Values))
}


// TODO: use schema
func update(ctx context.Context, conn *pgxpool.Conn, table string, cs randb.Update) error {
	if len(cs.Values) == 0 {
		slog.Debug("no update values generated", "table", table)
		return nil
	}

	batch := &pgx.Batch{}
	for i := range cs.Values {
		q := query.New(table).Update(cs.PrimaryKeys, cs.Columns[i]).String()
		slog.Debug("batch queue update query", "sql", q, "pks", cs.PrimaryKeys, "columns", cs.Columns[i], "params", cs.Values[i])
		batch.Queue(q, cs.Values[i]...)
	}

	res := conn.SendBatch(ctx, batch)
	defer res.Close()

	var err error
	for i := range cs.Values {
		_, e := res.Exec()
		if e != nil {
			slog.Error("update failed", "table", table, "pks", cs.PrimaryKeys, "columns", cs.Columns[i], "params", cs.Values[i], "err", e)
			err = e
		}
	}

	return err
}

// TODO: use schema
func delete(ctx context.Context, conn *pgxpool.Conn, table string, cs randb.Delete) error {
	if len(cs.Values) == 0 {
		slog.Debug("no delete values generated", "table", table)
		return nil
	}

	q := query.New(table).Delete(cs.PrimaryKeys).String()

	batch := &pgx.Batch{}
	for i := range cs.Values {
		slog.Debug("batch queue delete query", "sql", q, "params", cs.Values[i])
		batch.Queue(q, cs.Values[i]...)
	}

	res := conn.SendBatch(ctx, batch)
	defer res.Close()

	var err error
	for i := range cs.Values {
		_, e := res.Exec()
		if e != nil {
			slog.Error("delete failed", "table", table, "pks", cs.PrimaryKeys, "params", cs.Values[i], "err", e)
			err = e
		}
	}

	return err
}
