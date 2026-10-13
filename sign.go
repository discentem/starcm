package main

import (
	"fmt"
	"os"

	"github.com/discentem/starcm/libraries/signing"
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
				Name:  "key-id",
				Usage: "key id to record in the signature; defaults to the key file's name before the first dot",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return cli.Exit("sign: no files given", 1)
			}
			keyPEM, err := os.ReadFile(c.String("key"))
			if err != nil {
				return cli.Exit(fmt.Sprintf("sign: %v", err), 1)
			}
			key, err := signing.ParsePrivateKey(keyPEM)
			if err != nil {
				return cli.Exit(fmt.Sprintf("sign: %s: %v", c.String("key"), err), 1)
			}
			keyID := c.String("key-id")
			if keyID == "" {
				keyID = signing.KeyIDFromPath(c.String("key"))
			}
			for _, path := range c.Args().Slice() {
				data, err := os.ReadFile(path)
				if err != nil {
					return cli.Exit(fmt.Sprintf("sign: %v", err), 1)
				}
				sig, err := signing.Sign(key, keyID, data)
				if err != nil {
					return cli.Exit(fmt.Sprintf("sign: %v", err), 1)
				}
				sigPath := path + signing.SignatureSuffix
				if err := os.WriteFile(sigPath, sig.Bytes(), 0o644); err != nil {
					return cli.Exit(fmt.Sprintf("sign: %v", err), 1)
				}
				fmt.Printf("signed %s with key %q -> %s\n", path, keyID, sigPath)
			}
			return nil
		},
	}
}
