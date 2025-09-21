package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"randb/pkg/driver/pg/query"
	"randb/pkg/randb"
)


// TODO: use schema
func sampletab(ctx context.Context, conn *pgxpool.Conn, chunker randb.Chunker, table string, columns []string) ([][]any, error) {
	var count int
	if err := conn.QueryRow(ctx, query.New(table).Count().String()).Scan(&count); err != nil {
		return nil, fmt.Errorf("failed to get rows count: %w", err)
	}

	q := query.New(table)
	for _, c := range chunker.Chunk(count) {
		q = q.Union(query.New(table).Select(strings.Join(columns, ", ")).Offset(c.Offset).Limit(c.Limit))
	}

	rows := [][]any{}
	err := queryRows(ctx, conn, q.String(), []any{}, func(row []any) error {
		rows = append(rows, row)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query table rows: %w", err)
	}

	return rows, nil
}
