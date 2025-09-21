package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/urfave/cli/v2"

	"randb/pkg/domain"
	"randb/pkg/driver/pg"
	"randb/pkg/randb"
)

const (
	exitCodeSuccess = 0
	exitCodeGeneric = 1
	exitCodeBadArg  = 2
)

var conf Config

var app = cli.App{
	Name:   "randb",
	Usage:  "Generate random data in database",
	Action: wrap(run),

	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "log-level",
			EnvVars:     env("LOG_LEVEL"),
			Value:       logLevels[0],
			Usage:       fmt.Sprintf("log level, one of: %s", strings.Join(logLevels, ", ")),
			Destination: &conf.Log.Level,
		},
		&cli.StringFlag{
			Name:        "log-format",
			EnvVars:     env("LOG_FORMAT"),
			Value:       logFormats[0],
			Usage:       fmt.Sprintf("log format, one of: %s", strings.Join(logFormats, ", ")),
			Destination: &conf.Log.Format,
		},
		&cli.StringFlag{
			Name:        "database",
			EnvVars:     env("DATABASE_URL"),
			Usage:       "database URl",
			Destination: &conf.Database.URL,
		},
		&cli.StringFlag{
			Name:        "schema",
			EnvVars:     env("DATABASE_SCHEMA"),
			Value:       "public",
			Usage:       "database schema",
			Destination: &conf.Database.Schema,
		},
		&cli.IntFlag{
			Name:        "max-connections",
			EnvVars:     env("DATABASE_MAX_CONNECTIONS"),
			Value:       16,
			Usage:       "database max connections",
			Destination: &conf.Database.MaxConn,
		},
		&cli.DurationFlag{
			Name:        "query-timeout",
			EnvVars:     env("DATABASE_QUERY_TIMEOUT"),
			Value:       60 * time.Second,
			Usage:       "database query timeout",
			Destination: &conf.Database.Timeout,
		},
		&cli.IntFlag{
			Name:        "inserts",
			EnvVars:     env("INSERTS"),
			Value:       40,
			Usage:       "percent of inserts",
			Destination: &conf.Generator.Inserts,
		},
		&cli.IntFlag{
			Name:        "updates",
			EnvVars:     env("UPDATES"),
			Value:       40,
			Usage:       "percent of updates",
			Destination: &conf.Generator.Updates,
		},
		&cli.DurationFlag{
			Name:        "period",
			Value:       1 * time.Second,
			EnvVars:     env("PERIOD"),
			Usage:       "time interval between iteration",
			Destination: &conf.Generator.Period,
		},
		&cli.IntFlag{
			Name:        "batch-size",
			Value:       1000,
			EnvVars:     env("BATCH_SIZE"),
			Usage:       "table changeset size per iteration",
			Destination: &conf.Generator.BatchSize,
		},
		&cli.IntFlag{
			Name:        "iterations",
			Value:       100,
			EnvVars:     env("ITERATIONS"),
			Usage:       "number of iterations between tables scan",
			Destination: &conf.Generator.Iters,
		},
		&cli.IntFlag{
			Name:        "concurrency",
			Value:       32,
			EnvVars:     env("CONCURRENCY"),
			Usage:       "number of goroutines producing tables changes",
			Destination: &conf.Generator.Concurrency,
		},
		&cli.StringFlag{
			Name:        "include",
			Value:       "",
			EnvVars:     env("INCLUDE"),
			Usage:       "include tables pattern",
			Destination: &conf.Generator.Include,
		},
	},
}

func env(name string) []string {
	return []string{fmt.Sprintf("RANDB_%s", name)}
}

func wrap(fn cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		if err := fn(ctx); err != nil {
			if errs, ok := err.(govalidator.Errors); ok {
				fmt.Fprint(os.Stderr, "configuration errors:\n")
				for _, e := range errs {
					fmt.Fprintf(os.Stderr, "  %v\n", e)
				}
				os.Exit(exitCodeBadArg)
				return err
			}
			fmt.Fprintf(os.Stderr, "error: %v", err)
			os.Exit(exitCodeGeneric)
			return err
		}
		os.Exit(exitCodeSuccess)
		return nil
	}
}

func run(appctx *cli.Context) error {
	if err := setupLog(conf.Log); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(appctx.Context)
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		s := <-sig
		slog.Info("recevied signal", slog.Any("signal", s))
		cancel()
	}()

	driver, err := pg.New(ctx, conf.Database)
	if err != nil {
		return err
	}
	defer driver.Close()

	return randb.Run(ctx, conf.Generator, driver, domain.NewPredictor())
}

func main() {
	app.Run(os.Args)
}
