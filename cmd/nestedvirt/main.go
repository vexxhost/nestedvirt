package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/vexxhost/nestedvirt/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cli.SetVersion(cli.Version{
		Version: version,
		Commit:  commit,
		Date:    date,
	})

	os.Exit(cli.Run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
