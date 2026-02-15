package main

import (
	"context"
	"fmt"
	"os"

	"github.com/calvinmclean/ztg/cmd/challenge"
	"github.com/calvinmclean/ztg/cmd/identity"
	"github.com/calvinmclean/ztg/cmd/key"
	"github.com/calvinmclean/ztg/cmd/trust"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "ztg",
		Usage: "Zero Trust Gaming CLI",
		Commands: []*cli.Command{
			key.Command,
			challenge.Command,
			trust.Command,
			identity.Command,
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
