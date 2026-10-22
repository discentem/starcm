package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	loader "github.com/discentem/starcm/libraries/loader"
	localstore "github.com/discentem/starcm/libraries/local_store"
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
				Usage: "prefix log lines with the date and time (UTC)",
			},
			&cli.IntFlag{
				Name:  "v",
				Value: 1,
				Usage: "verbosity level",
			},
			&cli.StringFlag{
				Name:  "max-expiry",
				Usage: "with --trusted-keys, also refuse signatures that expire more than `DURATION` (e.g. 30d) from now",
			},
			&cli.StringFlag{
				Name:  "release",
				Usage: "with --trusted-keys, refuse signatures not made for release `ID` (starcm sign --release)",
			},
			&cli.StringFlag{
				Name:  "trusted-keys",
				Usage: "directory of *.pem public keys; when set, every .star file must have a valid <file>.sig signed by one of them, and verify() is enabled",
			},
			&cli.StringFlag{
				Name:  "store-dir",
				Usage: "directory download() saves into when save_to is omitted, as <dir>/sha256/<hash>/<name> (default: a system directory as root/admin, else the user cache directory)",
			},
			&cli.BoolFlag{
				Name:  "store-only",
				Usage: "refuse download(save_to = ...), so every download goes into the store",
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

			// Log lines go to stderr so stdout holds only what the configuration
			// prints and commands write.
			l := log.Default()
			l.SetOutput(os.Stderr)
			// deck's logger.Init turns 0 into log.LstdFlags; Lmsgprefix is
			// non-zero but adds nothing without a date or time.
			flags := log.Lmsgprefix
			if timestamps {
				flags = log.LstdFlags | log.LUTC
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
			storeDir := c.String("store-dir")
			if storeDir == "" {
				if storeDir, err = localstore.DefaultDir(); err != nil {
					return cli.Exit(err.Error(), 1)
				}
			}
			perms, err := localstore.NewPermissions()
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			// Opened on first use, so runs that never use the store never
			// create it.
			openStore := localstore.Lazy(storeDir, perms)
			opts := []loader.LoaderOption{
				loader.WithVars(vars),
				loader.WithStore(openStore),
				loader.WithStoreOnly(c.Bool("store-only")),
				loader.WithStoreCheck(localstore.CheckIfStored(storeDir, openStore)),
				loader.WithStoreDir(storeDir),
			}
			if dir := c.String("trusted-keys"); dir != "" {
				keyring, err := signing.LoadKeyring(fsys, dir)
				if err != nil {
					return cli.Exit(err.Error(), 1)
				}
				if maxExpiry := c.String("max-expiry"); maxExpiry != "" {
					d, err := signing.ParseDuration(maxExpiry)
					if err != nil {
						return cli.Exit(fmt.Sprintf("--max-expiry: %v", err), 1)
					}
					keyring.SetMaxExpiry(d)
				}
				keyring.SetRelease(c.String("release"))
				opts = append(opts, loader.WithKeyring(keyring))
			} else if c.String("max-expiry") != "" {
				return cli.Exit("--max-expiry needs --trusted-keys", 1)
			} else if c.String("release") != "" {
				return cli.Exit("--release needs --trusted-keys", 1)
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
