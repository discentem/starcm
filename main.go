package main

import (
	"context"
	"flag"
	"log"
	"path/filepath"

	load "github.com/discentem/starcm/internal/loading"
	"github.com/spf13/afero"

	"github.com/discentem/starcm/libraries/shell"
	"github.com/google/deck"
	"github.com/google/deck/backends/logger"
)

func run(rootFile string, verbosity int, timestamps bool, fsys afero.Fs, loggers []deck.Backend) error {

	l := log.Default()
	if timestamps {
		l.SetFlags(log.LUTC)
	}
	deck.Add(logger.Init(l.Writer(), l.Flags()))

	for _, l := range loggers {
		deck.Add(l)
	}

	deck.Info("starting starcm...")
	deck.SetVerbosity(verbosity)

	ctx := context.Background()

	loader := load.DefaultLoader(
		ctx,
		fsys,
		&shell.RealExecutor{},
		filepath.Dir(rootFile),
	)

	b, err := afero.ReadFile(fsys, rootFile)
	if err != nil {
		return err
	}

	return load.FromFile(
		context.Background(),
		rootFile,
		// If src is bytes, starlark-go will just execute it directly
		// without any additional processing.
		// https://github.com/google/starlark-go/blob/42030a7cedcee8b1fe3dc9309d4f545f6104715d/syntax/scan.go#L282
		b,
		loader.Sequential(context.Background()),
	)
}

func main() {
	f := flag.String(
		"root_file",
		"",
		"path to the first starlark file to run",
	)
	timestamps := flag.Bool("timestamps", true, "include timestamps in logs")
	verbosity := flag.Int("v", 1, "verbosity level")
	inmemfs := flag.Bool("inmem_fs", false, "use in-memory filesystem")
	flag.Parse()

	l := log.Default()
	if !*timestamps {
		l.SetFlags(log.LUTC)
	}
	fsys := afero.Fs(nil)
	if *inmemfs {
		fsys = afero.NewMemMapFs()
	} else {
		fsys = afero.NewOsFs()
	}

	err := run(*f, *verbosity, *timestamps, fsys, []deck.Backend{logger.Init(l.Writer(), l.Flags())})
	if err != nil {
		log.Fatal(err)
	}
}
