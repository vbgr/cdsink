package randb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTableStatus(t *testing.T) {
	const (
		tsProduct      = 0
		tsUser         = iota
		tsOrganization = iota
	)

	getFixtures := func() []*Table {
		fixtures := []*Table{
			tsProduct: {
				Name: "ts_product",
				Columns: []*Column{
					{
						Name: "code",
						IsPk: true,
						Type: DbTypeString,
					},
					{
						Name: "is_virtual",
						Type: DbTypeBool,
					},
				},
			},
			tsUser: {
				Name: "ts_user",
				Columns: []*Column{
					{
						Name: "id",
						IsPk: true,
						Auto: true,
						Type: DbTypeInt32,
					},
					{
						Name: "username",
						Type: DbTypeString,
					},
					{
						Name: "password",
						Type: DbTypeString,
					},
				},
			},
			tsOrganization: {
				Name: "ts_organization",
				Columns: []*Column{
					{
						Name: "id",
						IsPk: true,
						Auto: true,
						Type: DbTypeInt32,
					},
					{
						Name: "name",
						Type: DbTypeString,
					},
					{
						Name: "created_by",
						Type: DbTypeInt32,
					},
					{
						Name: "product_id",
						Type: DbTypeString,
					},
				},
			},
		}

		fixtures[tsOrganization].ForeignKeys = []*ForeignKey{
			{
				Columns:        []int{2},
				ForeignTable:   fixtures[tsUser],
				ForeignColumns: []int{0},
			},
			{
				Columns:        []int{3},
				ForeignTable:   fixtures[tsProduct],
				ForeignColumns: []int{0},
			},
		}

		return fixtures
	}

	t.Run("Ready", func(t *testing.T) {
		fixtures := getFixtures()
		fixtures[tsProduct].sample = Sample{Rows: [][]any{{1}}}
		fixtures[tsUser].sample = Sample{Rows: [][]any{{1}}}

		require.Equal(t, tableStatusReady, fixtures[tsOrganization].status())
	})

	t.Run("FkNotScanned", func(t *testing.T) {
		fixtures := getFixtures()
		fixtures[tsOrganization].ForeignKeys[0].ForeignTable = nil
		require.Equal(t, tableStatusFkNotScanned, fixtures[tsOrganization].status())
	})

	t.Run("FkNoData", func(t *testing.T) {
		fixtures := getFixtures()
		fixtures[tsProduct].sample = Sample{}
		fixtures[tsUser].sample = Sample{}
		require.Equal(t, tableStatusFkNoData, fixtures[tsOrganization].status())
	})
}

func TestTableNullable(t *testing.T) {
	table := &Table{
		Name: "ts_organization",
		Columns: []*Column{
			{
				Name: "id",
				IsPk: true,
				Auto: true,
				Type: DbTypeInt32,
			},
			{
				Name:     "name",
				Type:     DbTypeString,
				Nullable: true,
			},
			{
				Name: "created_by",
				Type: DbTypeInt32,
			},
			{
				Name:     "product_id",
				Type:     DbTypeString,
				Nullable: true,
			},
		},
	}

	assert := require.New(t)

	assert.True(table.nullable([]int{1, 3}))
	assert.False(table.nullable([]int{1, 2}))
}

func TestTableGetSampleColumns(t *testing.T) {
	table := &Table{
		Name: "ts_organization",
		Columns: []*Column{
			{
				Name: "id",
				IsPk: true,
				Auto: true,
				Type: DbTypeInt32,
			},
			{
				Name:     "name",
				Type:     DbTypeString,
				Nullable: true,
			},
			{
				Name: "created_by",
				Type: DbTypeInt32,
			},
			{
				Name:     "product_id",
				Type:     DbTypeString,
				Nullable: true,
			},
		},
	}

	table.sample.Columns = []int{0, 2, 3}
	exp := []string{
		table.Columns[0].Name,
		table.Columns[2].Name,
		table.Columns[3].Name,
	}
	act := table.getSampleColumns()

	require.Equal(t, exp, act)
}

func TestCreateSamples(t *testing.T) {
	const (
		tsProduct      = 0
		tsUser         = iota
		tsOrganization = iota
		tsRole         = iota
		tsUserRole     = iota
		tsUserProfile  = iota
	)

	fixtures := []*Table{
		tsProduct: {
			Name: "ts_product",
			Columns: []*Column{
				{
					Name: "code",
					IsPk: true,
					Type: DbTypeString,
				},
				{
					Name: "is_virtual",
					Type: DbTypeBool,
				},
			},
		},
		tsUser: {
			Name: "ts_user",
			Columns: []*Column{
				{
					Name: "id",
					IsPk: true,
					Auto: true,
					Type: DbTypeInt32,
				},
				{
					Name: "username",
					Type: DbTypeString,
				},
				{
					Name: "password",
					Type: DbTypeString,
				},
			},
		},
		tsOrganization: {
			Name: "ts_organization",
			Columns: []*Column{
				{
					Name: "id",
					IsPk: true,
					Auto: true,
					Type: DbTypeInt32,
				},
				{
					Name: "name",
					Type: DbTypeString,
				},
				{
					Name: "created_by",
					Type: DbTypeInt32,
				},
			},
		},
		tsRole: {
			Name: "ts_role",
			Columns: []*Column{
				{
					Name: "product_id",
					Type: DbTypeString,
					IsPk: true,
				},
				{
					Name: "organization_id",
					Type: DbTypeInt32,
					IsPk: true,
				},
				{
					Name: "name",
					Type: DbTypeString,
				},
			},
		},
		tsUserRole: {
			Name: "ts_user_role",
			Columns: []*Column{
				{
					Name: "user_id",
					Type: DbTypeInt32,
					IsPk: true,
				},
				{
					Name: "role_id",
					Type: DbTypeInt32,
					IsPk: true,
				},
				{
					Name: "assigned_by",
					Type: DbTypeInt32,
				},
			},
		},
		tsUserProfile: {
			Name: "ts_user_profile",
			Columns: []*Column{
				{
					Name: "username",
					Type: DbTypeString,
					IsPk: true,
				},
				{
					Name: "details",
					Type: DbTypeObject,
				},
			},
		},
	}

	fixtures[tsOrganization].ForeignKeys = []*ForeignKey{
		{
			Columns:        []int{2},
			ForeignTable:   fixtures[tsUser],
			ForeignColumns: []int{0},
		},
	}

	fixtures[tsRole].ForeignKeys = []*ForeignKey{
		{
			Columns:        []int{0},
			ForeignTable:   fixtures[tsProduct],
			ForeignColumns: []int{0},
		},
		{
			Columns:        []int{1},
			ForeignTable:   fixtures[tsOrganization],
			ForeignColumns: []int{0},
		},
	}

	fixtures[tsUserRole].ForeignKeys = []*ForeignKey{
		{
			Columns:        []int{0},
			ForeignTable:   fixtures[tsUser],
			ForeignColumns: []int{0},
		},
		{
			Columns:        []int{1},
			ForeignTable:   fixtures[tsRole],
			ForeignColumns: []int{0},
		},
		{
			Columns:        []int{2},
			ForeignTable:   fixtures[tsUser],
			ForeignColumns: []int{0},
		},
	}

	fixtures[tsUserProfile].ForeignKeys = []*ForeignKey{
		{
			Columns:        []int{0},
			ForeignTable:   fixtures[tsUser],
			ForeignColumns: []int{1},
		},
	}

	createSamples(fixtures)

	assert := require.New(t)

	assert.Equal([]int{0}, fixtures[tsProduct].sample.Columns)
	assert.Equal([]int{0, -1}, fixtures[tsProduct].sample.ColumnsMap)

	assert.Equal([]int{0, 1}, fixtures[tsUser].sample.Columns)
	assert.Equal([]int{0, 1, -1}, fixtures[tsUser].sample.ColumnsMap)

	assert.Equal([]int{0}, fixtures[tsOrganization].sample.Columns)
	assert.Equal([]int{0, -1, -1}, fixtures[tsOrganization].sample.ColumnsMap)

	assert.Equal([]int{0, 1}, fixtures[tsRole].sample.Columns)
	assert.Equal([]int{0, 1, -1}, fixtures[tsRole].sample.ColumnsMap)

	assert.Equal([]int{0, 1}, fixtures[tsUserRole].sample.Columns)
	assert.Equal([]int{0, 1, -1}, fixtures[tsUserRole].sample.ColumnsMap)

	assert.Equal([]int{0}, fixtures[tsUserProfile].sample.Columns)
	assert.Equal([]int{0, -1}, fixtures[tsUserProfile].sample.ColumnsMap)
}

func TestSample(t *testing.T) {
	r := random

	random = func(a, b int) int { return a }

	sample := Sample{
		Rows: [][]any{
			{0, 0},
			{1, 0},
			{2, 0},
			{3, 0},
		},
		DeleteMark: 2,
	}

	t.Run("selectForUpdate", func(t *testing.T) {
		require.Equal(t, []any{0, 0}, sample.selectForUpdate())
	})
	t.Run("selectForDelete", func(t *testing.T) {
		require.Equal(t, []any{2, 0}, sample.selectForDelete())
	})

	random = r
}

func TestFkGenerator(t *testing.T) {
	const (
		tsProduct      = 0
		tsUser         = iota
		tsOrganization = iota
	)

	fixtures := []*Table{
		tsProduct: {
			Name: "ts_product",
			Columns: []*Column{
				{
					Name: "code",
					IsPk: true,
					Type: DbTypeString,
				},
				{
					Name: "is_virtual",
					Type: DbTypeBool,
				},
			},
		},
		tsUser: {
			Name: "ts_user",
			Columns: []*Column{
				{
					Name: "id",
					IsPk: true,
					Auto: true,
					Type: DbTypeInt32,
				},
				{
					Name: "created_at",
					Type: DbTypeTimestamp,
				},
				{
					Name: "username",
					Type: DbTypeString,
				},
				{
					Name: "password",
					Type: DbTypeString,
				},
			},
		},
		tsOrganization: {
			Name: "ts_organization",
			Columns: []*Column{
				{
					Name: "id",
					IsPk: true,
					Auto: true,
					Type: DbTypeInt32,
				},
				{
					Name: "name",
					Type: DbTypeString,
				},
				{
					Name: "created_by",
					Type: DbTypeInt32,
				},
				{
					Name: "updated_by",
					Type: DbTypeInt32,
				},
				{
					Name: "product_id",
					Type: DbTypeString,
				},
				{
					Name: "test_fk_username",
					Type: DbTypeString,
				},
				{
					Name: "test_fk_password",
					Type: DbTypeString,
				},
			},
		},
	}

	fixtures[tsOrganization].ForeignKeys = []*ForeignKey{
		{
			Columns:        []int{2},
			ForeignTable:   fixtures[tsUser],
			ForeignColumns: []int{0},
		},
		{
			Columns:        []int{3},
			ForeignTable:   fixtures[tsUser],
			ForeignColumns: []int{0},
		},
		{
			Columns:        []int{4},
			ForeignTable:   fixtures[tsProduct],
			ForeignColumns: []int{0},
		},
		{
			Columns:        []int{5, 6},
			ForeignTable:   fixtures[tsUser],
			ForeignColumns: []int{2, 3},
		},
	}

	createSamples(fixtures)

	assert := require.New(t)

	assert.Equal([]int{0, 2, 3}, fixtures[tsUser].sample.Columns)
	assert.Equal([]int{0, -1, 1, 2}, fixtures[tsUser].sample.ColumnsMap)

	fixtures[tsUser].sample.DeleteMark = 5
	fixtures[tsUser].sample.Rows = [][]any{
		{1, "u1", "p1"},
		{2, "u2", "p2"},
		{3, "u3", "p3"},
		{4, "u4", "p4"},
		{5, "u5", "p5"},
		{6, "u6", "p6"},
		{7, "u7", "p7"},
	}

	r := random

	var dice int
	random = func(a, b int) int {
		v := dice
		dice += 1
		return v
	}

	t.Run("Single", func(t *testing.T) {
		dice = 0
		gen := fkGenerator(fixtures[tsOrganization].ForeignKeys[0], fixtures[tsUser])
			
		require.Equal(t, any(1), gen(2))
		require.Equal(t, any(2), gen(2))
		require.Equal(t, any(3), gen(2))
		require.Equal(t, any(4), gen(2))
	})

	t.Run("Composite", func(t *testing.T) {
		dice = 0
		gen := fkGenerator(fixtures[tsOrganization].ForeignKeys[3], fixtures[tsUser])
			
		require.Equal(t, any("u1"), gen(5))
		require.Equal(t, any("p1"), gen(6))
		require.Equal(t, any("u2"), gen(5))
		require.Equal(t, any("p2"), gen(6))
		require.Equal(t, any("u3"), gen(5))
		require.Equal(t, any("p3"), gen(6))
		require.Equal(t, any("u4"), gen(5))
		require.Equal(t, any("p4"), gen(6))
	})

	random = r
}

func TestGenerator(t *testing.T) {
	const (
		tsProduct      = 0
		tsUser         = iota
		tsOrganization = iota
	)

	fixtures := []*Table{
		tsProduct: {
			Name: "ts_product",
			Columns: []*Column{
				{
					Name: "code",
					IsPk: true,
					Type: DbTypeString,
				},
				{
					Name: "is_virtual",
					Type: DbTypeBool,
				},
			},
		},
		tsUser: {
			Name: "ts_user",
			Columns: []*Column{
				{
					Name: "id",
					IsPk: true,
					Auto: true,
					Type: DbTypeInt32,
				},
				{
					Name: "created_at",
					Type: DbTypeTimestamp,
				},
				{
					Name: "username",
					Type: DbTypeString,
				},
				{
					Name: "password",
					Type: DbTypeString,
				},
			},
		},
		tsOrganization: {
			Name: "ts_organization",
			Columns: []*Column{
				{
					Name: "id",
					IsPk: true,
					Auto: true,
					Type: DbTypeInt32,
				},
				{
					Name: "ecosystem_code",
					Type: DbTypeString,
					IsPk: true,
				},
				{
					Name: "name",
					Type: DbTypeString,
				},
				{
					Name: "created_by",
					Type: DbTypeInt32,
				},
				{
					Name: "updated_by",
					Type: DbTypeInt32,
				},
				{
					Name: "product_id",
					IsPk: true,
					Type: DbTypeString,
				},
				{
					Name: "test_fk_username",
					Type: DbTypeString,
				},
				{
					Name: "test_fk_password",
					Type: DbTypeString,
				},
				{
					Name: "address",
					Type: DbTypeString,
				},
			},
		},
	}

	fixtures[tsOrganization].ForeignKeys = []*ForeignKey{
		{
			Columns:        []int{3},
			ForeignTable:   fixtures[tsUser],
			ForeignColumns: []int{0},
		},
		{
			Columns:        []int{4},
			ForeignTable:   fixtures[tsUser],
			ForeignColumns: []int{0},
		},
		{
			Columns:        []int{5},
			ForeignTable:   fixtures[tsProduct],
			ForeignColumns: []int{0},
		},
		{
			Columns:        []int{6, 7},
			ForeignTable:   fixtures[tsUser],
			ForeignColumns: []int{2, 3},
		},
	}

	createSamples(fixtures)

	fixtures[tsProduct].sample.Rows = [][]any{
		{"product_1"},
		{"product_2"},
		{"product_3"},
	}
	fixtures[tsProduct].sample.DeleteMark = 3

	fixtures[tsUser].sample.Rows = [][]any{
		{1, "u1", "p1"},
		{2, "u2", "p2"},
		{3, "u3", "p3"},
		{4, "u4", "p4"},
		{5, "u5", "pj"},
		{6, "u6", "p6"},
		{7, "u7", "p7"},
	}
	fixtures[tsUser].sample.DeleteMark = 5

	fixtures[tsOrganization].sample.Rows = [][]any{
		{1, "e1", "p1"},
		{2, "e2", "p2"},
		{3, "e3", "p3"},
		{4, "e4", "p4"},
		{5, "e5", "p5"},
		{6, "e6", "p6"},
		{7, "e7", "p7"},
	}
	fixtures[tsOrganization].sample.DeleteMark = 5

	products := fixtures[tsProduct].sample.Rows
	users := fixtures[tsUser].sample.Rows
	orgs := fixtures[tsOrganization].sample.Rows

	stub := &struct{}{}

	domain := func(int) Generator { return func(i int) any { return stub } }

	var dice int
	rng := func(a, b int) int {
		v := dice
		dice += 1
		return v
	}

	r := random

	var generator *generator

	t.Run("constructor", func(t *testing.T) {
		generator = newGenerator(fixtures[tsOrganization], domain)

		assert := require.New(t)
		assert.Equal(fixtures[tsOrganization], generator.table)
		assert.Equal([]int{0, 1, 5}, generator.pkIndex)
		assert.Equal([]string{"id", "ecosystem_code", "product_id"}, generator.pkNames)
		assert.Equal([]int{1, 2, 3, 4 ,5, 6, 7, 8}, generator.columnIndex)
		assert.Equal(
			[]string{
				"ecosystem_code",
				"name",
				"created_by",
				"updated_by",
				"product_id",
				"test_fk_username",
				"test_fk_password",
				"address",
			},
			generator.columnNames,
		)

		fks := fixtures[tsOrganization].ForeignKeys
		assert.Equal(
			[]*ForeignKey{
				3: fks[0],
				4: fks[1],
				5: fks[2],
				6: fks[3],
				7: fks[3],
				8: nil,
			},
			generator.foreignKeys,
		)

		assert.NotNil(generator.factory)
	})

	t.Run("enumerateForUpdate", func(t *testing.T) {
		random = func(a, b int) int { return 0 }
		require.Equal(t, []int{2, 3, 4, 6, 7, 8}, generator.enumerateForUpdate())
	})

	t.Run("insert", func(t *testing.T) {
		dice = 0
		random = rng

		cs := Changeset{
			Table: generator.table.Name,

			Insert: Insert{
				Columns: generator.columnNames,
			},
		}

		generator.insert(&cs, generator.factory())

		assert := require.New(t)
		assert.Equal(
						[]string{
				"ecosystem_code",
				"name",
				"created_by",
				"updated_by",
				"product_id",
				"test_fk_username",
				"test_fk_password",
				"address",
			},
			cs.Insert.Columns,
		)

		assert.Equal(
			[][]any{
				{stub, stub, users[0][0], users[1][0], products[2][0], users[3][1], users[3][2], stub},
			},
			cs.Insert.Values,
		)

	})

	t.Run("update", func(t *testing.T) {
		dice = 0
		random = func(int, int) int { return 0 }

		cs := Changeset{
			Table: generator.table.Name,

			Update: Update{
				PrimaryKeys: generator.pkNames,
			},
		}

		generator.update(&cs, generator.factory())

		assert := require.New(t)

		assert.Equal(
			[][]string{
				{
					"name",
					"created_by",
					"updated_by",
					"test_fk_username",
					"test_fk_password",
					"address",
					"id",
					"ecosystem_code",
					"product_id",
				},
			},
			cs.Update.Columns,
		)

		assert.Equal(
			[][]any{
				{stub, users[0][0], users[0][0], users[0][1], users[0][2], stub, orgs[0][0], orgs[0][1], orgs[0][2]},
			},
			cs.Update.Values,
		)
	})

	t.Run("delete", func(t *testing.T) {
		random = func(a, b int) int { return a }

		cs := Changeset{
			Table: generator.table.Name,

			Delete: Delete{
				PrimaryKeys: generator.pkNames,
			},
		}

		generator.delete(&cs)

		require.Equal(t, [][]any{{6, "e6", "p6"}}, cs.Delete.Values)
	})

	random = r
}
