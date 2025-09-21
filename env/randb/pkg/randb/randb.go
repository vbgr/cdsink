package randb

import (
	"context"
	"log/slog"
	"regexp"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

// Scanner scans table schema and sample rows.
type Scanner interface {
	Scan() ([]*Table, error)
}

// Closer defines a closable entity.
type Closer interface {
	Close() error
}

// Selector defines a rows set selector.
type Selector interface {
	Select(Chunker, string, []string) ([][]any, error)
}

// Applier applies changeset to a table.
type Applyer interface {
	Apply(Changeset) error
}

// Driver defines DB adapter interface required to be used by randb main process.
type Driver interface {
	Closer
	Scanner
	Selector
	Applyer
}

// Run starts DB population process.
func Run(ctx context.Context, conf Config, driver Driver, predictor Predictor) error {
	if err := conf.Validate(); err != nil {
		return err
	}

	slog.Info("starting randb with configuration", "conf", conf)

	include, err := regexp.Compile(conf.Include)
	if err != nil {
		return err
	}

	rng := engine{
		conf:      conf,
		driver:    driver,
		predictor: predictor,
		include:   include,

		numInserts: int(float32(conf.BatchSize) * float32(conf.Inserts) / float32(100)),
		numUpdates: int(float32(conf.BatchSize) * float32(conf.Updates) / float32(100)),
		numDeletes: int(float32(conf.BatchSize) * float32(100-conf.Inserts-conf.Updates) / float32(100)),

		delWatermark: float32(conf.Inserts+conf.Updates) / float32(100),
	}

	ticker := time.NewTicker(conf.Period)
	func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("terminating")
				return
			case <-ticker.C:
				rng.do()
			}
		}
	}()

	return nil
}

type engine struct {
	conf      Config
	driver    Driver
	predictor Predictor
	iter      int
	gens      []*generator
	include   *regexp.Regexp

	numInserts int
	numUpdates int
	numDeletes int

	delWatermark float32
}

func selectSamples(driver Driver, delWatermark float32, dbTabs []*Table) {
	var wg sync.WaitGroup

	wg.Add(len(dbTabs))
	for _, t := range dbTabs {
		go func(t *Table) {
			defer wg.Done()

			rows, err := driver.Select(chunker(0), t.Name, t.getSampleColumns())
			if err != nil {
				slog.Error("failed to select samples for table", "table", t.Name, "err", err)
				return
			}

			gofakeit.ShuffleAnySlice(rows)

			t.sample.Rows = rows
			t.sample.DeleteMark = int(delWatermark * float32(len(t.sample.Rows)))
		}(t)
	}

	wg.Wait()
}

func craeteGenerators(include *regexp.Regexp, predictor Predictor, dbTabs []*Table) []*generator {
	gens := make([]*generator, 0, len(dbTabs))
	for i, t := range dbTabs {
		if !include.MatchString(t.Name) {
			slog.Info("skip table because it does not match include pattern", "table", t.Name)
			continue
		}

		switch t.status() {
		case tableStatusFkNotScanned:
			slog.Warn("skip table because not all dependencies were scanned", "table", t.Name)
			continue
		case tableStatusFkNoData:
			slog.Warn("skip table because not all required dependencies have data", "table", t.Name)
			continue
		}

		domain := predictor.Predict(i, dbTabs)
		gens = append(gens, newGenerator(t, domain))
	}
	return gens
}

func (g *engine) reset() error {
	slog.Info("scanning tables")

	dbTabs, err := g.driver.Scan()
	if err != nil {
		slog.Error("failed to scan tables", "error", err)
		return err
	}

	createSamples(dbTabs)
	selectSamples(g.driver, g.delWatermark, dbTabs)

	g.gens = craeteGenerators(g.include, g.predictor, dbTabs)
	g.iter = 0

	slog.Info("scanning tables completed")
	return nil
}

func (g *engine) do() error {
	slog.Info("new iteration")

	if g.iter == 0 || g.iter > g.conf.Iters {
		if err := g.reset(); err != nil {
			return err
		}
	}

	g.iter += 1

	if len(g.gens) == 0 {
		slog.Info("no tables found")
		g.iter += g.conf.Iters
		return nil
	}

	sem := make(chan bool, g.conf.Concurrency)

	var wg sync.WaitGroup
	wg.Add(len(g.gens))

	for i := 0; i < len(g.gens); i++ {
		go func(n int) {
			sem <- true

			slog.Info("generating changeset", "table", g.gens[n].table.Name)
			cs := g.gens[n].generate(g.conf)

			slog.Info("applying changeset", "table", g.gens[n].table.Name)
			// TODO: add error handling
			g.driver.Apply(cs)

			<-sem
			wg.Done()
		}(i)
	}

	wg.Wait()
	close(sem)

	return nil
}
