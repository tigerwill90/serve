package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/tigerwill90/serve/internal/server"
	"github.com/urfave/cli/v3"
)

func newStartCommand() *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "Start the file server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "host",
				Usage: "Host to bind to",
				Value: "127.0.0.1",
			},
			&cli.StringFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "Port for serving files",
				Value:   "8080",
			},
			&cli.StringFlag{
				Name:  "control-host",
				Usage: "Host to bind the control API to",
				Value: "127.0.0.1",
			},
			&cli.StringFlag{
				Name:  "control-port",
				Usage: "Port for the control API",
				Value: "8081",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			srv, err := server.New(server.Config{
				Host:        cmd.String("host"),
				Port:        cmd.String("port"),
				ControlHost: cmd.String("control-host"),
				ControlPort: cmd.String("control-port"),
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return err
			}
			return srv.Run()
		},
	}
}
