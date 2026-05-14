package main

import (
	"context"
	"os"

	"github.com/strider2038/knowledge-db/internal/cliapp"
)

func main() {
	app := cliapp.New()
	if err := app.RunContext(context.Background(), os.Args); err != nil {
		os.Exit(1)
	}
}
