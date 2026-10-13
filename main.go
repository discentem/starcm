package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	loader "github.com/discentem/starcm/libraries/loader"
	"github.com/discentem/starcm/libraries/shell"
	"github.com/discentem/starcm/libraries/signing"
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
		// --var values may contain commas (URLs, lists); don't split them.
		DisableSliceFlagSeparator: true,
		Commands:                  []*cli.Command{signCommand(), inspectCommand()},
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
			&cli.StringFlag{
				Name:  "trusted-keys",
				Usage: "directory of *.pem public keys; when set, every .star file must have a valid <file>.sig signed by one of them, and verify() is enabled",
			},
			&cli.StringSliceFlag{
				Name:  "var",
				Usage: "`name=value` exposed to Starlark as vars[\"name\"] (load(\"starcm\", \"vars\")); repeatable",
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

			vars := map[string]string{}
			for _, kv := range c.StringSlice("var") {
				name, value, ok := strings.Cut(kv, "=")
				if !ok || name == "" {
					return cli.Exit(fmt.Sprintf("--var %q: want name=value", kv), 1)
				}
				vars[name] = value
			}
			opts := []loader.LoaderOption{loader.WithVars(vars)}
			if dir := c.String("trusted-keys"); dir != "" {
				keyring, err := signing.LoadKeyring(fsys, dir)
				if err != nil {
					return cli.Exit(err.Error(), 1)
				}
				opts = append(opts, loader.WithKeyring(keyring))
			}

			starcmLoader := loader.Default(
				ctx,
				fsys,
				&shell.RealExecutor{},
				wd,
				opts...,
			)

			b, err := afero.ReadFile(fsys, rootFile)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			if err := starcmLoader.CheckSignature(rootFile, b); err != nil {
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
