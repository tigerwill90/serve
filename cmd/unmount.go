package cmd

import (
	"context"
	"fmt"

	"github.com/tigerwill90/serve/internal/client"
	"github.com/urfave/cli/v3"
)

func newUnmountCommand() *cli.Command {
	return &cli.Command{
		Name:      "unmount",
		Usage:     "Unmount a route",
		ArgsUsage: "<route>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			if args.Len() < 1 {
				return fmt.Errorf("usage: serve unmount <route>")
			}

			route := args.First()

			controlHost := cmd.Root().String("control-host")
			controlPort := cmd.Root().String("control-port")
			c := client.New(controlHost, controlPort)

			if err := c.Unmount(route); err != nil {
				return err
			}

			fmt.Printf("Unmounted %s\n", route)
			return nil
		},
	}
}
