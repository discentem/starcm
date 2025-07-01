package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	loader "github.com/discentem/starcm/libraries/loader"
	"github.com/discentem/starcm/libraries/shell"
	"github.com/spf13/afero"

	"github.com/google/deck"
	"github.com/google/deck/backends/logger"
)

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
	l.SetOutput(os.Stdout)
	if !*timestamps {
		l.SetFlags(log.LUTC)
	}
	deck.Add(logger.Init(l.Writer(), l.Flags()))

	deck.SetVerbosity(*verbosity)

	ctx := context.Background()

	fsys := afero.Fs(nil)

	if *inmemfs {
		fsys = afero.NewMemMapFs()
	} else {
		fsys = afero.NewOsFs()
	}

	starcmLoader := loader.Default(
		ctx,
		fsys,
		&shell.RealExecutor{},
		filepath.Dir(*f),
	)

	b, err := afero.ReadFile(fsys, *f)
	if err != nil {
		log.Fatal(err)
	}

	err = loader.LoadFromFile(
		context.Background(),
		*f,
		// If src is bytes, starlark-go will just execute it directly
		// without any additional processing.
		// https://github.com/google/starlark-go/blob/42030a7cedcee8b1fe3dc9309d4f545f6104715d/syntax/scan.go#L282
		b,
		starcmLoader.Sequential(context.Background()),
	)
	if err != nil {
		log.Fatal(err)
	}

}
