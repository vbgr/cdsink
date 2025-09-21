package randb

import (
	"fmt"
	"slices"

	"github.com/brianvoe/gofakeit/v6"
)

// DbType defines supported database data types.
type DbType int

// TODO: Remove from model because different databases may have different set of types.
const (
	DbTypeUnknown     DbType = 0
	DbTypeBool        DbType = iota
	DbTypeInt8        DbType = iota
	DbTypeInt16       DbType = iota
	DbTypeInt32       DbType = iota
	DbTypeInt64       DbType = iota
	DbTypeUInt8       DbType = iota
	DbTypeUInt16      DbType = iota
	DbTypeUInt32      DbType = iota
	DbTypeUInt64      DbType = iota
	DbTypeFloat32     DbType = iota
	DbTypeFloat64     DbType = iota
	DbTypeDecimal     DbType = iota
	DbTypeTime        DbType = iota
	DbTypeTimeTz      DbType = iota
	DbTypeDate        DbType = iota
	DbTypeTimestamp   DbType = iota
	DbTypeTimestampTz DbType = iota
	DbTypeString      DbType = iota
	DbTypeText        DbType = iota
	DbTypeArray       DbType = iota
	DbTypeObject      DbType = iota
	DbTypeUUID        DbType = iota
	DbTypeEnum        DbType = iota
)

const (
	tableStatusReady        = 0
	tableStatusFkNotScanned = iota
	tableStatusFkNoData     = iota
)

var random func(int, int) int = func(a, b int) int { return gofakeit.Number(a, b-1) }

// Generator generates a column value for a row in a changeset.
type Generator func(int) any

// Domain defines interfacae of generator factory.
type Domain func(int) Generator

// Predictor defines domain predictor interface.
type Predictor interface {
	Predict(int, []*Table) Domain
}

// Enum represents db enum values.
type Enum struct {
	Name   string
	Values []string
}

// Column defines a database table's column.
type Column struct {
	Name      string
	IsPk      bool
	Auto      bool
	Type      DbType
	Size      int
	Default   string
	Nullable  bool
	Precision int
	Scale     int
	Enum      *Enum
	SubType   DbType
}

// ForeignKey contains foreign key information.
type ForeignKey struct {
	Name           string
	Columns        []int
	ForeignTable   *Table
	ForeignColumns []int
}

// Table represents a single database table.
type Table struct {
	Name        string
	Columns     []*Column
	ForeignKeys []*ForeignKey

	sample Sample
}

func (t *Table) status() int {
	for _, fk := range t.ForeignKeys {
		if t.nullable(fk.Columns) {
			continue
		}
		if fk.ForeignTable == nil {
			return tableStatusFkNotScanned
		}
		if len(fk.ForeignTable.sample.Rows) == 0 {
			return tableStatusFkNoData
		}
	}
	return tableStatusReady
}

func (t *Table) nullable(columns []int) bool {
	for _, k := range columns {
		if !t.Columns[k].Nullable {
			return false
		}
	}
	return true
}

func (t *Table) getSampleColumns() []string {
	columns := []string{}
	for _, k := range t.sample.Columns {
		columns = append(columns, t.Columns[k].Name)
	}
	return columns
}

// Sample represents table sample data
type Sample struct {
	Rows        [][]any
	PrimaryKeys []int
	Columns     []int
	ColumnsMap  []int
	DeleteMark  int
}

func (s *Sample) addPk(i int) {
	if s.addColumn(i) {
		s.PrimaryKeys = append(s.PrimaryKeys, i)
	}
}

func (s *Sample) addColumn(i int) bool {
	if s.ColumnsMap[i] != -1 {
		return false
	}

	s.Columns = append(s.Columns, i)
	s.ColumnsMap[i] = len(s.Columns) - 1

	return true
}

func (s *Sample) selectForUpdate() []any {
	if len(s.Rows) == 0 || s.DeleteMark <= 0 {
		return nil
	}
	return s.Rows[random(0, s.DeleteMark)]
}

func (s *Sample) selectForDelete() []any {
	if s.DeleteMark < 0 || s.DeleteMark >= len(s.Rows) {
		return nil
	}
	return s.Rows[random(s.DeleteMark, len(s.Rows))]
}

// Changeset represents a table changes.
type Changeset struct {
	Table  string
	Insert Insert
	Delete Delete
	Update Update
}

type Insert struct {
	Columns []string
	Values  [][]any
}

type Update struct {
	PrimaryKeys []string
	Columns     [][]string
	Values      [][]any
}

type Delete struct {
	PrimaryKeys []string
	Values      [][]any
}

// createSamples creates empty samples for the tables provided.
func createSamples(tables []*Table) {
	for i := range tables {
		tables[i].sample = Sample{ColumnsMap: make([]int, len(tables[i].Columns))}
		for j, c := range tables[i].Columns {
			tables[i].sample.ColumnsMap[j] = -1
			if c.IsPk {
				tables[i].sample.addPk(j)
			}
		}
	}

	for i := range tables {
		for _, fk := range tables[i].ForeignKeys {
			for _, k := range fk.ForeignColumns {
				if fk.ForeignTable == nil {
					continue
				}
				fk.ForeignTable.sample.addColumn(k)
			}
		}
	}
}

// TODO: set better name
type generator struct {
	table       *Table
	factory     func() Generator
	foreignKeys []*ForeignKey
	pkNames     []string
	pkIndex     []int
	columnNames []string
	columnIndex []int
}

func newGenerator(table *Table, domain Domain) *generator {
	g := generator{
		table:       table,
		foreignKeys: make([]*ForeignKey, len(table.Columns)),
	}

	for i, c := range table.Columns {
		if c.IsPk {
			g.pkIndex = append(g.pkIndex, i)
			g.pkNames = append(g.pkNames, c.Name)
		}
		if c.IsPk && !c.Auto || !c.IsPk {
			g.columnIndex = append(g.columnIndex, i)
			g.columnNames = append(g.columnNames, c.Name)
		}
	}

	for _, fk := range table.ForeignKeys {
		for _, c := range fk.Columns {
			g.foreignKeys[c] = fk
		}
	}

	gens := make([]Generator, len(table.Columns))
	for i, c := range table.Columns {
		if c.IsPk && c.Auto {
			gens[i] = noGenerator
			continue
		}
		if g.foreignKeys[i] != nil {
			if gens[i] != nil {
				continue
			}

			fk := fkGenerator(g.foreignKeys[i], table)
			for _, f := range g.foreignKeys[i].Columns {
				gens[f] = fk
			}
			continue
		}
		gens[i] = domain(i)
	}
	g.factory = factory(gens)

	return &g
}

func (g *generator) generate(c Config) Changeset {
	cs := Changeset{
		Table: g.table.Name,

		Insert: Insert{
			Columns: g.columnNames,
		},
		Update: Update{
			PrimaryKeys: g.pkNames,
		},
		Delete: Delete{
			PrimaryKeys: g.pkNames,
		},
	}

	for i := 0; i < c.BatchSize; i++ {
		gen := g.factory()

		v := random(0, 100)
		switch {
		case v < c.Inserts:
			g.insert(&cs, gen)
		case v < c.Inserts+c.Updates:
			g.update(&cs, gen)
		default:
			g.delete(&cs)
		}
	}

	return cs
}

func (g *generator) enumerateForUpdate() []int {
	cols := make([]int, 0, len(g.columnIndex))

	i := 0
	for i < len(g.columnIndex) {
		c := g.columnIndex[i]
		if g.table.Columns[c].IsPk {
			i += 1
			continue
		}
		x := random(0, 3) // TODO: parametrize
		if x != 0 {
			i += 1
			continue
		}
		if g.foreignKeys[c] == nil {
			cols = append(cols, c)
			i += 1
			continue
		}
		for _, f := range g.foreignKeys[c].Columns {
			cols = append(cols, f)
			i += 1
		}
	}

	slices.Sort(cols)
	return cols
}

func (g *generator) insert(cs *Changeset, gen Generator) {
	row := make([]any, 0, len(g.columnIndex))
	for _, c := range g.columnIndex {
		row = append(row, gen(c))
	}
	cs.Insert.Values = append(cs.Insert.Values, row)
}

func (g *generator) update(cs *Changeset, gen Generator) {
	if len(g.table.sample.PrimaryKeys) == 0 {
		return
	}

	col := g.enumerateForUpdate() // TODO: avoid excess allocation
	if len(col) == 0 {
		return
	}

	row := g.table.sample.selectForUpdate()
	if row == nil {
		return
	}

	names := make([]string, 0, len(col))
	values := make([]any, 0, len(col))
	for _, c := range col {
		values = append(values, gen(c))
		names = append(names, g.table.Columns[c].Name)
	}
	for _, c := range g.pkIndex {
		values = append(values, row[g.table.sample.ColumnsMap[c]])
	}

	cs.Update.Columns = append(cs.Update.Columns, names)
	cs.Update.Values = append(cs.Update.Values, values)

	// TODO: add primary keys
}

func (g *generator) delete(cs *Changeset) {
	if len(g.table.sample.PrimaryKeys) == 0 {
		return
	}

	row := g.table.sample.selectForDelete()
	if row == nil {
		return
	}

	pks := make([]any, 0, len(row))
	for _, c := range g.pkIndex {
		pks = append(pks, row[g.table.sample.ColumnsMap[c]])
	}
	cs.Delete.Values = append(cs.Delete.Values, pks)
}

// TODO: avoid stateful generator
func fkGenerator(fk *ForeignKey, t *Table) Generator {
	var index = 0
	var value []any = nil

	return func(n int) any {
		if value == nil || index == len(fk.Columns) {
			index = 0
			value = fk.ForeignTable.sample.selectForUpdate()
		}

		if value == nil {
			return nil
		}

		if n != fk.Columns[index] {
			panic(fmt.Sprintf("generator for fk %s failed: expected index %d but was %d", fk.Name, fk.Columns[index], n))
		}

		i := fk.ForeignColumns[index]
		v := value[fk.ForeignTable.sample.ColumnsMap[i]]

		index += 1
		return v
	}
}

func noGenerator(int) any {
	panic("generator for pk && auto must not be called")
}

func factory(gens []Generator) func() Generator {
	return func() Generator {
		return func(i int) any {
			return gens[i](i)
		}
	}
}
