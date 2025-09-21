package pg

import (
	"context"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"randb/pkg/driver/pg/query"
	"randb/pkg/randb"
)

func isauto(v any) bool {
	if v == nil {
		return false
	}
	s := v.(string)
	if strings.HasPrefix(s, "next") {
		return true
	}
	if strings.HasPrefix(s, "uuid_generate_v4()") {
		return true
	}
	return false
}

func pgbool(v any) bool {
	if v == nil {
		return false
	}
	switch v.(string) {
	case "true", "TRUE", "yes", "YES", "on", "ON", "1":
		return true
	default:
		return false
	}
}

func pgint(v any) int {
	if v == nil {
		return 0
	}
	return int(v.(int32))
}

func createTable(name string) *randb.Table {
	return &randb.Table{Name: name}
}

func createForeignKey(name string, reference *randb.Table) *randb.ForeignKey {
	return &randb.ForeignKey{
		Name:         name,
		ForeignTable: reference,
	}
}

type dbScanner struct {
	ctx    context.Context
	conn   *pgxpool.Conn
	schema string
	tabs   []*randb.Table
	tmap   map[string]*randb.Table
	cmap   map[string]map[string]int
	enums  map[string]*randb.Enum
}

func newScanner(ctx context.Context, schema string, conn *pgxpool.Conn) *dbScanner {
	return &dbScanner{
		ctx:    ctx,
		conn:   conn,
		schema: schema,
		tabs:   []*randb.Table{},
		tmap:   map[string]*randb.Table{},
		cmap:   map[string]map[string]int{},
		enums:  map[string]*randb.Enum{},
	}
}

func (s *dbScanner) createColumn(row []any) *randb.Column {
	c := &randb.Column{
		Name:      row[1].(string),
		Auto:      isauto(row[7]),
		Type:      s.pgtype(row[2].(string), row[8].(string)),
		Nullable:  pgbool(row[3]),
		Size:      pgint(row[4]),
		Precision: pgint(row[5]),
		Scale:     pgint(row[6]),
	}
	udtName := row[8].(string)
	if c.Type == randb.DbTypeEnum {
		c.Enum = s.enums[udtName]
	} else {
		c.SubType = s.pgtype(udtName[1:], "")
		if c.SubType == randb.DbTypeString {
			c.Size = pgint(row[9]) - 4
		}
	}
	return c
}

// https://www.postgresql.org/docs/current/datatype.html#DATATYPE-TABLE
func (s *dbScanner) pgtype(tn, un string) randb.DbType {
	switch tn {
	case "boolean", "bool":
		return randb.DbTypeBool
	case "smallint", "int2", "smallserial", "serial2":
		return randb.DbTypeInt16
	case "integer", "int", "int4", "serial", "serial4":
		return randb.DbTypeInt32
	case "bigint", "int8":
		return randb.DbTypeInt64
	case "real", "float4":
		return randb.DbTypeFloat32
	case "double precision", "float8":
		return randb.DbTypeFloat64
	case "numeric", "decimal":
		return randb.DbTypeDecimal
	case "time", "time without time zone":
		return randb.DbTypeTime
	case "timetz", "time with time zone":
		return randb.DbTypeTimeTz
	case "date":
		return randb.DbTypeDate
	case "timestamp without time zone":
		return randb.DbTypeTimestamp
	case "timestamptz", "timestamp with time zone":
		return randb.DbTypeTimestampTz
	case "character", "character varying", "char", "varchar":
		return randb.DbTypeString
	case "text":
		return randb.DbTypeText
	case "json", "jsonb":
		return randb.DbTypeObject
	case "uuid":
		return randb.DbTypeUUID
	case "USER-DEFINED":
		if _, ok := s.enums[un]; ok {
			return randb.DbTypeEnum
		}
		return randb.DbTypeUnknown
	case "ARRAY", "array":
		return randb.DbTypeArray
	default:
		return randb.DbTypeUnknown
	}
}

func (s *dbScanner) scanenums() error {
	var enum *randb.Enum

	return queryRows(s.ctx, s.conn, query.Enums, []any{s.schema}, func(row []any) error {
		name := row[0].(string)
		if enum == nil || enum.Name != name {
			enum = &randb.Enum{Name: name}
			s.enums[enum.Name] = enum

			slog.Debug("scanned enum", "enum", name)
		}

		enum.Values = append(enum.Values, row[1].(string))
		return nil
	})
}

func (s *dbScanner) scantabs() error {
	var table *randb.Table

	return queryRows(s.ctx, s.conn, query.Tables, []any{s.schema}, func(row []any) error {
		tname := row[0].(string)
		if table == nil || tname != table.Name {
			table = createTable(tname)
			s.tabs = append(s.tabs, table)
			s.tmap[tname] = table

			slog.Debug("scanned table", "table", table.Name)
		}

		column := s.createColumn(row)
		table.Columns = append(table.Columns, column)

		slog.Debug("scanned column", "table", table.Name, "column", column.Name)

		cmap, ok := s.cmap[table.Name]
		if !ok {
			cmap = map[string]int{}
			s.cmap[table.Name] = cmap
		}

		cmap[column.Name] = len(table.Columns) - 1
		return nil
	})
}

func (s *dbScanner) scanpks() error {
	var table *randb.Table

	return queryRows(s.ctx, s.conn, query.PrimaryKeys, []any{s.schema}, func(row []any) error {
		tname := row[0].(string)
		if table == nil || tname != table.Name {
			table = s.tmap[tname]
		}

		cname := row[1].(string)
		index := s.cmap[tname][cname]
		table.Columns[index].IsPk = true

		return nil
	})
}

func (s *dbScanner) scanfks() error {
	var table *randb.Table
	var foreignKey *randb.ForeignKey

	return queryRows(s.ctx, s.conn, query.ForeignKeys, []any{s.schema}, func(row []any) error {
		tname := row[0].(string)
		fkname := row[1].(string)
		fktable := row[3].(string)

		if table == nil || fkname != foreignKey.Name {
			foreignKey = createForeignKey(fkname, s.tmap[fktable])
			table = s.tmap[tname]
			table.ForeignKeys = append(table.ForeignKeys, foreignKey)
			slog.Debug("scanned foreign key", "table", table.Name, "fk", fkname)
		}

		cname := row[2].(string)
		fkcolumn := row[4].(string)

		foreignKey.Columns = append(foreignKey.Columns, s.cmap[tname][cname])
		foreignKey.ForeignColumns = append(foreignKey.ForeignColumns, s.cmap[fktable][fkcolumn])

		slog.Debug("scanned foreign key column", "fk", foreignKey.Name, "column", cname)
		return nil
	})
}
