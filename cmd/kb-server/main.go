package main

import (
	"log/slog"
	"os"

	"github.com/strider2038/knowledge-db/internal/bootstrap"
)

func main() {
	if err := bootstrap.Run(); err != nil {
		slog.Error("bootstrap", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
