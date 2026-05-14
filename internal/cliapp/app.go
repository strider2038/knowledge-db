package cliapp

import (
	"github.com/strider2038/knowledge-db/internal/bootstrap"
	"github.com/urfave/cli/v2"
)

var runServe = bootstrap.Run

func New() *cli.App {
	return &cli.App{
		Name:  "kb",
		Usage: "Консольная утилита для работы с базой знаний",
		Commands: []*cli.Command{
			serveCmd(),
			validateCmd(),
			initCmd(),
			dumpImagesCmd(),
			expandUrlsCmd(),
			reindexLinksCmd(),
		},
	}
}

func serveCmd() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Запустить серверную часть приложения",
		Action: func(cCtx *cli.Context) error {
			return runServe()
		},
	}
}
