package cmd

import (
	"context"

	"github.com/urfave/cli/v3"
)

func Execute(ctx context.Context, args []string) error {
	app := &cli.Command{
		Name:  "serve",
		Usage: "Dynamic file server with runtime mount support",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "control-host",
				Usage: "Host for the control API",
				Value: "127.0.0.1",
			},
			&cli.StringFlag{
				Name:  "control-port",
				Usage: "Port for the control API",
				Value: "8081",
			},
		},
		Commands: []*cli.Command{
			newStartCommand(),
			newMountCommand(),
			newUnmountCommand(),
			newListCommand(),
		},
	}

	return app.Run(ctx, args)
}
