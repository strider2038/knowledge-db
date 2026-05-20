package cliapp

import (
	"context"

	"github.com/strider2038/knowledge-db/internal/bootstrap"
	"github.com/urfave/cli/v3"
)

var runServe = bootstrap.Run

func New() *cli.Command {
	return &cli.Command{
		Name:  "kb",
		Usage: "Консольная утилита для работы с базой знаний",
		Commands: []*cli.Command{
			serveCmd(),
			validateCmd(),
			initCmd(),
			dumpImagesCmd(),
			expandUrlsCmd(),
			reindexLinksCmd(),
			migrateNodeIDsCmd(),
		},
	}
}

func serveCmd() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Запустить серверную часть приложения",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runServe()
		},
	}
}
