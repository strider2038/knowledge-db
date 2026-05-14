package cliapp

import (
	"context"
	"fmt"
	"os"

	"github.com/strider2038/knowledge-db/internal/kb"
	"github.com/urfave/cli/v2"
)

func validateCmd() *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Проверить структуру базы знаний",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Aliases: []string{"p"},
				Usage:   "путь к базе знаний (по умолчанию текущая директория)",
			},
		},
		Action: func(cCtx *cli.Context) error {
			path := cCtx.String("path")
			if path == "" {
				path = "."
			}
			path, err := absPath(path)
			if err != nil {
				return err
			}
			violations, err := kb.Validate(context.Background(), path)
			if err != nil {
				return fmt.Errorf("validate: %w", err)
			}
			if len(violations) > 0 {
				for _, v := range violations {
					fmt.Fprintf(os.Stderr, "  %s: %s\n", v.Path, v.Message)
				}
				os.Exit(1)
			}
			fmt.Println("OK: структура базы валидна")

			return nil
		},
	}
}
