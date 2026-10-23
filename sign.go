package main

import (
	"fmt"
	"path/filepath"
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
		Usage:     "sign files, writing <file>.sig next to each; or, with --dir, a folder's manifest",
		ArgsUsage: "FILE... | --dir DIR",
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
				Name:  "dir",
				Usage: "write `DIR`/" + signing.ManifestName + ", listing every file in DIR with its sha256, and sign it, for download() and load_remote() of the folder",
			},
			&cli.StringFlag{
				Name:  "release",
				Usage: "release `ID` to sign the files for; `starcm --release ID` refuses files signed for any other release",
			},
		},
		Action: func(c *cli.Context) error {
			dir := c.String("dir")
			if dir != "" && c.NArg() > 0 {
				return cli.Exit("sign: pass files or --dir, not both", 1)
			}
			if dir == "" && c.NArg() == 0 {
				return cli.Exit("sign: no files given", 1)
			}
			expires, err := signing.ParseExpiry(c.String("expires"), time.Now())
			if err != nil {
				return cli.Exit(fmt.Sprintf("sign: %v", err), 1)
			}
			fsys := afero.NewOsFs()
			if dir != "" {
				m, sig, _, err := signing.SignDir(fsys, c.String("key"), c.String("key-id"), c.String("release"), dir, expires)
				if err != nil {
					return cli.Exit(fmt.Sprintf("sign: %v", err), 1)
				}
				manifestPath := filepath.Join(dir, signing.ManifestName)
				files := "files"
				if len(m.Files) == 1 {
					files = "file"
				}
				fmt.Printf("wrote %s listing %d %s\n", manifestPath, len(m.Files), files)
				fmt.Println(signing.Describe(manifestPath, sig))
				return nil
			}
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
