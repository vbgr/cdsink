package query

import (
	"fmt"
	"strings"
)

const Tables = `
SELECT
    c.table_name,
    c.column_name,
    c.data_type,
    c.is_nullable,
    c.character_maximum_length,
    c.numeric_precision,
    c.numeric_scale,
    c.column_default,
    c.udt_name,
    pa.atttypmod
FROM information_schema.columns c
    JOIN pg_class pc on c.table_name = pc.relname
    JOIN pg_attribute pa on pc.oid = pa.attrelid and pa.attname = c.column_name
WHERE table_schema = $1
ORDER BY c.table_name, c.ordinal_position
`
const PrimaryKeys = `
SELECT DISTINCT
    c.relname as table_name,
    a.attname as column_name
FROM pg_class c
    INNER JOIN pg_namespace n ON n.oid = c.relnamespace
    INNER JOIN pg_attribute a ON a.attrelid = c.oid
    LEFT JOIN pg_index i
        ON i.indrelid = c.oid
            AND a.attnum = ANY (i.indkey[0:(i.indnkeyatts - 1)])
WHERE a.attnum > 0
    AND i.indisunique IS TRUE
    AND i.indisprimary IS TRUE
    AND n.nspname = $1
ORDER BY table_name, column_name
`

const ForeignKeys = `
SELECT DISTINCT
    tc.table_name,
    tc.constraint_name,
    kcu.column_name,
    ccu.table_name AS fk_table_name,
    ccu.column_name AS fk_column_name
FROM information_schema.table_constraints AS tc
    JOIN information_schema.key_column_usage AS kcu
        ON tc.constraint_name = kcu.constraint_name
        AND tc.table_schema = kcu.table_schema
    JOIN information_schema.constraint_column_usage AS ccu
        ON ccu.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
    AND tc.table_schema = $1
ORDER BY tc.table_name, tc.constraint_name, kcu.column_name
`

const Enums = `
SELECT DISTINCT
    c.udt_name,
    e.enumlabel
FROM information_schema.columns c
    join pg_catalog.pg_type t on t.typname = c.udt_name
    join pg_enum e on t.oid = e.enumtypid
WHERE c.table_schema = $1
    AND c.data_type ILIKE 'USER-DEFINED'
ORDER BY c.udt_name
`

type Query struct {
	table string
	stmt  string
}

func New(table string) *Query {
	return &Query{table: table}
}

func (q *Query) String() string {
	return q.stmt
}

func (q *Query) Count() *Query {
	q.stmt = fmt.Sprintf("SELECT count(*) FROM %s", q.table)
	return q
}

func (q *Query) Select(columns string) *Query {
	q.stmt = fmt.Sprintf("SELECT %s FROM %s", columns, q.table)
	return q
}

func (q *Query) Offset(v int) *Query {
	q.stmt = fmt.Sprintf("%s OFFSET %d", q.stmt, v)
	return q
}

func (q *Query) Limit(v int) *Query {
	q.stmt = fmt.Sprintf("%s LIMIT %d", q.stmt, v)
	return q
}

func (q *Query) Union(o *Query) *Query {
	if q.stmt == "" {
		return o
	}

	q.stmt = fmt.Sprintf("%s\nUNION\n%s", q.stmt, o.stmt)
	return q
}

func (q *Query) Insert(cols []string) *Query {
	params := []string{}
	for i := range cols {
		params = append(params, fmt.Sprintf("$%d", i))
	}
	q.stmt = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", q.table, strings.Join(cols, ", "), strings.Join(params, ", "))
	return q
}

func (q *Query) Update(pks, cols []string) *Query {
	params := []string{}
	for i := range cols {
		params = append(params, fmt.Sprintf("\"%s\"=$%d", cols[i], i + 1))
	}
	conds := []string{}
	for i := range pks {
		conds = append(conds, fmt.Sprintf("\"%s\"=$%d", pks[i], len(cols) + i + 1))
	}
	q.stmt = fmt.Sprintf("UPDATE %s SET %s WHERE %s", q.table, strings.Join(params, ", "), strings.Join(conds, " AND "))
	return q
}

func (q *Query) Delete(pks []string) *Query {
	conds := []string{}
	for i := range pks {
		conds = append(conds, fmt.Sprintf("%s=$%d", pks[i], i + 1))
	}
	q.stmt = fmt.Sprintf("DELETE FROM %s WHERE %s", q.table, strings.Join(conds, " AND "))
	return q
}
