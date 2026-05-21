package cliapp

import (
	"context"
	"fmt"
	"os"

	"github.com/muonsoft/errors"
	"github.com/urfave/cli/v3"

	"github.com/strider2038/knowledge-db/internal/bootstrap"
)

func rebuildIndexCmd() *cli.Command {
	return &cli.Command{
		Name:  "rebuild-index",
		Usage: "Полная перестройка embedding-index (.kb/index.db)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Aliases: []string{"p"},
				Usage:   "путь к базе знаний (по умолчанию KB_DATA_PATH или ./data)",
			},
		},
		Action: runRebuildIndex,
	}
}

func runRebuildIndex(ctx context.Context, cmd *cli.Command) error {
	path := cmd.String("path")
	if path == "" {
		path = os.Getenv("KB_DATA_PATH")
	}
	if path == "" {
		path = "./data"
	}
	basePath, err := absPath(path)
	if err != nil {
		return errors.Errorf("rebuild-index: %w", err)
	}

	fmt.Fprintf(os.Stderr, "rebuild-index: starting (data_path=%s)\n", basePath)

	result, err := bootstrap.RunIndexRebuild(ctx, basePath)
	if err != nil {
		return errors.Errorf("rebuild-index: %w", err)
	}

	if result != nil && result.Status != nil {
		s := result.Status
		fmt.Printf(
			"rebuild-index: done (nodes=%d chunks=%d model=%s keyword_index=%s status=%s)\n",
			s.TotalNodes,
			s.TotalChunks,
			s.EmbeddingModel,
			s.KeywordIndex,
			s.Status,
		)
	} else {
		fmt.Println("rebuild-index: done")
	}

	return nil
}
