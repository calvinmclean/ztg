package main

import (
	"context"
	"fmt"
	"os"

	"ztg/cmd/challenge"
	"ztg/cmd/key"
	"ztg/cmd/server"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "ztg",
		Usage: "Zero Trust Gaming CLI",
		Commands: []*cli.Command{
			key.Command,
			challenge.Command,
			server.Command,
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
