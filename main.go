package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tigerwill90/serve/cmd"
)

func main() {
	if err := cmd.Execute(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
