package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/discentem/starcm/libraries/inspect"
	"github.com/urfave/cli/v2"
)

// inspectCommand implements `starcm inspect`, which prints what each file
// loads and calls, as JSON, without running it.
func inspectCommand() *cli.Command {
	return &cli.Command{
		Name:      "inspect",
		Usage:     "print what .star files load and call, as JSON, without running them",
		ArgsUsage: "FILE...",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return cli.Exit("inspect: no files given", 1)
			}
			reports := []*inspect.Report{}
			for _, path := range c.Args().Slice() {
				src, err := os.ReadFile(path)
				if err != nil {
					return cli.Exit(fmt.Sprintf("inspect: %v", err), 1)
				}
				r, err := inspect.File(path, src)
				if err != nil {
					return cli.Exit(fmt.Sprintf("inspect: %v", err), 1)
				}
				reports = append(reports, r)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(reports)
		},
	}
}
