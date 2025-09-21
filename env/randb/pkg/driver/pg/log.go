package pg

import (
	"log/slog"
	"context"

	"github.com/jackc/pgx/v5"
)

type tracer slog.Logger

func (t *tracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	(*slog.Logger)(t).With("sql", data.SQL).With("args", data.Args).Debug("")
	return ctx
}

func (t *tracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
}
