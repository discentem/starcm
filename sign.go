package main

import (
	"fmt"
	"time"

	"github.com/discentem/starcm/libraries/signing"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v2"
)

// signCommand implements `starcm sign`, which writes <file>.sig next to each
// file in the format --trusted-keys and verify() check.
func signCommand() *cli.Command {
	return &cli.Command{
		Name:      "sign",
		Usage:     "sign files, writing <file>.sig next to each",
		ArgsUsage: "FILE...",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "key",
				Usage:    "PEM private key (PKCS #8), e.g. from `openssl genpkey -algorithm ed25519`",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "expires",
				Usage:    "when the signatures expire: a duration from now (`30d`, 36h), a date (2026-11-14, midnight UTC) or an RFC 3339 time",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "key-id",
				Usage: "key id to record in the signature; defaults to the key file's name before the first dot",
			},
			&cli.StringFlag{
				Name:  "release",
				Usage: "release `ID` to sign the files for; `starcm --release ID` refuses files signed for any other release",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return cli.Exit("sign: no files given", 1)
			}
			expires, err := signing.ParseExpiry(c.String("expires"), time.Now())
			if err != nil {
				return cli.Exit(fmt.Sprintf("sign: %v", err), 1)
			}
			fsys := afero.NewOsFs()
			for _, path := range c.Args().Slice() {
				sig, _, err := signing.SignFile(fsys, c.String("key"), c.String("key-id"), c.String("release"), path, expires)
				if err != nil {
					return cli.Exit(fmt.Sprintf("sign: %v", err), 1)
				}
				fmt.Println(signing.Describe(path, sig))
			}
			return nil
		},
	}
}
