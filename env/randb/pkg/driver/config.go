package driver

import "time"

type Config struct {
	URL     string        `valid:"required~database URL is required,url~database URL is malformed"`
	Schema  string        `valid:"required~database schema is required"`
	MaxConn int           `valid:"range(1|256)"`
	Timeout time.Duration `valud:"-"`
}
