package main

import (
	"context"
	"os"

	"github.com/strider2038/knowledge-db/internal/cliapp"
)

func main() {
	os.Exit(cliapp.Execute(context.Background(), os.Args, os.Stderr))
}
