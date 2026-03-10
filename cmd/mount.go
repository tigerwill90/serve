package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tigerwill90/serve/internal/client"
	"github.com/urfave/cli/v3"
)

func newMountCommand() *cli.Command {
	return &cli.Command{
		Name:      "mount",
		Usage:     "Mount a directory or file on a route",
		ArgsUsage: "<path> <route>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			if args.Len() < 2 {
				return fmt.Errorf("usage: serve mount <path> <route>")
			}

			localPath := args.Get(0)
			route := args.Get(1)

			absPath, err := filepath.Abs(localPath)
			if err != nil {
				return fmt.Errorf("invalid path: %w", err)
			}

			if _, err := os.Stat(absPath); err != nil {
				return fmt.Errorf("path not found: %w", err)
			}

			controlPort := cmd.Root().String("control-port")
			c := client.New(controlPort)

			info, err := c.Mount(absPath, route)
			if err != nil {
				return err
			}

			fmt.Printf("Mounted %s (%s) on %s\n", info.LocalPath, info.Type, info.Route)
			return nil
		},
	}
}
