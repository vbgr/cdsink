package generic

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/Pallinder/go-randomdata"
	"github.com/brianvoe/gofakeit/v6"

	"randb/pkg/randb"
)

type generator func(*randb.Column) randb.Generator

var generators = []generator{
	randb.DbTypeBool:        Bool,
	randb.DbTypeInt8:        I8,
	randb.DbTypeInt16:       I16,
	randb.DbTypeInt32:       I32,
	randb.DbTypeInt64:       I64,
	randb.DbTypeUInt8:       U8,
	randb.DbTypeUInt16:      U16,
	randb.DbTypeUInt32:      U32,
	randb.DbTypeUInt64:      U64,
	randb.DbTypeFloat32:     F32,
	randb.DbTypeFloat64:     F64,
	randb.DbTypeDecimal:     Decimal,
	randb.DbTypeDate:        Date,
	randb.DbTypeTime:        Time,
	randb.DbTypeTimeTz:      Time,
	randb.DbTypeTimestamp:   Timestamp,
	randb.DbTypeTimestampTz: TimestampTz,
	randb.DbTypeString:      String,
	randb.DbTypeText:        Text,
	randb.DbTypeUUID:        UUID,
	randb.DbTypeObject:      Object,
	randb.DbTypeEnum:        Enum,
	randb.DbTypeArray:       nil,  // set at init
}

func init() {
	generators[randb.DbTypeArray] = func(c *randb.Column) randb.Generator {
		if c.SubType == randb.DbTypeArray {
			panic("array of array is not supported")
		}

		return func(n int) any {
			v := make([]any, gofakeit.Number(0, 10))
			for i := range v {
				v[i] = generators[c.SubType](c)(n)
			}
			return v
		}
	}
}

func Cast[T any](f func() T) randb.Generator {
	return func(int) any { return f() }
}

func Bool(*randb.Column) randb.Generator {
	return Cast(gofakeit.Bool)
}

func I8(*randb.Column) randb.Generator {
	return Cast(gofakeit.Int8)
}

func I16(*randb.Column) randb.Generator {
	return Cast(gofakeit.Int16)
}

func I32(*randb.Column) randb.Generator {
	return Cast(gofakeit.Int32)
}

func I64(*randb.Column) randb.Generator {
	return Cast(gofakeit.Int64)
}

func U8(*randb.Column) randb.Generator {
	return Cast(gofakeit.Uint8)
}

func U16(*randb.Column) randb.Generator {
	return Cast(gofakeit.Uint16)
}

func U32(*randb.Column) randb.Generator {
	return Cast(gofakeit.Uint32)
}

func U64(*randb.Column) randb.Generator {
	return Cast(gofakeit.Uint64)
}

func F32(*randb.Column) randb.Generator {
	return Cast(gofakeit.Float32)
}

func F64(*randb.Column) randb.Generator {
	return Cast(gofakeit.Float64)
}

func Decimal(c *randb.Column) randb.Generator {
	return func(int) any {
		return randomdata.Decimal(c.Precision, c.Scale)
	}
}

func Date(c *randb.Column) randb.Generator {
	return func(int) any {
		return gofakeit.Date().Format("2006-01-02")
	}
}

func Time(*randb.Column) randb.Generator {
	return func(int) any {
		v := gofakeit.Date()
		return fmt.Sprintf("%02d-%02d-%02d", v.Hour(), v.Minute(), v.Second())
	}
}

func Timestamp(*randb.Column) randb.Generator {
	return Cast(gofakeit.Date)
}

func TimestampTz(*randb.Column) randb.Generator {
	return Cast(gofakeit.Date)
}

func String(c *randb.Column) randb.Generator {
	if c.Name == "uuid" {
		return func(int) any {
			return gofakeit.UUID()
		}
	}

	return func(int) any {
		return str(c.Size)
	}
}

func Text(*randb.Column) randb.Generator {
	return func(int) any {
		return gofakeit.Paragraph(1, 5, 20, ".")
	}
}

func Object(*randb.Column) randb.Generator {
	return Cast(gofakeit.Map)
}

func UUID(*randb.Column) randb.Generator {
	return Cast(gofakeit.UUID)
}

func Enum(c *randb.Column) randb.Generator {
	return func(int) any {
		return gofakeit.RandomString(c.Enum.Values)
	}
}

func str(n int) any {
	return strings.ToLower(randomdata.RandStringRunes(gofakeit.Number(0, n)))
}

func stub(int) any {
	return randomdata.RandStringRunes(gofakeit.Number(0, 8))
}

func Domain(t *randb.Table) randb.Domain {
	g := make([]randb.Generator, len(t.Columns))
	for i, c := range t.Columns {
		if c.Type == randb.DbTypeUnknown {
			slog.Warn("unknown column type", "table", t.Name, "column", c.Name)
			g[i] = stub
			continue
		}

		g[i] = generators[c.Type](c)
	}

	return func(int) randb.Generator {
		return func(i int) any { return g[i](i) }
	}
}
