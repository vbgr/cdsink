package user

import "randb/pkg/randb"

func CreateUserGenerator(t *randb.Table) randb.Domain {
	return func(c *randb.Column) randb.Generator {
		return nil
	}
}
