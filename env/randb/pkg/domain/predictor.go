package domain

import (
	"math/rand"
	"time"

	"randb/pkg/domain/generic"
	"randb/pkg/randb"
)

type predictor int

func init() {
	rand.Seed(time.Now().Unix())
}

func (predictor) Predict(i int, tables []*randb.Table) randb.Domain {
	// TODO: add real domain prediction.
	return generic.Domain(tables[i])
}

func NewPredictor() randb.Predictor {
	return predictor(0)
}
