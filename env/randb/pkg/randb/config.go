package randb

import (
	"errors"
	"time"

	"github.com/asaskevich/govalidator"
)

type Config struct {
	Inserts     int           `valid:"range(0|100)"`
	Updates     int           `valid:"range(0|100)"`
	BatchSize   int           `valid:"range(1|100000)"`
	Concurrency int           `valid:"range(1|128)"`
	Iters       int           `valid:"range(1|1000000)"`
	Period      time.Duration `valid:"range(0|100)"`
	Include     string        `valid:"-"`
}

func (c Config) Validate() error {
	var errs []error
	if _, err := govalidator.ValidateStruct(&c); err != nil {
		errs = append(errs, err.(govalidator.Errors)...)
	}

	if c.Inserts+c.Updates > 100 {
		name := "Inserts + Updates"
		errs = append(errs, govalidator.Error{Name: name, Err: errors.New("must be <= 100")})
	}

	if len(errs) > 0 {
		return govalidator.Errors(errs)
	}

	return nil
}
