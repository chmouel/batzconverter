package main

import (
	"context"
	"fmt"
	"os"

	"github.com/chmouel/batzconverter/internal/app"
	"github.com/chmouel/batzconverter/internal/command"
)

func main() {
	if err := app.Run(context.Background(), command.OSRunner{}, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "batz: %v\n", err)
		os.Exit(1)
	}
}
