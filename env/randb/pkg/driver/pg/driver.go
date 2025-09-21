package pg

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/asaskevich/govalidator"
	"github.com/jackc/pgx/v5/pgxpool"

	"randb/pkg/driver"
	"randb/pkg/randb"
)

func queryRows(ctx context.Context, conn *pgxpool.Conn, sql string, args []any, fn func(row []any) error) error {
	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("faild to execute query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		row, err := rows.Values()
		if err != nil {
			return fmt.Errorf("failed to read row: %w", err)
		}

		if err := fn(row); err != nil {
			return err
		}
	}

	return nil
}

func New(ctx context.Context, cfg driver.Config) (randb.Driver, error) {
	if _, err := govalidator.ValidateStruct(cfg); err != nil {
		return nil, err
	}

	config, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	config.ConnConfig.Tracer = (*tracer)(slog.Default())
	config.MaxConns = int32(cfg.MaxConn)

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	driver := pgdriver{
		ctx:    ctx,
		cfg:    cfg,
		schema: cfg.Schema,
		pool:   pool,
	}

	return &driver, nil
}

type pgdriver struct {
	ctx     context.Context
	cfg     driver.Config
	schema  string
	pool    *pgxpool.Pool
}

func (d *pgdriver) Close() error {
	d.pool.Close()
	return nil
}

func (d *pgdriver) Scan() ([]*randb.Table, error) {
	slog.Debug("acquiring connection")
	conn, err := d.pool.Acquire(d.ctx)
	if err != nil {
		slog.Error("unable to acquire connection from pool", "error", err)
		return nil, err
	}
	defer conn.Release()

	scanner := newScanner(d.ctx, d.schema, conn)

	slog.Debug("scanning enums")
	if err := scanner.scanenums(); err != nil {
		return nil, fmt.Errorf("failed to scan enums: %w", err)
	}

	slog.Debug("scanning tables")
	if err := scanner.scantabs(); err != nil {
		return nil, fmt.Errorf("failed to scan tables: %w", err)
	}

	slog.Debug("scanning primary keys")
	if err := scanner.scanpks(); err != nil {
		return nil, fmt.Errorf("failed to scan primary keys: %w", err)
	}

	slog.Debug("scanning foreign keys")
	if err := scanner.scanfks(); err != nil {
		return nil, fmt.Errorf("failed to scan foreign keys: %w", err)
	}

	return scanner.tabs, nil
}

func (d *pgdriver) Select(chunker randb.Chunker, table string, columns []string) ([][]any, error) {
	conn, err := d.pool.Acquire(d.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	return sampletab(d.ctx, conn, chunker, table, columns)
}

func (d *pgdriver) Apply(changeset randb.Changeset) error {
	conn, err := d.pool.Acquire(d.ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	return apply(d.ctx, conn, changeset)
}
