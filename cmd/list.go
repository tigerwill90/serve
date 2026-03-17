package cmd

import (
	"context"
	"fmt"
	"text/tabwriter"

	"github.com/tigerwill90/serve/internal/client"
	"github.com/urfave/cli/v3"
)

func newListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all active mounts",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			controlHost := cmd.Root().String("control-host")
			controlPort := cmd.Root().String("control-port")
			c := client.New(controlHost, controlPort)

			mounts, err := c.List()
			if err != nil {
				return err
			}

			if len(mounts) == 0 {
				fmt.Println("No active mounts")
				return nil
			}

			w := tabwriter.NewWriter(cmd.Root().Writer, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "ROUTE\tLOCAL PATH\tTYPE")
			for _, m := range mounts {
				fmt.Fprintf(w, "%s\t%s\t%s\n", m.Route, m.LocalPath, m.Type)
			}
			return w.Flush()
		},
	}
}
