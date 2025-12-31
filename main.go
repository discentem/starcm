package main

import (
	"context"
	"log"
	"os"

	loader "github.com/discentem/starcm/libraries/loader"
	"github.com/discentem/starcm/libraries/shell"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v2"

	"github.com/google/deck"
	"github.com/google/deck/backends/logger"
)

func main() {
	app := &cli.App{
		Name:  "starcm",
		Usage: "A configuration management language using Starlark",
		Args:  true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "timestamps",
				Usage: "include timestamps in logs",
			},
			&cli.IntFlag{
				Name:  "v",
				Value: 1,
				Usage: "verbosity level",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return cli.ShowAppHelp(c)
			}

			rootFile := c.Args().First()
			timestamps := c.Bool("timestamps")
			verbosity := c.Int("v")

			l := log.Default()
			l.SetOutput(os.Stdout)
			flags := log.LstdFlags
			if timestamps {
				flags |= log.LUTC
			}
			deck.Add(logger.Init(l.Writer(), flags))
			deck.SetVerbosity(verbosity)

			ctx := context.Background()

			fsys := afero.NewOsFs()

			wd, err := os.Getwd()
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			starcmLoader := loader.Default(
				ctx,
				fsys,
				&shell.RealExecutor{},
				wd,
			)

			b, err := afero.ReadFile(fsys, rootFile)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			err = loader.LoadFromFile(
				context.Background(),
				rootFile,
				b,
				starcmLoader.Sequential(context.Background()),
			)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
