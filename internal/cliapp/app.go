package cliapp

import (
	"context"
	"fmt"
	"io"

	"github.com/strider2038/knowledge-db/internal/bootstrap"
	"github.com/urfave/cli/v3"
)

var runServe = bootstrap.Run

// Execute запускает CLI. При ошибке пишет её в stderr и возвращает код 1.
// urfave/cli печатает только ExitCoder, а обычные ошибки старта (в том числе
// валидация OpenCode) иначе теряются — docker logs остаётся пустым.
func Execute(ctx context.Context, args []string, stderr io.Writer) int {
	if err := New().Run(ctx, args); err != nil {
		fmt.Fprintln(stderr, err)

		return 1
	}

	return 0
}

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
			rebuildIndexCmd(),
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
